#!/bin/bash

# Adminer Security Test Script
# Tests the security implementation of Adminer access

echo "🔐 Adminer Security Test Suite"
echo "================================"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
PASSED=0
FAILED=0

# Function to test
test_case() {
    local name=$1
    local command=$2
    local expected=$3
    
    echo -n "Testing: $name ... "
    
    result=$(eval $command 2>&1)
    
    if echo "$result" | grep -q "$expected"; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAILED${NC}"
        echo "  Expected: $expected"
        echo "  Got: $result"
        ((FAILED++))
    fi
}

echo "1. Service Availability Tests"
echo "------------------------------"

test_case "OTA service is running" \
    "curl -s -o /dev/null -w '%{http_code}' http://localhost:9999/login" \
    "200"

test_case "Forwarder service is running" \
    "curl -s -o /dev/null -w '%{http_code}' http://localhost:8888/login" \
    "200"

test_case "Adminer Proxy is running" \
    "curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/health" \
    "200"

echo ""
echo "2. Security Tests"
echo "-----------------"

test_case "Direct Adminer access redirects to login" \
    "curl -s -I http://localhost:8080 | grep -i location" \
    "login"

test_case "Adminer Proxy health endpoint is accessible" \
    "curl -s http://localhost:8080/health" \
    "OK"

test_case "OTA /adminer endpoint exists" \
    "curl -s -o /dev/null -w '%{http_code}' http://localhost:9999/adminer" \
    "302"

test_case "Forwarder /adminer endpoint exists" \
    "curl -s -o /dev/null -w '%{http_code}' http://localhost:8888/adminer" \
    "302"

echo ""
echo "3. Docker Container Tests"
echo "-------------------------"

test_case "Adminer container is running" \
    "docker ps | grep adminer" \
    "adminer"

test_case "Adminer Proxy container is running" \
    "docker ps | grep adminer-proxy" \
    "adminer-proxy"

test_case "OTA container is running" \
    "docker ps | grep ota-app" \
    "ota-app"

test_case "Forwarder container is running" \
    "docker ps | grep forwarder" \
    "forwarder"

echo ""
echo "4. Network Tests"
echo "----------------"

test_case "Adminer is NOT exposed externally" \
    "docker port adminer 2>&1" \
    "Error"

test_case "Adminer Proxy IS exposed on 8080" \
    "docker port adminer-proxy | grep 8080" \
    "8080"

test_case "Services are on same network" \
    "docker network inspect servfor_iot-net | grep -c 'adminer'" \
    "2"

echo ""
echo "5. Configuration Tests"
echo "----------------------"

test_case "OTA has Telegram bot token configured" \
    "docker exec ota-app printenv TELE_BOT_OTA" \
    "AAE"

test_case "Forwarder has Telegram bot token configured" \
    "docker exec forwarder printenv TELE_BOT_ALRT" \
    "AAE"

test_case "Telegram Chat ID is configured" \
    "docker exec ota-app printenv TELEGRAM_CHAT_ID" \
    "[0-9]"

echo ""
echo "================================"
echo "Test Results Summary"
echo "================================"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed! Security implementation is working correctly.${NC}"
    exit 0
else
    echo -e "${RED}⚠️  Some tests failed. Please check the configuration.${NC}"
    exit 1
fi
