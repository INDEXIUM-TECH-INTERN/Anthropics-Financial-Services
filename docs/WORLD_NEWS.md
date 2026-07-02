# Bản tin Tài chính Thế giới (Morning Digest)

> Tab **Bản tin Thế giới** trên frontend — tổng hợp thị trường và tin tức trước **07:00 GMT+7** mỗi ngày.

## Tổng quan

| Mục | Mô tả |
|-----|--------|
| Khung tin | 24h trước 07:00 sáng (GMT+7) |
| Chốt giá | Phiên Mỹ gần nhất (ET); dầu/vàng có snap tại cutoff |
| Nguồn chứng khoán | CNBC (16 mã), fallback Yahoo |
| Nguồn tin | CNBC, Reuters, WSJ, Google News RSS (VTV/VN) |
| Tóm tắt AI | Gemini — 800–1000 từ, **4–6 đoạn văn** |
| Cache | Theo ngày + `reportVersion` (hiện tại: **v27**) |

## API

### `GET /api/world-news?date=YYYY-MM-DD`

Trả về báo cáo đầy đủ cho ngày lịch (mặc định: hôm nay GMT+7).

**Response (rút gọn):**

```json
{
  "date": "2026-07-02",
  "reportVersion": 27,
  "digestWindow": "01/07/2026 07:00 – 02/07/2026 07:00",
  "digestUntil": "02/07/2026 07:00",
  "highlightSummary": "Đoạn 1...\n\nĐoạn 2...",
  "keyNumbers": [],
  "stocks": { "instruments": [], "tabs": [] },
  "breakingNews": [
    {
      "date": "02/07/2026",
      "time": "06:27",
      "source": "CNBC",
      "content": "...",
      "url": "https://www.cnbc.com/2026/07/02/..."
    }
  ],
  "generatedAt": "2026-07-02T07:15:00+07:00",
  "dataSource": "CNBC, Yahoo Finance, ..."
}
```

- `highlightSummary`: nhiều đoạn, phân tách bằng `\n\n` (frontend render thành từng `<p>`).
- `breakingNews[].date` + `time`: hiển thị dạng `02/07/2026 · 06:27` trên UI.

### `GET /api/world-news/dates`

Danh sách ngày cho dropdown (90 ngày gần nhất).

### `GET /api/world-news/favicon?host=...`

Proxy favicon nhà xuất bản (tránh CORS).

### `GET /api/world-news/image?url=...`

Proxy ảnh thumbnail bài viết.

## Cấu trúc code

```
Gemini/internal/worldnews/
├── service.go           # buildReport, cache, reportCacheVersion
├── digest.go            # MorningDigestHour = 7 (GMT+7)
├── cnbc.go              # Quotes Brent, Gold, S&P; gold cutoff qua Yahoo 1m
├── stocks.go              # 16 mã CNBC
├── rss.go                 # Breaking / VTV news, filter trước 07:00
├── highlights_summary*.go # Tóm tắt 800–1000 từ, nhiều đoạn
└── types.go               # WorldNewsReport schema

frontend/src/pages/chat/page.ts   # renderWorldNews, breaking cards
frontend/src/styles/world-news.css
```

## Cache bust

Tăng `reportCacheVersion` trong `service.go` khi đổi schema JSON hoặc logic tổng hợp quan trọng. Key cache: `{date}:v{version}`.

## Kiểm tra nhanh

```powershell
# Thay 3000 bằng port bạn đang dùng
curl "http://localhost:3000/api/world-news?date=2026-07-02"
```

Kỳ vọng: `reportVersion=27`, `breakingNews[].date` có giá trị, `highlightSummary` chứa `\n\n`.

## Ghi chú vận hành

- **Redis** (tùy chọn): cache báo cáo; không có Redis vẫn chạy in-memory.
- **Lần tải đầu** có thể mất 20–40s (fetch CNBC, RSS, AI summary).
- **Gold tại cutoff**: Yahoo `GC=F` 1m — bar ngay trước 07:00 (06:59 GMT+7).