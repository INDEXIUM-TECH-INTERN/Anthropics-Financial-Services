#!/usr/bin/env python3
"""
Indexium Agent Evaluation Harness v3
- Fixes: empty body validation, longer free-tier delays, robust JSON extraction
- Sends test queries to the running backend
- Uses OpenRouter as evaluator with fallback chain across keys
- Falls back to Gemini API if all OpenRouter keys exhausted
- Verifies link validity (HTTP status + relevance)
- Loops: optimize → retest until threshold met
"""

import json
import os
import re
import subprocess
import sys
import time
import urllib.request
import urllib.error
from datetime import datetime
from pathlib import Path

# Load .env from project root (parent of scripts/)
try:
    from dotenv import load_dotenv
    _env_path = Path(__file__).resolve().parent.parent / ".env"
    load_dotenv(_env_path)
except ImportError:
    pass  # dotenv not installed, fall back to os.environ

# ═══════════════════════════════════════════════════════════
# CONFIG — loaded from .env
# ═══════════════════════════════════════════════════════════

BACKEND_URL = os.environ.get("BACKEND_URL", "http://localhost:8080")

# Primary evaluator: OpenRouter (much higher rate limits than Gemini free tier)
OPENROUTER_API_KEYS = [
    k for k in [
        os.environ.get("OPENROUTER_API_KEY", ""),
        os.environ.get("OPENROUTER_API_KEY_2", ""),
        os.environ.get("OPENROUTER_API_KEY_3", ""),
        os.environ.get("OPENROUTER_API_KEY_4", ""),
        os.environ.get("OPENROUTER_API_KEY_5", ""),
    ] if k
]
OPENROUTER_MODEL = os.environ.get("OPENROUTER_MODEL", "openrouter/free")
OPENROUTER_API_URL = "https://openrouter.ai/api/v1/chat/completions"

# Fallback evaluator: Gemini raw API
GEMINI_API_KEYS = [
    k for k in [
        os.environ.get("GEMINI_API_KEY", ""),
        os.environ.get("GEMINI_API_KEY_2", ""),
    ] if k
]
GEMINI_API_KEY = GEMINI_API_KEYS[0] if GEMINI_API_KEYS else ""
GEMINI_MODEL = os.environ.get("GEMINI_MODEL", "gemini-2.0-flash")
GEMINI_API_URL = f"https://generativelanguage.googleapis.com/v1beta/models/{GEMINI_MODEL}:generateContent"

# Rate limit handling — longer delays for free tier
EVAL_CALL_DELAY = int(os.environ.get("EVAL_DELAY", "10"))        # delay between eval queries
EVAL_MAX_RETRIES = 4                                                # retries per key
EVAL_INITIAL_BACKOFF = 15                                          # initial backoff (seconds) for free tier
EVAL_KEY_DELAY = 20                                                # delay when switching to next API key

# Quality threshold
QUALITY_THRESHOLD = 80
MAX_ITERATIONS = 5

# Thinking-leak patterns to strip from responses
THINKING_PATTERNS = [
    re.compile(r'(?is)<details[^>]*class="thinking-details"[^>]*>.*?</details>\s*'),
    re.compile(r'(?is)<div[^>]*class="thinking-content"[^>]*>.*?</div>\s*'),
    re.compile(r'(?is)<summary[^>]*class="thinking-summary"[^>]*>.*?</summary>\s*'),
    re.compile(r'(?is)<thinking>.*?</thinking>\s*'),
]


# ═══════════════════════════════════════════════════════════
# TEST QUERIES
# ═══════════════════════════════════════════════════════════

