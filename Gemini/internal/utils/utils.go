package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// --- Các hàm tiện ích ---

func LoadEnv() {
	envPath := ".env"
	file, err := os.Open(envPath)
	if err != nil {
		// Fallback to absolute path if running from different location but prefer CWD
		exePath, _ := os.Executable()
		envPath = filepath.Join(filepath.Dir(exePath), ".env")
		file, err = os.Open(envPath)
		if err != nil {
			return
		}
	}
	defer file.Close()

	fmt.Printf("📂 [Config] Loading environment from: %s\n", envPath)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimPrefix(line, "\ufeff")
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)

			if key != "" {
				os.Setenv(key, val)
			}
		}
	}
}

func ResolveProjectPath(rel string) string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return rel
	}

	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
	return filepath.Join(moduleRoot, rel)
}

func LoadPrompt(name string) string {
	path := ResolveProjectPath(filepath.Join("internal", "prompt", name))
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("⚠️ [Prompt] Không đọc được file prompt %s: %v\n", path, err)
		return ""
	}

	return strings.TrimSpace(string(content))
}

func RenderPromptTemplate(name string, replacements map[string]string) string {
	prompt := LoadPrompt(name)
	for key, value := range replacements {
		prompt = strings.ReplaceAll(prompt, "{{"+key+"}}", value)
	}

	return prompt
}

func TranslateWeekday(wd time.Weekday) string {
	days := map[time.Weekday]string{
		time.Sunday:    "Chủ Nhật",
		time.Monday:    "Thứ Hai",
		time.Tuesday:   "Thứ Ba",
		time.Wednesday: "Thứ Tư",
		time.Thursday:  "Thứ Năm",
		time.Friday:    "Thứ Sáu",
		time.Saturday:  "Thứ Bảy",
	}
	return days[wd]
}
