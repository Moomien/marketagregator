package wb

import (
	"agregator/adapters/models"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func setHeaders(req *http.Request, referer string) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.6045.105 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Origin", "https://www.wildberries.ru")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
}

func warmUp(client *http.Client) error {
	req, err := http.NewRequest("GET", "https://www.wildberries.ru/", nil)
	if err != nil {
		return err
	}
	setHeaders(req, "")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return nil
}

func wildberries(query string) ([]byte, error) {
	apiUrl := "https://search.wb.ru/exactmatch/ru/common/v18/search?appType=1&curr=rub&dest=-1257786&lang=ru&page=1&query=" + url.QueryEscape(query) + "&resultset=catalog&sort=priceup&spp=30"
	referer := "https://www.wildberries.ru/catalog/0/search.aspx?search=" + url.QueryEscape(query)

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("[WB] ошибка cookiejar:%w", err)
	}

	wd, _ := os.Getwd()
	path := filepath.Join(wd, "proxy.txt")
	dat, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot read file proxy.txt: %v", err)
	}
	var transport *http.Transport
	proxyStr := strings.TrimSpace(string(dat))
	if proxyStr != "" {
		proxyURL, _ := url.Parse(string(dat))
		transport = &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: jar,
	}

	if err := warmUp(client); err != nil {
		return nil, fmt.Errorf("[WB] Warmup errors:%w", err)
	}

	for attempt := 0; attempt <= 30; attempt++ {
		fmt.Println("[WB] STEP:", attempt)
		fmt.Println("[WB] URL:", apiUrl)
		req, err := http.NewRequest("GET", apiUrl, nil)
		if err != nil {
			fmt.Println("[WB] GET request err:", err)
			continue
		}
		setHeaders(req, referer)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("[WB] Client error:", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("[WB] ошибка чтения resp.body")
			continue
		}

		if resp.StatusCode == 200 {
			fmt.Println("WB RESPONSE OK:\n", resp.Status)
			return body, nil
		}
		resp.Body.Close()
		time.Sleep(time.Second * 2)
		if len(body) > 0 {
			s := string(body)
			if len(s) > 500 {
				s = s[:1000]
			}
			fmt.Println("[WB] resp status code != 200. last body snippet:", s)
		}
		fmt.Println("WB unexpected status:", resp.Status)
		resp.Body.Close()
	}
	return nil, errors.New("[WB]  не удалось получить данные")
}

func Parse(query string) ([]models.Product, error) {
	body, err := wildberries(query)
	if err != nil {
		return nil, fmt.Errorf("[WB] ошибка сбора json WB:%w", err)
	}
	root := gjson.ParseBytes(body)
	products := root.Get("products")
	if !products.Exists() || !products.IsArray() {
		return nil, errors.New("[WB] items not found or not array")
	}

	var items []models.Product
	products.ForEach(func(_, products gjson.Result) bool {
		id := products.Get("id").Int()
		link := "https://www.wildberries.ru/catalog/" + strconv.FormatInt(id, 10) + "/detail.aspx"
		name := products.Get("name").String()
		discprice := (products.Get("sizes").Get("0").Get("price").Get("product").Int()) / 100
		baseprice := (products.Get("sizes").Get("0").Get("price").Get("basic").Int()) / 100
		stars := products.Get("rating").String()
		reviews := products.Get("feedbacks").String()
		var statistic string
		if stars != "" && reviews != "" {
			statistic = stars + " • " + reviews
		}
		fmt.Println("[WB] Пытаюсь собрать линку для изображения")
		vol := id / 100000
		part := id / 1000
		host := ""
		switch {
		case vol >= 0 && vol <= 143:
			host = "01"
		case vol <= 287:
			host = "02"
		case vol <= 431:
			host = "03"
		case vol <= 719:
			host = "04"
		case vol <= 1007:
			host = "05"
		case vol <= 1061:
			host = "06"
		case vol <= 1115:
			host = "07"
		case vol <= 1169:
			host = "08"
		case vol <= 1313:
			host = "09"
		case vol <= 1601:
			host = "10"
		case vol <= 1655:
			host = "11"
		case vol <= 1919:
			host = "12"
		case vol <= 2045:
			host = "13"
		case vol <= 2189:
			host = "14"
		case vol <= 2405:
			host = "15"
		case vol <= 2621:
			host = "16"
		case vol <= 2837:
			host = "17"
		case vol <= 3053:
			host = "18"
		case vol <= 3269:
			host = "19"
		case vol <= 3485:
			host = "20"
		case vol <= 3701:
			host = "21"
		case vol <= 3917:
			host = "22"
		case vol <= 4133:
			host = "23"
		case vol <= 4349:
			host = "24"
		case vol <= 4565:
			host = "25"
		case vol <= 4877:
			host = "26"
		case vol <= 5189:
			host = "27"
		case vol <= 5501:
			host = "28"
		case vol <= 5813:
			host = "29"
		case vol <= 6125:
			host = "30"
		case vol <= 6437:
			host = "31"
		case vol <= 6749:
			host = "32"
		case vol <= 7061:
			host = "33"
		case vol <= 7373:
			host = "34"
		case vol <= 7685:
			host = "35"
		case vol <= 7997:
			host = "36"
		case vol <= 8309:
			host = "37"
		default:
			host = "38"
		}

		pic := fmt.Sprintf(
			"https://basket-%s.wbbasket.ru/vol%d/part%d/%d/images/big/1.webp",
			host, vol, part, id)

		resp, err := http.Get(pic)
		fmt.Println("[WB] послал запрос к изображению")
		if err != nil {
			fmt.Println("[WB] ошибка запроса к изображению:", err)
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
		} else {
			fmt.Println("[WB] успех запроса к изображению")
		}

		p := models.Product{
			Link:             link,
			IMG:              pic,
			ProductID:        strconv.Itoa(int(id)),
			ProductName:      name,
			DiscountPrice:    strconv.Itoa(int(discprice)),
			BasePrice:        strconv.Itoa(int(baseprice)),
			ProductStatistic: statistic,
			ProductStars:     stars,
			ProductReviews:   reviews,
		}
		items = append(items, p)
		return true
	})
	fmt.Println("[WB] items:", len(products.Array()))
	return items, nil
}