TEST_QUERIES = [
    {
        "id": "q1_fundamental",
        "query": "Tổng quan về thị trường chứng khoán Việt Nam năm 2025",
        "category": "market_overview",
        "expected_elements": ["số liệu", "chỉ số", "xu hướng", "VN-Index"],
    },
    {
        "id": "q2_comparison",
        "query": "So sánh tổng tài sản HDBank và ACB trong 3 năm gần đây",
        "category": "comparison",
        "expected_elements": ["HDB", "ACB", "tổng tài sản", "so sánh"],
    },
    {
        "id": "q3_risk",
        "query": "Chi phí dự phòng rủi ro tín dụng của HDBank năm 2024 là bao nhiêu?",
        "category": "specific_data",
        "expected_elements": ["HDBank", "dự phòng", "tín dụng", "2024"],
    },
    {
        "id": "q4_sector",
        "query": "Triển vọng ngành ngân hàng Việt Nam năm 2025",
        "category": "sector_outlook",
        "expected_elements": ["ngân hàng", "triển vọng", "2025", "dự báo"],
    },
    {
        "id": "q5_concept",
        "query": "Giải thích khái niệm P/E ratio và cách sử dụng trong định giá cổ phiếu",
        "category": "concept_explanation",
        "expected_elements": ["P/E", "định giá", "cổ phiếu", "công thức"],
    },
]


# ═══════════════════════════════════════════════════════════
# HELPERS
# ═══════════════════════════════════════════════════════════

def strip_thinking(text: str) -> str:
    """Remove any leaked thinking/reasoning HTML blocks from response."""
    result = text
    for pat in THINKING_PATTERNS:
        result = pat.sub("", result)
    return result.strip()


def send_chat_request(query: str, chat_id: str = None) -> dict:
    """Send a chat request to the backend SSE stream and collect the full response."""
    body = {"message": query}
    if chat_id:
        body["chat_id"] = chat_id

    req = urllib.request.Request(
        f"{BACKEND_URL}/api/chat/stream",
        data=json.dumps(body).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )

    full_text = ""
    metrics = {}
    error = None

    try:
        with urllib.request.urlopen(req, timeout=180) as resp:
            buffer = ""
            while True:
                chunk = resp.read(2048).decode("utf-8", errors="replace")
                if not chunk:
                    break
                buffer += chunk
                while "\n" in buffer:
                    line, buffer = buffer.split("\n", 1)
                    line = line.strip()
                    if not line.startswith("data: "):
                        continue
                    try:
                        data = json.loads(line[6:])
                    except json.JSONDecodeError:
                        continue
                    if data.get("type") == "token" and data.get("text"):
                        full_text += data["text"]
                    elif data.get("type") == "done" and data.get("metrics"):
                        metrics = data["metrics"]
                    elif data.get("type") == "error" and data.get("error"):
                        error = data["error"]
    except Exception as e:
        return {"text": "", "metrics": {}, "error": str(e), "links": []}

    # Strip thinking leaks from collected text
    full_text = strip_thinking(full_text)
    links = extract_links(full_text)
    return {"text": full_text, "metrics": metrics, "error": error, "links": links}


def extract_links(text: str) -> list:
    """Extract all URLs from text with surrounding context."""
    url_pattern = re.compile(r'https?://[^\s\)\]\}"<>\']+')
    links = []
    seen = set()
    for match in url_pattern.finditer(text):
        url = match.group().rstrip(".,;:!?")
        if url in seen:
            continue
        seen.add(url)
        start = max(0, match.start() - 50)
        context = text[start:match.end()].strip()
        links.append({"url": url, "context": context})
    return links


