package main

import (
	"agregator/adapters/models"
	"agregator/adapters/ozon"
	"agregator/adapters/wb"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

/* Программа реализует веб-сервер на Go для поиска товаров на маркетплейсах Ozon и Wildberries
с использованием кэша Redis и возможностью fallback-парсинга через Python скрипт.

Основные компоненты программы:

1. Подключение к Redis:
   - Через функцию initRedis() создаётся клиент Redis, который подключается к локальному серверу на порту 6379.
   - Пытается подключиться несколько раз с интервалом в 2 секунды.
   - Если подключение не удалось, программа продолжает работать, но без кэширования.

2. Нормализация поискового запроса:
   - Функция normalizeQuery() убирает лишние пробелы, приводит строку к нижнему регистру.
   - Это позволяет одинаково обрабатывать запросы вроде " IPHONE  12 " и "iphone 12".

3. Работа с Redis:
   - saveToRedis() сохраняет список товаров в Redis в виде JSON на 1 час по нормализованному ключу запроса.
   - getFromRedis() пытается получить список товаров из Redis. Если ключа нет — возвращает ошибку.

4. Парсинг товаров (searchProducts):
   - Сначала проверяется, есть ли данные в Redis. Если есть — возвращаются они.
   - Если в кэше нет, данные собираются параллельно с Ozon и Wildberries с использованием goroutine и sync.WaitGroup.

   - Ozon:
     - Основной парсер: ozon.Parse(query).
     - Если основной парсер падает, выполняется fallback через Python скрипт (fallback.py).
     - Python скрипт возвращает JSON, который десериализуется в []models.Product.

   - Wildberries:
     - Парсер wb.Parse(query), ошибки логируются.

   - После получения товаров:
     - Объединяются результаты с Ozon и WB.
     - Сортируются по цене (DiscountPrice) по возрастанию.
     - Сохраняются в Redis для ускорения последующих запросов.

5. HTTP обработчик searchHandler:
   - Принимает GET-запрос на эндпоинт /search с параметром query.
   - Нормализует query, вызывает searchProducts().
   - Возвращает JSON с найденными товарами.
   - В случае ошибки возвращает HTTP 400 (пустой или некорректный query) или 500 (ошибка поиска или парсинга).
*/

var (
	ctx         = context.Background()
	redisClient *redis.Client
)

func initRedis() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	for i := 0; i <= 3; i++ {
		_, err := redisClient.Ping(ctx).Result()
		if err == nil {
			log.Println("Подключено к Redis")
			return
		}
		log.Printf("Попытка %d подключиться к Redis не удалась:%v", i+1, err)
		time.Sleep(time.Second * 2)
	}
	log.Println("Работа без кэширования")
}

func normalizeQuery(q string) string {
	q = strings.TrimSpace(q)
	q = strings.ToLower(q)

	re := regexp.MustCompile(`\s+`)
	q = re.ReplaceAllString(q, " ")
	return q
}

func saveToRedis(query string, products []models.Product) error {
	data, err := json.Marshal(products)
	if err != nil {
		return err
	}
	query = normalizeQuery(query)
	err = redisClient.Set(ctx, query, data, 1*time.Hour).Err()
	return err
}

func getFromRedis(query string) ([]models.Product, error) {
	query = normalizeQuery(query)
	val, err := redisClient.Get(ctx, query).Result()
	if err == redis.Nil {
		return nil, errors.New("Ключа нет в redis")
	} else if err != nil {
		return nil, fmt.Errorf("Ошибка Redis:%w", err)
	}
	var products []models.Product
	err = json.Unmarshal([]byte(val), &products)
	return products, err
}

func searchProducts(query string) ([]models.Product, error) {
	query = normalizeQuery(query)
	cachedProducts, err := getFromRedis(query)
	if err == nil {
		fmt.Printf("Использован кэш для запроса %s\n", query)
		return cachedProducts, nil
	}

	var (
		wg        sync.WaitGroup
		itemsOzon []models.Product
		itemWB    []models.Product
		errOzon   error
		errWB     error
	)
	wg.Add(2)

	go func() {
		defer wg.Done()
		itemsOzon, errOzon = ozon.Parse(query)
		if errOzon != nil {
			fmt.Println("[OZON] стартую фолбэк")
			cmd := exec.Command("python3", "./adapters/ozon/fallback.py", query)
			output, err := cmd.Output()
			if err != nil {
				errOzon = fmt.Errorf("[OZON] Ошибка запуска скрипта:%w", err)
			}
			if len(output) == 0 {
				errOzon = fmt.Errorf("[OZON] Скрипт вернул пустой вывод")
			}
			err = json.Unmarshal(output, &itemsOzon)
			if err != nil {
				errOzon = fmt.Errorf("[OZON] Не удалось десериализовать %w", err)
			}
			fmt.Printf("[OZON] Python фолбэк вернул %d товаров:", len(itemsOzon))
		}
	}()

	go func() {
		defer wg.Done()
		itemWB, errWB = wb.Parse(query)
		if errWB != nil {
			errWB = fmt.Errorf("[WB] Не удалось спарсить wb:%w", errWB)
		}
	}()

	wg.Wait()

	if errOzon != nil && errWB != nil {
		return nil, fmt.Errorf("ошибка парсинга: ozon=%v wildberries=%v", errOzon, errWB)
	}

	items := append(itemsOzon, itemWB...)
	if len(items) == 0 {
		return nil, errors.New("товары не найдены")
	}

	sort.Slice(items, func(i, j int) bool {
		cleanedI := strings.ReplaceAll(items[i].DiscountPrice, " ", "")
		cleanedI = strings.ReplaceAll(items[i].DiscountPrice, "₽", "")
		cleanedJ := strings.ReplaceAll(items[j].DiscountPrice, " ", "")
		cleanedJ = strings.ReplaceAll(items[j].DiscountPrice, "₽", "")
		pi, _ := strconv.Atoi(cleanedI)
		pj, _ := strconv.Atoi(cleanedJ)
		return pi < pj
	})

	err = saveToRedis(query, items)
	if err != nil {
		fmt.Printf("не удалось сохранить в Redis: %v", err)
	}

	return items, nil
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	query = normalizeQuery(query)
	if query == "" {
		http.Error(w, "Параметр query обязателен", http.StatusBadRequest)
		return
	}

	products, err := searchProducts(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(products)
	if err != nil {
		http.Error(w, "ошибка кодирования ответа", http.StatusInternalServerError)
	}
}

func main() {
	initRedis()
	http.HandleFunc("/search", searchHandler)
	http.Handle("/", http.FileServer(http.Dir("web/dist")))
	fmt.Println("Сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
