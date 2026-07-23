import json
import sys
import io
import atexit
from urllib.parse import quote
from typing import List, Dict
import re

import undetected_chromedriver as uc
from bs4 import BeautifulSoup
from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait
uc.Chrome.__del__ = lambda self: None

_active_driver = None

"""
Программа открывает поиск Ozon по заданному запросу, собирает ссылки карточек, 
затем по каждой ссылке загружает страницу товара, извлекает HTML и парсит его с помощью BeautifulSoup. 
Извлекаются поля: product_url, image_url (основное изображение из галереи товара), product_id (артикул), 
product_name, цены product_discount_price, product_base_price, 
product_statistic (средняя оценка + количество отзывов), product_stars (средняя оценка), product_reviews (количество отзывов).
статистика и отзывы. 
Выводится один JSON-массив товаров в stdout, который используется в main.go для сохранения в бд.   
Запуск(пример): python adapters/ozon/fallback.py "iphone 15".
"""

def _cleanup_driver():
    global _active_driver
    if _active_driver:
        try:
            _active_driver.quit()
        except Exception:
            pass
        _active_driver = None

atexit.register(_cleanup_driver)


def page_down(driver):
    try:
        driver.execute_script(
            """
            const scrollStep = 400;
            const scrollInterval = 150;
            const scrollHeight = document.documentElement.scrollHeight;
            let currentPosition = 0;
            const interval = setInterval(() => {
                window.scrollBy(0, scrollStep);
                currentPosition += scrollStep;
                if (currentPosition >= scrollHeight) {
                    clearInterval(interval);
                }
            }, scrollInterval);
            """
        )
    except Exception:
        pass


def collect_product_info(driver, wait: WebDriverWait, url: str) -> Dict[str, str]:
    try:
        if len(driver.window_handles) == 1:
            driver.switch_to.new_window("tab")
        else:
            driver.switch_to.window(driver.window_handles[1])

        driver.get(url)

        try:
            wait.until(EC.presence_of_element_located((By.XPATH, '//div[contains(@data-widget,"webProductHeading")]//h1')))
        except Exception:
            pass

        product_id = ""
        try:
            el = wait.until(EC.presence_of_element_located((By.XPATH, '//div[contains(text(),"Артикул: ")]')))
            product_id = el.text.split("Артикул: ")[1].strip()
        except Exception:
            pass

        page_source = driver.page_source
        soup = BeautifulSoup(page_source, "lxml")

        product_name = ""
        try:
            name_tag = soup.find("div", attrs={"data-widget": "webProductHeading"})
            if name_tag:
                h1 = name_tag.find("h1")
                if h1:
                    product_name = h1.text.strip().replace("\t", "").replace("\n", " ")
        except Exception:
            pass

        image_url = ""
        try:
            gallery = soup.find("div", attrs={"data-widget": "webGallery"})
            img = None
            if gallery:
                img = gallery.find("img", src=True) or gallery.find("img", attrs={"srcset": True})
            if not img:
                img = soup.find("img", attrs={"elementtiming": lambda s: s and "webGallery" in s}) or soup.find("img", attrs={"data-lcp-name": lambda s: s and "webGallery" in s})
            if img:
                src = img.get("src") or ""
                if not src:
                    srcset = img.get("srcset") or ""
                    if srcset:
                        parts = [p.strip().split()[0] for p in srcset.split(",") if p.strip()]
                        pref = None
                        for p in parts:
                            if "wc1000" in p:
                                pref = p
                                break
                        src = pref or (parts[-1] if parts else "")
                if src:
                    image_url = src
            if not image_url:
                meta_og = soup.find("meta", attrs={"property": "og:image"})
                if meta_og and meta_og.get("content"):
                    image_url = meta_og.get("content") or ""
            if image_url and "ir.ozone.ru" not in image_url:
                for i in soup.find_all("img", src=True):
                    src = i.get("src") or ""
                    if "ir.ozone.ru" in src:
                        image_url = src
                        break
            if image_url:
                image_url = re.sub(r"/wc\d+/", "/wc1000/", image_url)
        except Exception:
            image_url = ""

        product_stars = ""
        product_reviews = ""
        product_statistic = ""
        try:
            score = soup.find("div", attrs={"data-widget": "webSingleProductScore"})
            txt = score.text.strip() if score else ""
            if " • " in txt:
                product_stars, product_reviews = [t.strip() for t in txt.split(" • ", 1)]
                product_statistic = f"{product_stars} • {product_reviews}"
            else:
                product_statistic = txt
        except Exception:
            pass

        product_discount_price = ""
        product_base_price = ""

        def _extract_text(el) -> str:
            try:
                return el.text.strip()
            except Exception:
                return ""

        try:
            card_label = soup.find("span", string=lambda s: s and ("с Ozon Картой" in s or "c Ozon Картой" in s))
            if card_label:
                card_block = card_label.parent
                card_price_span = card_block.find("div")
                card_price_span = card_price_span.find("span") if card_price_span else None
                product_discount_price = _extract_text(card_price_span)
        except Exception:
            pass

        try:
            no_card_label = soup.find("span", string=lambda s: s and ("без Ozon Карты" in s))
            if no_card_label:
                price_spans = no_card_label.parent.parent.find("div").find_all("span")
                if price_spans:
                    first = _extract_text(price_spans[0]) if len(price_spans) >= 1 else ""
                    second = _extract_text(price_spans[1]) if len(price_spans) >= 2 else ""
                    if not product_discount_price:
                        product_discount_price = first
                    product_base_price = second or first
        except Exception:
            pass

        if not product_discount_price or not product_base_price:
            try:
                spans = soup.find("div", attrs={"data-widget": "webPrice"}).find_all("span")
                values = [s.text.strip() for s in spans if s and s.text.strip()]
                if values:
                    if not product_discount_price:
                        product_discount_price = values[0]
                    if not product_base_price and len(values) >= 2:
                        product_base_price = values[1]
            except Exception:
                pass

        product_data = {
            "product_url": url or "",
            "image_url": image_url or "",
            "product_id": product_id or "",
            "product_name": product_name or "",
            "product_discount_price": product_discount_price or "",
            "product_base_price": product_base_price or "",
            "product_statistic": product_statistic or "",
            "product_stars": product_stars or "",
            "product_reviews": product_reviews or "",
        }

        return product_data
    
    except Exception as e:
        return {
            "product_url": url or "",
            "image_url": "",
            "product_id": "",
            "product_name": "",
            "product_discount_price": "",
            "product_base_price": "",
            "product_statistic": "",
            "product_stars": "",
            "product_reviews": "",
        }


