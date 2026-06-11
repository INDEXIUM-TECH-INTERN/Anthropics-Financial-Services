package market

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractTicker(t *testing.T) {
	tests := []struct {
		query      string
		expected   string
		expectedOk bool
	}{
		{"Xem thông tin FPT hôm nay", "FPT", true},
		{"Báo cáo tài chính VNM quý 1", "VNM", true},
		{"Mã chứng khoán hdb.vn có gì mới?", "HDB", true},
		{"ACB và HPG tăng trưởng thế nào?", "ACB", true}, // Match first
		{"Không có mã nào ở đây cả", "", false},
		{"Chứng khoán VN30", "", false},
		{"FPTGROUP", "", false}, // No word boundary match
	}

	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			ticker, ok := extractTicker(tc.query)
			if ok != tc.expectedOk {
				t.Errorf("expected ok=%v, got=%v", tc.expectedOk, ok)
			}
			if ticker != tc.expected {
				t.Errorf("expected ticker=%q, got=%q", tc.expected, ticker)
			}
		})
	}
}

func TestFetchYahooFinanceDataFallback(t *testing.T) {
	// Start local mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "FPT.VN") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{
				"chart": {
					"result": [
						{
							"meta": {
								"symbol": "FPT.VN",
								"exchangeName": "HOSE",
								"regularMarketPrice": 128500.0,
								"chartPreviousClose": 128200.0,
								"currency": "VND"
							}
						}
					]
				}
			}`)
		} else if strings.Contains(r.URL.Path, "VNM") {
			// Mock plain VNM fallback
			if strings.Contains(r.URL.Path, "VNM.VN") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{
				"chart": {
					"result": [
						{
							"meta": {
								"symbol": "VNM",
								"exchangeName": "HOSE",
								"regularMarketPrice": 68000.0,
								"chartPreviousClose": 69000.0,
								"currency": "VND"
							}
						}
					]
				}
			}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Redirect base URL to our mock server
	oldBase := yahooBaseURL
	yahooBaseURL = server.URL
	defer func() { yahooBaseURL = oldBase }()

	// Test FPT.VN path
	resFPT := FetchYahooFinanceDataFallback("FPT")
	if !strings.Contains(resFPT, "stock: FPT.VN") || !strings.Contains(resFPT, "price: 128500.00") || !strings.Contains(resFPT, "+300.00 (+0.23%)") {
		t.Errorf("FPT fallback result unexpected: %s", resFPT)
	}

	// Test VNM plain fallback path
	resVNM := FetchYahooFinanceDataFallback("VNM")
	if !strings.Contains(resVNM, "stock: VNM") || !strings.Contains(resVNM, "price: 68000.00") || !strings.Contains(resVNM, "-1000.00 (-1.45%)") {
		t.Errorf("VNM fallback result unexpected: %s", resVNM)
	}

	// Test failure path
	resFail := FetchYahooFinanceDataFallback("ACB")
	if !strings.Contains(resFail, "Lỗi:") {
		t.Errorf("expected error string, got: %s", resFail)
	}
}

func TestSearchDuckDuckGoFree(t *testing.T) {
	// Start local mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `
			<!DOCTYPE html>
			<html>
			<body>
			<div class="result results_links results_links_deep web-result ">
			  <a class="result__a" href="https://example.com/fpt"><b>FPT</b> Group</a>
			  <a class="result__snippet" href="https://example.com/fpt">FPT Corporation is a leading IT services company in Vietnam.</a>
			</div>
			<div class="result results_links results_links_deep web-result ">
			  <a class="result__a" href="/l/?kh=-1&uddg=https%3A%2F%2Fexample.com%2Fvnm">Vinamilk</a>
			  <a class="result__snippet" href="/l/?kh=-1&uddg=https%3A%2F%2Fexample.com%2Fvnm">Vietnam Dairy Products Joint Stock Company.</a>
			</div>
			</body>
			</html>
		`)
	}))
	defer server.Close()

	oldBase := ddgBaseURL
	ddgBaseURL = server.URL
	defer func() { ddgBaseURL = oldBase }()

	res := SearchDuckDuckGoFree("FPT VNM")
	if !strings.Contains(res, "Search Results:") {
		t.Errorf("expected Search Results header, got: %s", res)
	}
	if !strings.Contains(res, "FPT Group [URL: https://example.com/fpt]: FPT Corporation is a leading IT services company in Vietnam.") {
		t.Errorf("first result missing or wrong: %s", res)
	}
	if !strings.Contains(res, "Vinamilk [URL: https://example.com/vnm]: Vietnam Dairy Products Joint Stock Company.") {
		t.Errorf("second result (uddg clean) missing or wrong: %s", res)
	}
}

func TestExecuteFreeSearchFallback(t *testing.T) {
	// Start local mock server
	serverYahoo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{
			"chart": {
				"result": [
					{
						"meta": {
							"symbol": "FPT.VN",
							"exchangeName": "HOSE",
							"regularMarketPrice": 128500.0,
							"chartPreviousClose": 128200.0,
							"currency": "VND"
						}
					}
				]
			}
		}`)
	}))
	defer serverYahoo.Close()

	serverDDG := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `
			<div class="result">
			  <a class="result__a" href="https://example.com/news">FPT News</a>
			  <a class="result__snippet" href="https://example.com/news">FPT stock breaks all-time high.</a>
			</div>
		`)
	}))
	defer serverDDG.Close()

	oldYahoo := yahooBaseURL
	oldDDG := ddgBaseURL
	yahooBaseURL = serverYahoo.URL
	ddgBaseURL = serverDDG.URL
	defer func() {
		yahooBaseURL = oldYahoo
		ddgBaseURL = oldDDG
	}()

	res := ExecuteFreeSearchFallback("Xem giá cổ phiếu FPT hôm nay")
	if !strings.Contains(res, "Stock Information:") {
		t.Errorf("expected Stock Information block in fallback: %s", res)
	}
	if !strings.Contains(res, "Search Results:") {
		t.Errorf("expected Search Results block in fallback: %s", res)
	}
}
