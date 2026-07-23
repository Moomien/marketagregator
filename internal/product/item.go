package product

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Product struct {
	Link                 string `json:"product_url"`
	IMG                  string `json:"image_url"`
	ProductID            string `json:"product_id"`
	ProductName          string `json:"product_name"`
	DiscountPriceKopecks int64  `json:"product_discount_price"`
	BasePriceKopecks     int64  `json:"product_base_price"`
	ProductStatistic     string `json:"product_statistic"`
	ProductStars         string `json:"product_stars"`
	ProductReviews       string `json:"product_reviews"`
}

func ParsePrice(value string) (int64, error) {
	original := value
	var digits strings.Builder
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9', r == ',', r == '.':
			digits.WriteRune(r)
		case unicode.IsSpace(r), r == '₽':
			continue
		default:
			return 0, fmt.Errorf("invalid price %q", original)
		}
	}
	value = digits.String()
	if value == "" {
		return 0, fmt.Errorf("empty price")
	}

	separator := strings.LastIndexAny(value, ",.")
	if separator == -1 {
		rubles, err := strconv.ParseInt(value, 10, 64)
		if err != nil || rubles <= 0 {
			return 0, fmt.Errorf("invalid price %q", value)
		}
		return rubles * 100, nil
	}

	rublesText := strings.ReplaceAll(strings.ReplaceAll(value[:separator], ",", ""), ".", "")
	kopecksText := value[separator+1:]
	if rublesText == "" || len(kopecksText) == 0 || len(kopecksText) > 2 {
		return 0, fmt.Errorf("invalid price %q", value)
	}
	rubles, rublesErr := strconv.ParseInt(rublesText, 10, 64)
	kopecks, kopecksErr := strconv.ParseInt(kopecksText, 10, 64)
	if rublesErr != nil || kopecksErr != nil || rubles < 0 {
		return 0, fmt.Errorf("invalid price %q", value)
	}
	if len(kopecksText) == 1 {
		kopecks *= 10
	}
	return rubles*100 + kopecks, nil
}
