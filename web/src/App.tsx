import React, { useState } from 'react'
import ProductCard from './components/ProductCard'

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

export default function App() {
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [items, setItems] = useState<Product[]>([])

  function parsePrice(s: string) {
    const cleaned = String(s || '').replace(/[^\d.,]/g, '').replace(/\s+/g, '')
    const normalized = cleaned.replace(',', '.')
    const n = parseFloat(normalized)
    return isNaN(n) ? Infinity : n
  }

  function normalizeText(s: string) {
    return String(s || '')
      .toLowerCase()
      .normalize('NFKC')
      .replace(/[^a-z0-9а-яё]+/gi, ' ')
      .replace(/\s+/g, ' ')
      .trim()
  }
  function queryTokens(s: string) {
    const n = normalizeText(s)
    return n ? n.split(' ').filter(Boolean) : []
  }
  function matchesQuery(name: string, qraw: string) {
    const nn = normalizeText(name)
    const tokens = queryTokens(qraw)
    if (!tokens.length) return true
    return tokens.every(t => {
      if (/^\d+$/.test(t)) {
        const re = new RegExp(`(?:^|\\D)${t}(?:\\D|$)`)
        return re.test(nn)
      }
      return nn.includes(t)
    })
  }

  async function search() {
    if (!q.trim()) return
    setLoading(true)
    setError(null)
    setItems([])
    try {
      const r = await fetch(`/search?query=${encodeURIComponent(q.trim())}`)
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data: Product[] = await r.json()
      const filtered = data.filter(p => matchesQuery(p.product_name, q))
      const sorted = [...filtered].sort((a, b) => parsePrice(a.product_discount_price) - parsePrice(b.product_discount_price))
      setItems(sorted)
    } catch (e: any) {
      setError(e.message || 'Ошибка')
    } finally {
      setLoading(false)
    }
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    search()
  }

  const bestItem = items.length ? items.reduce((min, p) =>
    parsePrice(p.product_discount_price) < parsePrice(min.product_discount_price) ? p : min,
    items[0]
  ) : null

  function shopName(url: string) {
    try {
      const u = new URL(url)
      const h = u.hostname
      if (h.includes('ozon')) return 'Ozon'
      if (h.includes('wildberries')) return 'Wildberries'
      return h.replace('www.', '')
    } catch {
      return ''
    }
  }

  const bestPriceText = bestItem ? `${bestItem.product_discount_price} — ${shopName(bestItem.product_url)}` : ''

  return (
    <div className="app">
      <header className="header">
        <div className="container">
          <div className="brand">
            <h1>Агрегатор цен</h1>
            <p>Сравниваем цены из нескольких магазинов. Введите запрос — покажем лучшие предложения.</p>
            <div className="badges">
              <span className="badge">Ozon</span>
              <span className="badge">Wildberries</span>
            </div>
          </div>
          <form className="search-bar" onSubmit={onSubmit}>
            <input value={q} onChange={e => setQ(e.target.value)} placeholder="Введите запрос для сравнения цен" />
            <button type="submit">Сравнить</button>
          </form>
          <div className="meta">{items.length ? `Найдено: ${items.length} • Лучшая цена: ${bestPriceText}` : 'Введите запрос, например «iphone 15»'}</div>
        </div>
      </header>
      <main className="container">
        {loading && <div className="state loading">Загрузка…</div>}
        {error && <div className="state error">Ошибка: {error}</div>}
        {!loading && !error && items.length === 0 && q && <div className="state empty">Ничего не найдено</div>}
        <div className="grid" style={{ display: items.length ? 'grid' : 'none' }}>
          {items.map((p, i) => <ProductCard key={p.product_id || i.toString()} product={p} />)}
        </div>
      </main>
    </div>
  )
}
