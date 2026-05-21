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
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	envPaths := []string{
		".env",
		filepath.Join(exeDir, ".env"),
		filepath.Join(exeDir, "Gemini", ".env"),
		ResolveProjectPath(".env"),
		filepath.Join(filepath.Dir(ResolveProjectPath(".")), ".env"),
	}

	for _, envPath := range envPaths {
		file, err := os.Open(envPath)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		found := false
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
					found = true
				}
			}
		}
		file.Close()
		if found {
			return
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
