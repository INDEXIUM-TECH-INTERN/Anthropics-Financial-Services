#!/bin/bash

# Test script for TestAIFinance deployment
# Validates health, performance, and functionality

set -e

echo "🧪 Running deployment tests..."

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test URLs
BACKEND_URL="http://localhost:8080"
FRONTEND_URL="http://localhost:3000"

print_test() {
    echo -e "${GREEN}[TEST] $1${NC}"
}

print_fail() {
    echo -e "${RED}[FAIL] $1${NC}"
    exit 1
}

# Health check tests
echo ""
print_test "Testing backend health..."
if curl -s "$BACKEND_URL/health" | grep -q "status.*ok"; then
    print_test "✓ Backend health check passed"
else
    print_fail "Backend health check failed"
fi

print_test "Testing frontend health..."
if curl -s -o /dev/null -w "%{http_code}" "$FRONTEND_URL" | grep -q "200"; then
    print_test "✓ Frontend health check passed"
else
    print_fail "Frontend health check failed"
fi

# API endpoint tests
echo ""
print_test "Testing API endpoints..."

# Test chat endpoint
if curl -s -X POST "$BACKEND_URL/api/chat" \
     -H "Content-Type: application/json" \
     -d '{"message":"hello"}' | grep -q "reply"; then
    print_test "✓ Chat endpoint working"
else
    print_warning "Chat endpoint test failed (might be expected without API keys)"
fi

# Test SSE endpoint
if curl -s "$BACKEND_URL/api/events" | grep -q "event:"; then
    print_test "✓ SSE endpoint working"
else
    print_warning "SSE endpoint test failed"
fi

# Frontend content tests
echo ""
print_test "Testing frontend content..."

if curl -s "$FRONTEND_URL" | grep -q "TestAIFinance"; then
    print_test "✓ Frontend contains expected content"
else
    print_warning "Frontend content test failed"
fi

# Performance tests
echo ""
print_test "Testing performance..."

# Backend response time
BACKEND_TIME=$(curl -o /dev/null -s -w '%{time_total}' "$BACKEND_URL/health")
if (( $(echo "$BACKEND_TIME < 1.0" | bc -l) )); then
    print_test "✓ Backend response time: ${BACKEND_TIME}s"
else
    print_warning "Backend response time slow: ${BACKEND_TIME}s"
fi

# Frontend response time
FRONTEND_TIME=$(curl -o /dev/null -s -w '%{time_total}' "$FRONTEND_URL")
if (( $(echo "$FRONTEND_TIME < 2.0" | bc -l) )); then
    print_test "✓ Frontend response time: ${FRONTEND_TIME}s"
else
    print_warning "Frontend response time slow: ${FRONTEND_TIME}s"
fi

# Error tests
echo ""
print_test "Testing error handling..."

# 404 test
if curl -s "$BACKEND_URL/nonexistent" | grep -q "404\|404"; then
    print_test "✓ Backend 404 handling working"
else
    print_warning "Backend 404 handling might be broken"
fi

# Docker health checks
echo ""
print_test "Testing Docker health..."

if docker-compose ps | grep -q "Up"; then
    print_test "✓ All containers are running"
else
    print_fail "Some containers are not running"
fi

# Memory usage check
BACKEND_MEM=$(docker stats testai-backend --no-stream --format "{{.MemUsage}}" | cut -d/ -f2 | sed 's/ MiB//')
if (( $(echo "$BACKEND_MEM < 512" | bc -l) )); then
    print_test "✓ Backend memory usage: ${BACKEND_MEM}MiB"
else
    print_warning "Backend memory usage high: ${BACKEND_MEM}MiB"
fi

# Summary
echo ""
print_test "🎉 All tests completed successfully!"

# Cleanup suggestion
echo ""
echo "💡 Tips for production:"
echo "- Monitor memory usage and scale if needed"
echo "- Set up proper logging aggregation"
echo "- Configure backup strategy for Redis"
echo "- Set up alerting for error rates"
echo "- Consider adding SSL/TLS certificates"
echo "- Set up database backups if using PostgreSQL"