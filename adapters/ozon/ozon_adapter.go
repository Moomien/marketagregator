package ozon

import (
	"agregator/adapters/models"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func setHeaders(req *http.Request, referer string) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Origin", "https://www.ozon.ru")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	req.Header.Set("Sec-Ch-Ua", `"Not_A Brand";v="99", "Google Chrome";v="119", "Chromium";v="119"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", "Windows")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Dnt", "1")

}

func warmUp(client *http.Client) error {
	req, err := http.NewRequest("GET", "https://www.ozon.ru/", nil)
	if err != nil {
		return err
	}
	setHeaders(req, "")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return nil
}

func loadCookies(jar *cookiejar.Jar) error {
	wd, _ := os.Getwd()
	path := filepath.Join(wd, "cookies.json")
	dat, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Cannot read file cookies.json: %v", err)
	}
	var items []struct {
		Name           string  `json:"name"`
		Value          string  `json:"value"`
		Domain         string  `json:"domain"`
		Path           string  `json:"path"`
		Secure         bool    `json:"secure"`
		HttpOnly       bool    `json:"httpOnly"`
		ExpirationDate float64 `json:"expirationDate"`
		SameSite       string  `json:"sameSite"`
		Session        bool    `json:"session"`
	}
	if err := json.Unmarshal(dat, &items); err != nil {
		return fmt.Errorf("Cannot unmarshal cookies.json: %v", err)
	}
	uApi, _ := url.Parse("https://api.ozon.ru/")
	uWWW, _ := url.Parse("https://www.ozon.ru/")
	for _, c := range items {
		ck := &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
		}
		switch c.SameSite {
		case "no_restriction":
			ck.SameSite = http.SameSiteNoneMode
		case "lax":
			ck.SameSite = http.SameSiteLaxMode
		case "strict":
			ck.SameSite = http.SameSiteStrictMode
		default:
			ck.SameSite = http.SameSiteDefaultMode
		}
		if !c.Session && c.ExpirationDate > 0 {
			sec := int64(c.ExpirationDate)
			ck.Expires = time.Unix(sec, 0)
		}

		jar.SetCookies(uApi, []*http.Cookie{ck})
		jar.SetCookies(uWWW, []*http.Cookie{ck})
	}
	fmt.Printf("[OZON] Cookies loaded!")
	return nil
}

var globalCookie *cookiejar.Jar

// получение json
func ozonResponse(query string) ([]byte, error) {
	searchpath := "/search?text=" + url.QueryEscape(query) + "&sorting=price&page=1"
	apiUrl := "https://api.ozon.ru/composer-api.bx/page/json/v2?url=" + url.QueryEscape(searchpath)
	referer := "https://www.ozon.ru/search/?text=" + url.QueryEscape(query) // для warmup

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("[OZON] cookiejar:%w", err)
	}
	globalCookie = jar
	if err := loadCookies(jar); err != nil {
		return nil, fmt.Errorf("[OZON] load cookies err: %w", err)
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
		Jar: globalCookie,
	}

	if err := warmUp(client); err != nil {
		return nil, fmt.Errorf("[OZON] warmup: %w", err)
	}

	current := apiUrl
	for step := 0; step < 2; step++ {
		fmt.Println("[OZON] Step:", step)
		fmt.Println("[OZON] URL:", current)
		seconds := rand.Intn(8) + 3
		time.Sleep(time.Duration(seconds) * time.Second)
		req, err := http.NewRequest("GET", current, nil)
		if err != nil {
			return nil, fmt.Errorf("[OZON] new request: %w", err)
		}

		setHeaders(req, referer)
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("[OZON] sending request err: %w", err)
		}

		if resp.StatusCode == 301 || resp.StatusCode == 302 || resp.StatusCode == 303 || resp.StatusCode == 307 || resp.StatusCode == 308 {
			loc := resp.Header.Get("Location")
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if loc == "" {
				return nil, fmt.Errorf("[OZON] redirect without location")
			}
			if strings.HasPrefix(loc, "/") {
				loc = "https://api.ozon.ru" + loc
			} else if strings.HasPrefix(loc, "composer-api") {
				loc = "https://api.ozon.ru/" + loc
			}
			current = loc
			time.Sleep(time.Second * 2)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			fmt.Println("[OZON] read error:", readErr)
		}

		if resp.StatusCode == 200 {
			fmt.Println("OZON OK", resp.Status)
			return body, nil
		}
		fmt.Println("[OZON] Unexpected status:", resp.Status)
		if len(body) > 0 {
			s := string(body)
			if len(s) > 1000 {
				s = s[:1000]
			}
			fmt.Println("Body snippet:\n", s)
		}
	}
	return nil, errors.New("[OZON]  не удалось получить данные")

}

