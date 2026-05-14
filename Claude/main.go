package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []Message `json:"messages"`
}

type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ Vui lòng set biến môi trường ANTHROPIC_API_KEY")
		fmt.Println("Cách set (Windows PowerShell): $env:ANTHROPIC_API_KEY='sk-ant-...'")
		return
	}

	repoRoot := `C:\Users\Rabuno\Documents\Repository\financial-services`

	agentBrain := readFile(filepath.Join(repoRoot, "agents", "pitch-agent.md"))
	compsSkill := readFile(filepath.Join(repoRoot, "skills", "comps-analysis", "SKILL.md"))

	if agentBrain == "" || compsSkill == "" {
		fmt.Println("❌ Không thể đọc file hướng dẫn từ repo. Vui lòng kiểm tra lại repoRoot trong code.")
		return
	}

	systemInstruction := fmt.Sprintf("%s\n\nCORE SKILL DATA:\n%s", agentBrain, compsSkill)

	userPrompt := "Hãy hướng dẫn tôi các bước so sánh định giá (comps) cho công ty VinFast (VFS) so với các hãng xe điện khác (Tesla, BYD, Rivian)."

	fmt.Println("🚀 Đang kết nối với Claude Financial Agent...")

	payload := AnthropicRequest{
		Model:     "claude-3-5-sonnet-20240620",
		MaxTokens: 2048,
		System:    systemInstruction,
		Messages: []Message{
			{Role: "user", Content: userPrompt},
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Lỗi kết nối API: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var anthropicResp AnthropicResponse
	json.Unmarshal(body, &anthropicResp)

	if anthropicResp.Error.Message != "" {
		fmt.Printf("❌ Lỗi từ Claude API: %s\n", anthropicResp.Error.Message)
		return
	}

	if len(anthropicResp.Content) > 0 {
		fmt.Println("\n✅ --- PHẢN HỒI TỪ CHUYÊN GIA TÀI CHÍNH ---")
		fmt.Println(anthropicResp.Content[0].Text)
	} else {
		fmt.Println("⚠️ Không nhận được phản hồi.")
	}
}

func readFile(path string) string {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}
