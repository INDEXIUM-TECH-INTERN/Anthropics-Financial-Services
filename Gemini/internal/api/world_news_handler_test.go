package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleWorldNewsDates(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/world-news/dates", nil)
	rr := httptest.NewRecorder()
	handleWorldNewsDates(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	dates, ok := body["dates"].([]interface{})
	if !ok || len(dates) == 0 {
		t.Fatal("expected dates array")
	}
}

func TestHandleWorldNewsLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live world news handler test")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/world-news", nil)
	rr := httptest.NewRecorder()
	handleWorldNews(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["date"] == nil {
		t.Fatal("missing date")
	}
	if body["stocks"] == nil {
		t.Fatal("missing stocks")
	}
}