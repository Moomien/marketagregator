package wb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"agregator/internal/product"

	"github.com/tidwall/gjson"
)

type Client struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Client {
	return &Client{logger: logger.With("marketplace", "wb")}
}

func (c *Client) Search(ctx context.Context, query string) ([]product.Product, error) {
	body, err := c.wildberries(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("[WB] ошибка сбора json WB:%w", err)
	}
	root := gjson.ParseBytes(body)
	products := root.Get("products")
	if !products.Exists() || !products.IsArray() {
		return nil, errors.New("[WB] items not found or not array")
	}

	var (
		items    []product.Product
		parseErr error
	)
	products.ForEach(func(_, products gjson.Result) bool {
		id := products.Get("id").Int()
		link := "https://www.wildberries.ru/catalog/" + strconv.FormatInt(id, 10) + "/detail.aspx"
		name := products.Get("name").String()
		discountPrice := products.Get("sizes.0.price.product")
		if !discountPrice.Exists() || discountPrice.Int() <= 0 {
			parseErr = fmt.Errorf("parse discount price for WB product %d", id)
			return false
		}
		basePrice := products.Get("sizes.0.price.basic")
		stars := products.Get("rating").String()
		reviews := products.Get("feedbacks").String()
		var statistic string
		if stars != "" && reviews != "" {
			statistic = stars + " • " + reviews
		}
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

		p := product.Product{
			Link:                 link,
			IMG:                  pic,
			ProductID:            strconv.FormatInt(id, 10),
			ProductName:          name,
			DiscountPriceKopecks: discountPrice.Int(),
			BasePriceKopecks:     basePrice.Int(),
			ProductStatistic:     statistic,
			ProductStars:         stars,
			ProductReviews:       reviews,
		}
		items = append(items, p)
		return true
	})
	if parseErr != nil {
		return nil, parseErr
	}
	c.logger.Info("parsed products", "count", len(products.Array()))
	return items, nil
}

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

func warmUp(ctx context.Context, client *http.Client) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.wildberries.ru/", nil)
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

func (c *Client) wildberries(ctx context.Context, query string) ([]byte, error) {
	apiUrl := "https://search.wb.ru/exactmatch/ru/common/v18/search?appType=1&curr=rub&dest=-1257786&lang=ru&page=1&query=" + url.QueryEscape(query) + "&resultset=catalog&sort=priceup&spp=30"
	referer := "https://www.wildberries.ru/catalog/0/search.aspx?search=" + url.QueryEscape(query)

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("[WB] ошибка cookiejar:%w", err)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	proxyURLText := strings.TrimSpace(os.Getenv("PROXY_URL"))
	if proxyURLText != "" {
		proxyURL, err := url.Parse(proxyURLText)
		if err != nil {
			return nil, fmt.Errorf("parse PROXY_URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
		c.logger.Info("using proxy from environment")
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: jar,
	}

	if err := warmUp(ctx, client); err != nil {
		return nil, fmt.Errorf("[WB] Warmup errors:%w", err)
	}

	for attempt := 0; attempt <= 10; attempt++ {
		c.logger.Debug("request attempt", "attempt", attempt+1, "url", apiUrl)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiUrl, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		setHeaders(req, referer)
		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			c.logger.Warn("request failed", "attempt", attempt+1, "error", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response body: %w", err)
		}

		if resp.StatusCode == 200 {
			c.logger.Debug("request completed", "status", resp.StatusCode)
			return body, nil
		}
		if err := wait(ctx, 2*time.Second); err != nil {
			return nil, err
		}
		if len(body) > 0 {
			s := string(body)
			if len(s) > 1000 {
				s = s[:1000]
			}
			c.logger.Debug("response body", "body", s)
		}
		c.logger.Warn("unexpected response status", "status", resp.StatusCode)
	}
	return nil, errors.New("[WB]  не удалось получить данные")
}

func wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
