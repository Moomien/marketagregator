import React from 'react'

type Product = {
  product_url: string
  image_url: string
  product_id: string
  product_name: string
  product_discount_price: string
  product_base_price: string
  product_statistic: string
  product_stars: string
  product_reviews: string
}

type Props = { product: Product }

function parseStars(s: string) {
  const v = s.replace(',', '.').trim()
  const n = parseFloat(v)
  if (isNaN(n)) return null
  return Math.max(0, Math.min(5, n))
}

function renderStars(stars: number | null) {
  if (stars == null) return ''
  const full = Math.floor(stars)
  const half = stars - full >= 0.5
  const empty = 5 - full - (half ? 1 : 0)
  return '★'.repeat(full) + (half ? '☆' : '') + '☆'.repeat(empty)
}

function normalizePrice(p: string) {
  if (!p) return ''
  return String(p).replace(/\s+/g, ' ').trim()
}

export default function ProductCard({ product }: Props) {
  const link = product.product_url
  const img = product.image_url
  const name = product.product_name
  const priceNow = normalizePrice(product.product_discount_price)
  const priceOld = normalizePrice(product.product_base_price)
  const starsVal = parseStars(product.product_stars)
  const reviewsText = product.product_reviews
  const statistic = product.product_statistic

  return (
    <div className="card">
      <div className="thumb">
        {img ? <img src={img} alt={name} onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }} /> : null}
      </div>
      <div className="content">
        <div className="title" title={name}>{name || 'Товар'}</div>
        <div className="price-row">
          {priceNow ? <div className="price-now">{priceNow}</div> : null}
          {priceOld ? <div className="price-old">{priceOld}</div> : null}
        </div>
        <div className="stats">
          {starsVal != null ? <span className="stars" title={product.product_stars}>{renderStars(starsVal)}</span> : null}
          {reviewsText ? <span>{reviewsText} отзывов</span> : null}
          {!starsVal && !reviewsText && statistic ? <span>{statistic}</span> : null}
        </div>
        <div className="actions">
          {link ? <a className="primary" href={link} target="_blank" rel="noopener">Открыть на сайте</a> : null}
        </div>
      </div>
    </div>
  )
}
