package messaging

import "testing"

func TestSanitizeAssistantContentRemovesSkillLeak(t *testing.T) {
	raw := `...to the broader market?

### Step 5: Investment Implications
- Where are the best risk/reward opportunities?
## Important Notes
- Source all market size data

Chào bạn, với tư cách là chuyên viên nghiên cứu thị trường`

	got := SanitizeAssistantContent(raw)
	if contains(got, "Step 5") {
		t.Fatalf("skill leak remained: %q", got)
	}
	if !contains(got, "Chào bạn") {
		t.Fatalf("expected Vietnamese body, got %q", got)
	}
}

func TestIsBootstrapPayload(t *testing.T) {
	if !IsBootstrapPayload("SKILL MARKDOWN (sector-overview)\n### Step 1") {
		t.Fatal("expected bootstrap detection")
	}
	if IsBootstrapPayload("Chào bạn, đây là báo cáo") {
		t.Fatal("expected normal content")
	}
}

func TestFilterPublicHistorySkipsInternal(t *testing.T) {
	history := []Message{
		{Role: RoleUser, Content: "Tổng quan ngành ngân hàng"},
		{Role: RoleUser, Content: "SKILL MARKDOWN (x)", Internal: true},
		{Role: RoleAssistant, Content: "### Step 5:\n\nChào bạn"},
	}
	out := FilterPublicHistory(history)
	if len(out) != 2 {
		t.Fatalf("expected 2 public messages, got %d", len(out))
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || stringIndex(s, sub) >= 0)
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}