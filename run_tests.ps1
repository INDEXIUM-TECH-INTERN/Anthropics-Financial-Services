$queries = @(
    "Tổng tài sản của HDB trong 10 năm qua thay đổi thế nào?",
    "Chỉ tiêu 'Chi phí dự phòng rủi ro tín dụng' nằm ở trang nào trong báo cáo tài chính năm 2023 của HDB?",
    "Trích dẫn báo cáo nói về chi phí dự phòng của HDB năm 2024",
    "Chi phí dự phòng rủi ro của HDB năm 2024",
    "Báo cáo tài chính 6 tháng đầu năm 2025?",
    "HDB có những lợi thế gì trong những năm gần đây để bức tốc phát triển trong các năm sắp tới?",
    "Ban lãnh đạo hiện tại của HDB gồm những ai?",
    "Tổng quát ngành ngân hàng năm 2025",
    "So sánh tổng tài sản HDB và ACB trong 3 năm gần đây"
)

foreach ($q in $queries) {
    Write-Host "`n🚀 Testing Query: $q"
    $body = @{ message = $q } | ConvertTo-Json
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/api/chat" -Method Post -Body $body -ContentType "application/json" -TimeoutSec 120
        if ($response.error) {
            Write-Host "❌ Error from API: $($response.error)"
            exit 1
        } else {
            Write-Host "✅ Response received ($($response.metrics.latency_ms)ms)"
            # Write-Host "🤖 Reply: $($response.reply.Substring(0, [Math]::Min(100, $response.reply.Length)))..."
        }
    } catch {
        Write-Host "❌ Request Failed: $_"
        exit 1
    }
    Start-Sleep -Seconds 2
}

Write-Host "`n✨ All tests completed successfully!"