def get_products_links(item_name: str) -> List[Dict[str, str]]:
    global _active_driver
    
    options = uc.ChromeOptions()
    options.add_argument("--headless")
    options.add_argument("--disable-blink-features=AutomationControlled")
    options.add_argument("--start-maximized")
    options.add_argument("--disable-dev-shm-usage")
    options.add_argument("--no-sandbox")
    
    driver = None
    try:
        driver = uc.Chrome(options=options)
        _active_driver = driver
        wait = WebDriverWait(driver, 20)

        search_url = "https://ozon.ru/search/?text=" + quote(item_name) + "&sorting=price"
        driver.get(search_url)

        wait.until(EC.presence_of_element_located((By.CLASS_NAME, "tile-clickable-element")))
        page_down(driver)
        wait.until(EC.presence_of_all_elements_located((By.CLASS_NAME, "tile-clickable-element")))

        products_urls: List[str] = []
        try:
            tiles = driver.find_elements(By.CLASS_NAME, "tile-clickable-element")
            products_urls = list({t.get_attribute("href") for t in tiles if t.get_attribute("href")})
        except Exception as e:
            return []

        products_data: List[Dict[str, str]] = []
        
        try:
            driver.switch_to.new_window("tab")
        except Exception:
            pass

        for i, url in enumerate(products_urls, 1):
            try:
                data = collect_product_info(driver=driver, wait=wait, url=url)
                products_data.append(data)
            except Exception as e:
                continue

        return products_data
        
    except Exception as e:
        return []
    
    finally:
        if driver:
            try:
                while len(driver.window_handles) > 1:
                    driver.switch_to.window(driver.window_handles[-1])
                    driver.close()
                
                driver.quit()
                _active_driver = None
            except Exception:
                pass

def main():
    args = [a for a in sys.argv[1:] if a and not a.startswith("--")]
    query = " ".join(args) if args else None

    try:
        if not query:
            print(json.dumps([], ensure_ascii=False, indent=2))
            return
        products_data = get_products_links(query)
        print(json.dumps(products_data, ensure_ascii=False, indent=2))
    except Exception as e:
        print(json.dumps([], ensure_ascii=False, indent=2))


if __name__ == '__main__':
    main()