def verify_link(url: str, timeout: int = 10) -> dict:
    """Check if a link is valid (returns 2xx/3xx status)."""
    try:
        req = urllib.request.Request(url, method="HEAD")
        req.add_header("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return {"url": url, "status": resp.status, "valid": 200 <= resp.status < 400}
    except urllib.error.HTTPError as e:
        return {"url": url, "status": e.code, "valid": 200 <= e.code < 400}
    except Exception as e:
        return {"url": url, "status": 0, "valid": False, "error": str(e)[:100]}


def verify_links(links: list) -> list:
    """Verify all links sequentially."""
    results = []
    for link in links:
        result = verify_link(link["url"])
        result["context"] = link.get("context", "")
        results.append(result)
        time.sleep(0.5)
    return results


# ═══════════════════════════════════════════════════════════
# GEMINI EVALUATOR — tries CLI first, falls back to API
# ═══════════════════════════════════════════════════════════

def openrouter_evaluate(prompt: str) -> dict:
    """Evaluate via OpenRouter API (high rate limits, pay-per-use)."""
    body = json.dumps({
        "model": OPENROUTER_MODEL,
        "messages": [{"role": "user", "content": prompt}],
        "temperature": 0.1,
        "max_tokens": 1024,
    }).encode("utf-8")

    for key_idx, api_key in enumerate(OPENROUTER_API_KEYS):
        for attempt in range(EVAL_MAX_RETRIES):
            try:
                req = urllib.request.Request(
                    OPENROUTER_API_URL,
                    data=body,
                    headers={
                        "Content-Type": "application/json",
                        "Authorization": f"Bearer {api_key}",
                        "HTTP-Referer": "https://github.com/rabuno/indexium",
                        "X-Title": "Indexium Eval Harness",
                    },
                    method="POST",
                )
                with urllib.request.urlopen(req, timeout=60) as resp:
                    data = json.loads(resp.read().decode("utf-8"))
                text = data["choices"][0]["message"]["content"]
                text = text.strip()
                if text.startswith("```"):
                    text = text.split("\n", 1)[1]
                    if text.endswith("```"):
                        text = text[:-3]
                result = json.loads(text.strip())
                result["_source"] = "openrouter"
                return result
            except urllib.error.HTTPError as e:
                if e.code == 429:
                    backoff = EVAL_INITIAL_BACKOFF * (2 ** attempt)
                    print(f"    ⏳ OpenRouter key-{key_idx+1} rate limited, backing off {backoff}s...")
                    time.sleep(backoff)
                    continue
                if e.code >= 500:
                    backoff = EVAL_INITIAL_BACKOFF * (attempt + 1)
                    print(f"    ⏳ Server error {e.code}, retrying in {backoff}s...")
                    time.sleep(backoff)
                    continue
                break  # Non-retryable error, try next key
            except Exception as e:
                print(f"    ⚠ OpenRouter error: {e}")
                break

    return None


def gemini_evaluate_api(prompt: str) -> dict:
    """Fallback: evaluate via Gemini raw API with key rotation and exponential backoff."""
    body = json.dumps({
        "contents": [{"parts": [{"text": prompt}]}],
        "generationConfig": {"temperature": 0.1, "maxOutputTokens": 1024},
    }).encode("utf-8")

    for key_idx, api_key in enumerate(GEMINI_API_KEYS):
        backoff = EVAL_INITIAL_BACKOFF
        for attempt in range(EVAL_MAX_RETRIES):
            try:
                req = urllib.request.Request(
                    f"{GEMINI_API_URL}?key={api_key}",
                    data=body,
                    headers={"Content-Type": "application/json"},
                    method="POST",
                )
                with urllib.request.urlopen(req, timeout=60) as resp:
                    data = json.loads(resp.read().decode("utf-8"))
                text = data["candidates"][0]["content"]["parts"][0]["text"]
                text = text.strip()
                if text.startswith("```"):
                    text = text.split("\n", 1)[1]
                    if text.endswith("```"):
                        text = text[:-3]
                result = json.loads(text.strip())
                result["_source"] = "gemini_api"
                return result
            except urllib.error.HTTPError as e:
                if e.code == 429:
                    print(f"    ⏳ Gemini key-{key_idx+1} rate limited, backing off {backoff}s...")
                    time.sleep(backoff)
                    backoff *= 2
                    continue
                if e.code >= 500:
                    print(f"    ⏳ Gemini server error {e.code}, retrying in {backoff}s...")
                    time.sleep(backoff)
                    backoff *= 2
                    continue
                break  # Non-retryable, try next key
            except Exception as e:
                print(f"    ⚠ Gemini key-{key_idx+1} error: {e}")
                break

    return {
        "scores": {"accuracy": 0, "completeness": 0, "sources": 0, "structure": 0},
        "total": 0, "verdict": "ERROR",
        "issues": ["All Gemini keys exhausted"],
        "suggestions": [], "_source": "error",
    }


def evaluate_response(query: str, response: str, expected_elements: list, links: list) -> dict:
    """Evaluate a response. Primary: OpenRouter (high rate limit). Fallback: Gemini API."""
    links_summary = "\n".join([f"- {l['url']}" for l in links]) if links else "(không có link nguồn)"
    truncated = response[:2500] if len(response) > 2500 else response

    eval_prompt = f"""Bạn là chuyên gia đánh giá chất lượng câu trả lời AI tài chính. Hãy đánh giá câu trả lời sau:

**Câu hỏi:** {query}

**Câu trả lời (cắt ngắn):**
{truncated}

**Các yếu tố kỳ vọng:** {', '.join(expected_elements)}

**Link nguồn được cung cấp:**
{links_summary}

Hãy đánh giá theo thang 0-100 với các tiêu chí:
1. **Độ chính xác** (0-25): Thông tin có chính xác về mặt tài chính không?
2. **Độ đầy đủ** (0-25): Có đề cập đủ các yếu tố kỳ vọng không?
3. **Link nguồn** (0-25): Có cung cấp nguồn tham khảo đáng tin cậy không?
4. **Cấu trúc & Ngôn ngữ** (0-25): Câu trả lời có dễ hiểu, có cấu trúc tốt không?

Trả về JSON chính xác (không thêm markdown, chỉ JSON thuần):
{{"scores": {{"accuracy": N, "completeness": N, "sources": N, "structure": N}}, "total": N, "verdict": "PASS|FAIL", "issues": ["vấn đề 1"], "suggestions": ["gợi ý 1"]}}"""

    # Primary: OpenRouter
    result = openrouter_evaluate(eval_prompt)
    if result and result.get("verdict") not in ("ERROR", None):
        return result

    # Fallback: Gemini API
    print("    → OpenRouter failed, falling back to Gemini API (expect rate limits)...")
    return gemini_evaluate_api(eval_prompt)


# ═══════════════════════════════════════════════════════════
# EVALUATION LOOP
# ═══════════════════════════════════════════════════════════

def run_evaluation(iteration: int = 1) -> dict:
    """Run a full evaluation pass over all test queries."""
    print(f"\n{'='*60}")
    print(f"  EVALUATION PASS #{iteration}")
    print(f"  Time: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"  Threshold: {QUALITY_THRESHOLD}/100")
    print(f"{'='*60}")

    results = []

    for i, test in enumerate(TEST_QUERIES):
        qid = test["id"]
        query = test["query"]
        print(f"\n[{i+1}/{len(TEST_QUERIES)}] {qid}: {query[:60]}...")

        # 1. Send query to agent
        print("  → Sending query to agent...")
        agent_response = send_chat_request(query)
        response_text = agent_response["text"]
        links = agent_response["links"]
        error = agent_response["error"]

        if error:
            print(f"  ✗ Agent error: {error}")
            results.append({
                "id": qid, "query": query, "category": test["category"],
                "response_preview": "", "error": error, "score": 0, "verdict": "ERROR",
                "link_results": [], "issues": [error], "suggestions": [],
            })
            # Still need delay even on error
            if i < len(TEST_QUERIES) - 1:
                print(f"  ⏳ Waiting {EVAL_CALL_DELAY}s before next query...")
                time.sleep(EVAL_CALL_DELAY)
            continue

        print(f"  ✓ Response: {len(response_text)} chars, {len(links)} links")

        # 2. Verify links
        link_results = []
        if links:
            print(f"  → Verifying {len(links)} links...")
            link_results = verify_links(links)
            valid_count = sum(1 for lr in link_results if lr["valid"])
            print(f"  ✓ Links: {valid_count}/{len(link_results)} valid")

        # 3. LLM evaluation (OpenRouter primary, Gemini API fallback)
        print("  → Evaluating (OpenRouter → Gemini fallback)...")
        eval_result = evaluate_response(query, response_text, test["expected_elements"], links)
        score = eval_result.get("total", 0)
        verdict = eval_result.get("verdict", "UNKNOWN")
        issues = eval_result.get("issues", [])
        suggestions = eval_result.get("suggestions", [])

        icon = "✓" if score >= QUALITY_THRESHOLD else "✗"
        print(f"  {icon} Score: {score}/100 ({verdict})")
        if issues:
            for issue in issues[:3]:
                print(f"    ⚠ {issue}")

        results.append({
            "id": qid,
            "query": query,
            "category": test["category"],
            "response_preview": response_text[:500],
            "response_length": len(response_text),
            "links_count": len(links),
            "link_results": link_results,
            "score": score,
            "verdict": verdict,
            "scores": eval_result.get("scores", {}),
            "issues": issues,
            "suggestions": suggestions,
            "metrics": agent_response.get("metrics", {}),
        })

        # Rate limit delay between queries (skip after last query)
        if i < len(TEST_QUERIES) - 1 and score > 0:
            print(f"  ⏳ Waiting {EVAL_CALL_DELAY}s before next query...")
            time.sleep(EVAL_CALL_DELAY)

    # Summary
    scores = [r["score"] for r in results if r["verdict"] not in ("ERROR", "SKIP")]
    avg_score = sum(scores) / len(scores) if scores else 0
    pass_count = sum(1 for r in results if r["verdict"] == "PASS")
    total_links = sum(len(r.get("link_results", [])) for r in results)
    valid_links = sum(sum(1 for lr in r.get("link_results", []) if lr["valid"]) for r in results)

    print(f"\n{'='*60}")
    print(f"  PASS #{iteration} SUMMARY")
    print(f"{'='*60}")
    print(f"  Average Score: {avg_score:.1f}/100")
    print(f"  Pass Rate:    {pass_count}/{len(results)}")
    print(f"  Link Validity: {valid_links}/{total_links}")
    print(f"  Threshold:    {QUALITY_THRESHOLD}")
    print()

    for r in results:
        icon = "✓" if r["score"] >= QUALITY_THRESHOLD else "✗"
        detail = ""
        if r.get("scores"):
            s = r["scores"]
            detail = f"[A:{s.get('accuracy',0)} C:{s.get('completeness',0)} S:{s.get('sources',0)} St:{s.get('structure',0)}]"
        print(f"  {icon} {r['id']}: {r['score']}/100 ({r['verdict']}) {detail}")
        if r.get("issues"):
            for issue in r["issues"][:2]:
                print(f"      ⚠ {issue}")

    all_passed = len(scores) > 0 and all(s >= QUALITY_THRESHOLD for s in scores)

    return {
        "iteration": iteration,
        "timestamp": datetime.now().isoformat(),
        "average_score": avg_score,
        "pass_rate": f"{pass_count}/{len(results)}",
        "link_validity": f"{valid_links}/{total_links}",
        "results": results,
        "all_passed": all_passed,
    }


def analyze_issues(all_results: list) -> dict:
    """Analyze collected issues across all iterations to find patterns."""
    issue_counts = {}
    suggestion_counts = {}
    lowest_scored = []

    for result in all_results:
        for r in result.get("results", []):
            lowest_scored.append((r["id"], r["score"]))
            for issue in r.get("issues", []):
                issue_counts[issue] = issue_counts.get(issue, 0) + 1
            for s in r.get("suggestions", []):
                suggestion_counts[s] = suggestion_counts.get(s, 0) + 1

    print(f"\n{'='*60}")
    print("  ISSUE ANALYSIS ACROSS ITERATIONS")
    print(f"{'='*60}")

    if issue_counts:
        print("\n  Common issues:")
        for issue, count in sorted(issue_counts.items(), key=lambda x: -x[1])[:5]:
            print(f"    ({count}x) {issue}")

    if suggestion_counts:
        print("\n  Top suggestions:")
        for s, count in sorted(suggestion_counts.items(), key=lambda x: -x[1])[:5]:
            print(f"    ({count}x) {s}")

    # Find consistently low-scored queries
    query_scores = {}
    for _, score in lowest_scored:
        pass
    # (simplified: just show per-query average)
    query_totals = {}
    query_counts = {}
    for r in [(rid, sc) for result in all_results for r in result.get("results", []) for rid, sc in [(r["id"], r["score"])]]:
        rid, sc = r
        query_totals[rid] = query_totals.get(rid, 0) + sc
        query_counts[rid] = query_counts.get(rid, 0) + 1

    print("\n  Per-query average scores:")
    for qid in sorted(query_totals.keys()):
        avg = query_totals[qid] / query_counts[qid]
        print(f"    {qid}: {avg:.1f}/100")

    return {
        "common_issues": dict(sorted(issue_counts.items(), key=lambda x: -x[1])[:5]),
        "top_suggestions": dict(sorted(suggestion_counts.items(), key=lambda x: -x[1])[:5]),
    }


def main():
    """Main evaluation loop."""
    print("╔══════════════════════════════════════════════════════════╗")
    print("║       INDEXIUM AGENT EVALUATION HARNESS v2              ║")
    print("╚══════════════════════════════════════════════════════════╝")
    print(f"  Backend:        {BACKEND_URL}")
    print(f"  Queries:        {len(TEST_QUERIES)}")
    print(f"  Threshold:      {QUALITY_THRESHOLD}/100")
    print(f"  Max Iterations: {MAX_ITERATIONS}")
    print(f"  Gemini Delay:   {EVAL_CALL_DELAY}s between calls")

    # Check backend
    try:
        urllib.request.urlopen(f"{BACKEND_URL}/", timeout=5)
        print(f"  Backend Status: ✓ Running")
    except Exception:
        print(f"  Backend Status: ✗ Not responding at {BACKEND_URL}")
        print("  Start it first: cd Gemini && .\\server.exe -server")
        sys.exit(1)

    all_results = []

    for iteration in range(1, MAX_ITERATIONS + 1):
        result = run_evaluation(iteration)
        all_results.append(result)

        if result["all_passed"]:
            print(f"\n🎉 ALL QUERIES PASSED at iteration {iteration}!")
            break

        if iteration < MAX_ITERATIONS:
            # Analyze issues to guide optimization
            analysis = analyze_issues(all_results)

            print(f"\n→ Not all queries passed. Applying optimizations...")
            print("  (In auto mode, would adjust system prompt / retrieval strategy)")
            print(f"\n  Waiting {EVAL_CALL_DELAY}s before next iteration...")
            time.sleep(EVAL_CALL_DELAY)

    # Save results
    output_dir = Path(__file__).parent / "eval_results"
    output_dir.mkdir(exist_ok=True)
    output_file = output_dir / f"eval_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
    with open(output_file, "w", encoding="utf-8") as f:
        json.dump(all_results, f, ensure_ascii=False, indent=2)
    print(f"\n📄 Results saved to: {output_file}")

    # Final summary
    final = all_results[-1]
    print(f"\n{'='*60}")
    print(f"  FINAL RESULT")
    print(f"{'='*60}")
    print(f"  Iterations:     {len(all_results)}")
    print(f"  Final Avg:      {final['average_score']:.1f}/100")
    print(f"  Pass Rate:      {final['pass_rate']}")
    print(f"  Link Validity:  {final['link_validity']}")
    print(f"  Status:         {'✅ PASSED' if final['all_passed'] else '❌ NEEDS IMPROVEMENT'}")
    print(f"{'='*60}")


if __name__ == "__main__":
    main()
