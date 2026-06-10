package main

import (
	"encoding/json"
	"fmt"
	"os"
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

func resolvePath(relPath string) string {
	if _, err := os.Stat(relPath); err == nil {
		return relPath
	}
	if strings.HasPrefix(relPath, "Gemini/") {
		noPrefix := strings.TrimPrefix(relPath, "Gemini/")
		if _, err := os.Stat(noPrefix); err == nil {
			return noPrefix
		}
	}
	withPrefix := "Gemini/" + relPath
	if _, err := os.Stat(withPrefix); err == nil {
		return withPrefix
	}
	return relPath
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
	testCasesPath := resolvePath("Gemini/internal/evaluator/test_cases.json")
	data, err := os.ReadFile(testCasesPath)
	if err != nil {
		fmt.Printf("Error reading test cases from %s: %v\n", testCasesPath, err)
		return nil, false
	}

	var tests []TestCase
	json.Unmarshal(data, &tests)

	allPassed := true
	var results []TestResult

	for _, t := range tests {
		fmt.Printf("Testing Case #%d: %s... ", t.ID, t.UserQuery)

		// Run test_router.go with SYSTEM_DATE_OVERRIDE=2026-05-21 to anchor "today"
		cmd := exec.Command("go", "run", "cmd/gemini-cli/test_router.go", t.UserQuery)
		cmd.Env = append(os.Environ(), "SYSTEM_DATE_OVERRIDE=2026-05-21")
		
		if _, err := os.Stat("go.mod"); err == nil {
			cmd.Dir = ""
		} else {
			cmd.Dir = "Gemini"
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("ERROR running test: %v. Output: %s\n", err, string(out))
			allPassed = false
			results = append(results, TestResult{
				ID:      t.ID,
				Passed:  false,
				Message: fmt.Sprintf("Error running test command: %v. Output: %s", err, string(out)),
			})
			continue
		}

		// Parse RESULT_JSON từ output
		outputStr := string(out)
		jsonLine := ""
		for _, line := range strings.Split(outputStr, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "RESULT_JSON: ") {
				jsonLine = strings.TrimPrefix(line, "RESULT_JSON: ")
				break
			}
		}

		if jsonLine == "" {
			fmt.Printf("ERROR: missing RESULT_JSON in output: %s\n", outputStr)
			allPassed = false
			results = append(results, TestResult{
				ID:      t.ID,
				Passed:  false,
				Message: fmt.Sprintf("Missing RESULT_JSON in test_router output: %s", outputStr),
			})
			continue
		}

		var actual map[string]any
		if err := json.Unmarshal([]byte(jsonLine), &actual); err != nil {
			fmt.Printf("ERROR parsing actual JSON: %v. JSON line: %s\n", err, jsonLine)
			allPassed = false
			results = append(results, TestResult{
				ID:      t.ID,
				Passed:  false,
				Message: fmt.Sprintf("Failed to unmarshal RESULT_JSON: %v", err),
			})
			continue
		}

		passed := true
		msg := "OK"

		// 1. Kiểm tra Agent (nếu được chỉ định mong đợi)
		if t.Expected.Agent != "" {
			actualAgent, _ := actual["agent"].(string)
			if actualAgent != t.Expected.Agent {
				passed = false
				msg = fmt.Sprintf("Sai Agent: Mong đợi '%s', Nhận '%s'", t.Expected.Agent, actualAgent)
			}
		}

		// 2. Kiểm tra Temporal (nếu được chỉ định mong đợi)
		temporal, hasTemporal := actual["temporal"].(map[string]any)
		if passed {
			if t.Expected.Temporal.Intent != "" {
				if !hasTemporal {
					passed = false
					msg = "Thiếu thông tin Temporal trong kết quả thực tế"
				} else {
					actualIntent, _ := temporal["intent"].(string)
					if actualIntent != t.Expected.Temporal.Intent {
						passed = false
						msg = fmt.Sprintf("Sai Temporal Intent: Mong đợi '%s', Nhận '%s'", t.Expected.Temporal.Intent, actualIntent)
					}
				}
			}
		}
		if passed {
			if t.Expected.Temporal.ResolvedDate != "" {
				if !hasTemporal {
					passed = false
					msg = "Thiếu thông tin Temporal trong kết quả thực tế"
				} else {
					actualResolvedDate, _ := temporal["resolved_date"].(string)
					if actualResolvedDate != t.Expected.Temporal.ResolvedDate {
						passed = false
						msg = fmt.Sprintf("Sai Temporal ResolvedDate: Mong đợi '%s', Nhận '%s'", t.Expected.Temporal.ResolvedDate, actualResolvedDate)
					}
				}
			}
		}
		if passed {
			if t.Expected.Temporal.IsFuture {
				if !hasTemporal {
					passed = false
					msg = "Thiếu thông tin Temporal trong kết quả thực tế"
				} else {
					actualIsFuture, _ := temporal["is_future"].(bool)
					if !actualIsFuture {
						passed = false
						msg = "Sai Temporal IsFuture: Mong đợi true, Nhận false"
					}
				}
			}
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
	data, _ := json.MarshalIndent(results, "", "  ")
	testResultsPath := resolvePath("Gemini/internal/evaluator/test_results.json")
	os.WriteFile(testResultsPath, data, 0644)
	fmt.Printf("📝 Đã ghi kết quả vào %s. AI đang phân tích...\n", testResultsPath)
}
