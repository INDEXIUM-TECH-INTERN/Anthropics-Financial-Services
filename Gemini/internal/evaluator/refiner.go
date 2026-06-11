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
	tests, err := loadTestCases()
	if err != nil {
		fmt.Printf("Error loading test cases: %v\n", err)
		return nil, false
	}

	var results []TestResult
	allPassed := true

	for _, t := range tests {
		result := runSingleTest(t)
		results = append(results, result)
		if !result.Passed {
			allPassed = false
		}
	}

	return results, allPassed
}

func loadTestCases() ([]TestCase, error) {
	testCasesPath := resolvePath("Gemini/internal/evaluator/test_cases.json")
	data, err := os.ReadFile(testCasesPath)
	if err != nil {
		return nil, fmt.Errorf("reading test cases from %s: %w", testCasesPath, err)
	}

	var tests []TestCase
	if err := json.Unmarshal(data, &tests); err != nil {
		return nil, fmt.Errorf("parsing test cases: %w", err)
	}
	return tests, nil
}

func runSingleTest(t TestCase) TestResult {
	fmt.Printf("Testing Case #%d: %s... ", t.ID, t.UserQuery)

	actual, err := executeTestCommand(t.UserQuery)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return TestResult{ID: t.ID, Passed: false, Message: err.Error()}
	}

	passed, msg := validateResult(t, actual)
	if passed {
		fmt.Println("✅ PASS")
	} else {
		fmt.Println("🔴 FAIL:", msg)
	}

	return TestResult{ID: t.ID, Passed: passed, Actual: actual, Message: msg}
}

func executeTestCommand(query string) (map[string]any, error) {
	cmd := exec.Command("go", "run", "cmd/gemini-cli/test_router.go", query)
	cmd.Env = append(os.Environ(), "SYSTEM_DATE_OVERRIDE=2026-05-21")

	if _, err := os.Stat("go.mod"); err != nil {
		cmd.Dir = "Gemini"
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("running test: %v. Output: %s", err, string(out))
	}

	jsonLine := extractJSONFromOutput(string(out))
	if jsonLine == "" {
		return nil, fmt.Errorf("missing RESULT_JSON in output: %s", string(out))
	}

	var actual map[string]any
	if err := json.Unmarshal([]byte(jsonLine), &actual); err != nil {
		return nil, fmt.Errorf("parsing RESULT_JSON: %v", err)
	}
	return actual, nil
}

func extractJSONFromOutput(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "RESULT_JSON: ") {
			return strings.TrimPrefix(line, "RESULT_JSON: ")
		}
	}
	return ""
}

func validateResult(t TestCase, actual map[string]any) (bool, string) {
	if t.Expected.Agent != "" {
		if msg := validateAgent(t, actual); msg != "" {
			return false, msg
		}
	}

	temporal, hasTemporal := actual["temporal"].(map[string]any)
	if t.Expected.Temporal.Intent != "" || t.Expected.Temporal.ResolvedDate != "" || t.Expected.Temporal.IsFuture {
		if msg := validateTemporal(t, temporal, hasTemporal); msg != "" {
			return false, msg
		}
	}

	return true, "OK"
}

func validateAgent(t TestCase, actual map[string]any) string {
	actualAgent, _ := actual["agent"].(string)
	if actualAgent != t.Expected.Agent {
		return fmt.Sprintf("Sai Agent: Mong đợi '%s', Nhận '%s'", t.Expected.Agent, actualAgent)
	}
	return ""
}

func validateTemporal(t TestCase, temporal map[string]any, hasTemporal bool) string {
	if !hasTemporal {
		return "Thiếu thông tin Temporal trong kết quả thực tế"
	}

	if t.Expected.Temporal.Intent != "" {
		actualIntent, _ := temporal["intent"].(string)
		if actualIntent != t.Expected.Temporal.Intent {
			return fmt.Sprintf("Sai Temporal Intent: Mong đợi '%s', Nhận '%s'", t.Expected.Temporal.Intent, actualIntent)
		}
	}
	if t.Expected.Temporal.ResolvedDate != "" {
		actualDate, _ := temporal["resolved_date"].(string)
		if actualDate != t.Expected.Temporal.ResolvedDate {
			return fmt.Sprintf("Sai Temporal ResolvedDate: Mong đợi '%s', Nhận '%s'", t.Expected.Temporal.ResolvedDate, actualDate)
		}
	}
	if t.Expected.Temporal.IsFuture {
		actualIsFuture, _ := temporal["is_future"].(bool)
		if !actualIsFuture {
			return "Sai Temporal IsFuture: Mong đợi true, Nhận false"
		}
	}
	return ""
}

func refinePrompt(results []TestResult) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Printf("❌ Lỗi marshal results: %v\n", err)
		return
	}
	testResultsPath := resolvePath("Gemini/internal/evaluator/test_results.json")
	if err := os.WriteFile(testResultsPath, data, 0644); err != nil {
		fmt.Printf("❌ Lỗi ghi file: %v\n", err)
		return
	}
	fmt.Printf("📝 Đã ghi kết quả vào %s. AI đang phân tích...\n", testResultsPath)
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
