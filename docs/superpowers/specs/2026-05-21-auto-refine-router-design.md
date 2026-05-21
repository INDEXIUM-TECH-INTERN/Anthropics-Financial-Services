# Kế hoạch thiết kế: Vòng lặp tự tối ưu hóa Router (Self-Improving Router Loop)

## 1. Mục tiêu
Xây dựng hệ thống tự động kiểm tra, đánh giá và cập nhật prompt cho Router để xử lý thời gian và intent chính xác, dựa trên Option 2 (Full Context).

## 2. Các thành phần chính
- **Bộ Test Cases (`evaluator/test_cases.json`):** Chứa các câu hỏi khó về thời gian (ví dụ: "giá hôm kia", "thứ 2 tuần trước", "ngày mai").
- **Hành động Đánh giá (Evaluator):** Tôi (AI Agent) sẽ chạy từng test case, so sánh kết quả JSON của Router với logic thực tế.
- **Vòng lặp sửa lỗi (Refinement Loop):** 
    1. Chạy test.
    2. Nếu fail -> Phân tích lý do (hallucination, sai logic thời gian).
    3. Cập nhật `router_system_prompt.txt` hoặc `router_user_prompt.txt`.
    4. Chạy lại test cho đến khi pass hoặc đạt giới hạn retry.

## 3. Cấu trúc JSON mới cho RoutePlan
```go
type RoutePlan struct {
	Agent    string   `json:"agent"`
	Skills   []string `json:"skills"`
	Temporal struct {
		Intent       string `json:"intent"`        // e.g., "historical", "latest", "realtime"
		ResolvedDate string `json:"resolved_date"` // Định dạng YYYY-MM-DD
		IsFuture     bool   `json:"is_future"`
	} `json:"temporal"`
	Reason string `json:"reason"`
}
```

## 4. Các bước thực hiện
1. Cập nhật `routing.go` và `models` để hỗ trợ cấu trúc JSON mới.
2. Viết file `router_system_prompt.txt` mới với các chỉ dẫn cực kỳ khắt khe về thời gian.
3. Tạo script/công cụ để tôi có thể kích hoạt vòng lặp test từ CLI.
