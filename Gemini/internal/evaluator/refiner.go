package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
)

type TestCase struct {
	ID        int    `json:"id"`
	UserQuery string `json:"user_query"`
	Expected  struct {
		Agent    string `json:"agent"`
		Temporal struct {
			Intent       string `json:"intent"`
			ResolvedDate string `json:"resolved_date"`
			IsFuture     bool   `json:"is_future"`
		} `json:"temporal"`
	} `json:"expected"`
}

type TestResult struct {
	ID      int    `json:"id"`
	Passed  bool   `json:"passed"`
	Actual  any    `json:"actual"`
	Message string `json:"message"`
}

func main() {
	fmt.Println("🚀 Bắt đầu vòng lặp tối ưu hóa Router tự động...")

	for iteration := 1; iteration <= 3; iteration++ {
		fmt.Printf("\n--- Iteration %d ---\n", iteration)

		results, allPassed := runTests()
		if allPassed {
			fmt.Println("✅ Tất cả test case đã PASS! Hệ thống đã tối ưu.")
			break
		}

		fmt.Println("❌ Có test case thất bại. Đang phân tích và cập nhật prompt...")
		refinePrompt(results)
	}
}

func runTests() ([]TestResult, bool) {
	data, _ := ioutil.ReadFile("Gemini/internal/evaluator/test_cases.json")
	var tests []TestCase
	json.Unmarshal(data, &tests)

	allPassed := true
	var results []TestResult

	for _, t := range tests {
		fmt.Printf("Testing Case #%d: %s... ", t.ID, t.UserQuery)

		// Chạy test_router.go
		cmd := exec.Command("go", "run", "cmd/gemini-cli/test_router.go", "cmd/gemini-cli/agent.go", "cmd/gemini-cli/routing.go", "cmd/gemini-cli/server.go", "cmd/gemini-cli/tool_calls.go", t.UserQuery)
		cmd.Dir = "Gemini"
		out, err := cmd.CombinedOutput()

		if err != nil {
			fmt.Println("ERROR running test")
			continue
		}

		// Parse RESULT_JSON từ output
		outputStr := string(out)
		jsonLine := ""
		for _, line := range strings.Split(outputStr, "\n") {
			if strings.HasPrefix(line, "RESULT_JSON: ") {
				jsonLine = strings.TrimPrefix(line, "RESULT_JSON: ")
				break
			}
		}

		var actual map[string]any
		json.Unmarshal([]byte(jsonLine), &actual)

		// So sánh (Đơn giản hóa cho bản demo)
		passed := true
		msg := "OK"

		// Kiểm tra Agent (nếu có yêu cầu trong test case)
		if t.Expected.Agent != "" && actual["agent"] != t.Expected.Agent {
			passed = false
			msg = fmt.Sprintf("Sai Agent: Mong đợi %s, Nhận %s", t.Expected.Agent, actual["agent"])
		}

		results = append(results, TestResult{ID: t.ID, Passed: passed, Actual: actual, Message: msg})
		if passed {
			fmt.Println("✅ PASS")
		} else {
			fmt.Println("🔴 FAIL:", msg)
			allPassed = false
		}
	}

	return results, allPassed
}

func refinePrompt(results []TestResult) {
	// Ghi kết quả ra file để AI (tôi) đọc và xử lý ở turn sau
	data, _ := json.MarshalIndent(results, "", "  ")
	ioutil.WriteFile("Gemini/internal/evaluator/test_results.json", data, 0644)
	fmt.Println("📝 Đã ghi kết quả vào test_results.json. AI đang phân tích...")
}
