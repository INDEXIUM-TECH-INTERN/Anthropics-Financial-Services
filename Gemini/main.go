package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// --- Cấu trúc dữ liệu Gemini API ---
type GeminiRequest struct {
	SystemInstruction SystemInstruction `json:"system_instruction"`
	Contents          []Content         `json:"contents"`
}

type SystemInstruction struct {
	Parts []Part `json:"parts"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `\"'`)
			os.Setenv(key, val)
		}
	}
}

func main() {
	loadEnv()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ Lỗi: Không tìm thấy GEMINI_API_KEY trong .env")
		return
	}

	repoRoot := `C:\Users\Rabuno\Documents\Repository\financial-services`

	agentBrain := readFile(filepath.Join(repoRoot, "agents", "pitch-agent.md"))
	compsSkill := readFile(filepath.Join(repoRoot, "skills", "comps-analysis", "SKILL.md"))

	if agentBrain == "" || compsSkill == "" {
		fmt.Println("❌ Lỗi: Không đọc được các file hướng dẫn từ Repo.")
		return
	}

	systemInstruction := fmt.Sprintf("%s\n\nCORE SKILL DATA:\n%s", agentBrain, compsSkill)
	userPrompt := "Hãy hướng dẫn tôi các bước so sánh định giá (comps) cho công ty VinFast (VFS) so với các hãng xe điện khác."

	fmt.Println("🚀 Gemini Financial Agent đang xử lý (sử dụng model flash)...")

	payload := GeminiRequest{
		SystemInstruction: SystemInstruction{
			Parts: []Part{{Text: systemInstruction}},
		},
		Contents: []Content{
			{Parts: []Part{{Text: userPrompt}}},
		},
	}

	jsonData, _ := json.Marshal(payload)
	// Đổi từ gemini-1.5-pro sang gemini-1.5-flash để ổn định hơn
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s", apiKey)
	
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Lỗi kết nối: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var geminiResp GeminiResponse
	json.Unmarshal(body, &geminiResp)

	if geminiResp.Error.Message != "" {
		fmt.Printf("❌ Lỗi API: %s\n", geminiResp.Error.Message)
		return
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		fmt.Println("\n✅ --- PHẢN HỒI TỪ GEMINI ---")
		fmt.Println(geminiResp.Candidates[0].Content.Parts[0].Text)
	} else {
		fmt.Println("⚠️ Không có phản hồi từ AI. Nội dung lỗi: " + string(body))
	}
}

func readFile(path string) string {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}