func getMainStateText(item gjson.Result, stateType string) gjson.Result {
	var found gjson.Result
	item.Get("mainState").ForEach(func(_, st gjson.Result) bool {
		if st.Get("type").String() == stateType {
			found = st
			return false
		}
		return true
	})
	return found
}

func normalizeText(s string) string {
	s = strings.TrimSpace(s)
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00A0", " ")
	s = strings.ReplaceAll(s, "\u202F", " ")
	s = strings.ReplaceAll(s, "\u2009", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func Parse(query string) ([]models.Product, error) {
	ozon, err := ozonResponse(query)
	if err != nil {
		return nil, fmt.Errorf("[OZON] json collection error:%w", err)
	}

	root := gjson.ParseBytes(ozon)

	var tileKey string
	root.Get("widgetStates").ForEach(func(k, _ gjson.Result) bool {
		if strings.HasPrefix(k.String(), "tileGridDesktop-") {
			tileKey = k.String()
			return false
		}
		return true
	})
	if tileKey == "" {
		return nil, fmt.Errorf("[OZON_parsing] tileGridDesktop-* not found")
	}

	tileStr := root.Get("widgetStates." + tileKey).String()
	tile := gjson.Parse(tileStr)

	items := tile.Get("items")
	if !items.Exists() || !items.IsArray() {
		return nil, fmt.Errorf("[OZON_parsing] items not found or not array")
	}

	fmt.Println("items:", len(items.Array()))

	var products []models.Product
	items.ForEach(func(_, item gjson.Result) bool {
		sku := item.Get("sku").String()
		link := item.Get("action.link").String()
		textAtom := getMainStateText(item, "textAtom")
		title := textAtom.Get("textAtom.text").String()
		if link != "" && strings.HasPrefix(link, "/") {
			link = "https://www.ozon.ru" + link
		}
		priceV2State := getMainStateText(item, "priceV2")
		priceV2 := priceV2State.Get("priceV2")

		var priceNow string
		var originalPrice string
		priceV2.Get("price").ForEach(func(_, price gjson.Result) bool {
			ts := price.Get("textStyle").String()
			txt := price.Get("text").String()

			switch ts {
			case "PRICE":
				priceNow = txt
			case "ORIGINAL_PRICE":
				originalPrice = txt
			}
			return true
		})
		priceNow = normalizeText(priceNow)
		originalPrice = normalizeText(originalPrice)

		img := item.Get("tileImage.items.0.image.link").String()

		var stars, reviews, statistic string
		item.Get("mainState").ForEach(func(_, st gjson.Result) bool {
			if st.Get("type").String() != "labelList" {
				return true
			}
			s := st.Get("labelList.items.0.title").String()
			r := st.Get("labelList.items.1.title").String()
			if s != "" && r != "" && strings.Contains(r, "отзыв") {
				stars = s
				reviews = r
				return false
			}
			return true
		})

		stars = normalizeText(stars)
		reviews = normalizeText(reviews)
		title = normalizeText(title)

		if stars != "" && reviews != "" {
			statistic = stars + " • " + reviews
		}

		p := models.Product{
			Link:             link,
			IMG:              img,
			ProductID:        sku,
			ProductName:      title,
			DiscountPrice:    priceNow,
			BasePrice:        originalPrice,
			ProductStatistic: statistic,
			ProductStars:     stars,
			ProductReviews:   reviews,
		}
		products = append(products, p)

		return true
	})

	return products, nil
}
