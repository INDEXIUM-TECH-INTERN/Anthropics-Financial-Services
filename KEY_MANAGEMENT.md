# API Key Management Guide

## Overview
TestAIFinance sử dụng **Key Pool** để quản lý API keys với auto-load balancing và redundancy.

## Architecture

### Key Pool System
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Gemini API    │    │   SerpAPI      │    │   Tavily        │
│   Key Pool      │    │   Key Pool     │    │   Key Pool      │
│                 │    │                 │    │                 │
│ - key1          │    │ - serp1         │    │ - tav1          │
│ - key2          │    │ - serp2         │    │ - tav2          │
│ - key3          │    │ - serp3         │    │ - tav3          │
│ - ...           │    │ - ...           │    │ - ...           │
└─────────────────┘    └─────────────────┘    └─────────────────┘
       │                       │                       │
       └───────────┬───────────┼───────────┬───────────┘
                   ▼           ▼           ▼
            ┌─────────────────────────────────────┐
            │        Key Pool Manager            │
            │ - Round-robin scheduling           │
            │ - Usage tracking                   │
            │ - Auto-switch on failure           │
            │ - Health monitoring                │
            └─────────────────────────────────────┘
```

## Environment Variables

### Backend Service
```yaml
# Gemini Keys (Comma-separated)
GEMINI_API_KEYS="key1,key2,key3,key4,key5,key6"

# SerpAPI Keys (Comma-separated)  
SERPAPI_KEYS="serp1,serp2,serp3"

# Tavily Keys (Comma-separated)
TAVILY_KEYS="tav1,tav2,tav3"

# OpenRouter Keys (Optional)
OPENROUTER_KEYS="router1,router2"

# Fallback Single Keys
GEMINI_API_KEY_FALLBACK="backup_key"
SERPAPI_KEY_FALLBACK="backup_serp"
TAVILY_KEY_FALLBACK="backup_tav"
```

## How Key Pool Works

### 1. Round-Robin Scheduling
```go
// Đầu tiên dùng key1, sau key2, sau key3, rồi quay lại key1
keyPool.GetKey() // -> key1
keyPool.GetKey() // -> key2  
keyPool.GetKey() // -> key3
keyPool.GetKey() // -> key1 (quay lại đầu)
```

### 2. Least-Used Strategy
```go
// Tự động chọn key ít được dùng nhất
keyPool.GetLeastUsedKey() // -> xem usage stats và chọn key phù hợp
```

### 3. Random Rotation
```go
// Ngẫu nhiên chọn key (good for load distribution)
keyPool.GetRandomKey()
```

### 4. Auto-Switch on Failure
Khi một key hết hạn hoặc lỗi:
- System tự động thử key tiếp theo
- Logging lỗi để monitoring
- Health score tự động giảm

## Configuration in Render

### Step 1: Set Environment Variables
1. Login to Render Dashboard
2. Go to Backend Service → Settings → Environment
3. Add these variables:

```env
GEMINI_API_KEYS="your_key1,your_key2,your_key3"
SERPAPI_KEYS="your_serp1,your_serp2"
TAVILY_KEYS="your_tav1,your_tav2"
```

### Step 2: Test Keys
```bash
# Test locally
cd Gemini
go run cmd/gemini-cli/main.go --server
```

### Step 3: Monitor Usage
System tự động:
- Track usage per key
- Switch on rate limits
- Log error patterns
- Calculate health scores

## Benefits

### 1. High Availability
- Một key lỗi → tự động switch
- Không downtime khi hết quota
- Load balancing tự động

### 2. Cost Optimization
- Sử dụng tất cả keys đều nhau
- Tránh một key bị overuse
- Efficient resource usage

### 3. Easy Maintenance
- Add/remove keys runtime
- No code changes needed
- Auto-configuration

### 4. Monitoring
- Real-time health scores
- Usage statistics per key
- Error tracking & alerts

## Troubleshooting

### Common Issues

1. **All Keys Fail**
   ```
   Error: all API keys failed
   Solution: Check API keys validity in Render dashboard
   ```

2. **Rate Limited**
   ```
   Error: quota exceeded
   Solution: Pool auto-switches to next key
   ```

3. **Connection Issues**
   ```
   Error: connection refused
   Solution: Check service health in Render dashboard
   ```

### Debug Commands
```bash
# Check key pool status
curl http://localhost:8080/api/stats

# View current key usage
curl http://localhost:8080/api/keys/stats

# Force key rotation
curl -X POST http://localhost:8080/api/keys/rotate
```

## Best Practices

### 1. Key Management
- Rotate keys monthly
- Monitor usage patterns
- Keep 2-3 backup keys
- Test keys before deployment

### 2. Configuration
- Use different keys for different environments
- Store keys securely in Render dashboard
- Document keys with expiration dates
- Regular audit key usage

### 3. Monitoring
- Set up alerts for error rate > 5%
- Monitor health scores weekly
- Track API response times
- Check quota usage daily

## Example Usage Code

```go
// Khởi tạo
provider, err := NewGeminiProviderWithPool(
    os.Getenv("GEMINI_API_KEYS"),
    "gemini-3.1-flash-lite",
    3,
)

// Gửi message
response, err := provider.SendMessage(ctx, "Hello")

// Get stats
stats := provider.GetStats()
fmt.Printf("Health score: %.2f\n", stats["health_score"])
```