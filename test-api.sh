#!/bin/bash

# Comprehensive API Test Script
# Tests all core functionalities in order: Health → Register → Login → Profile → Upload → Videos → Video Info → Comments

set -e  # Exit on any error

echo "🚀 Starting comprehensive API tests..."
echo "======================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
TEST_COUNT=0
PASSED_TESTS=0

function run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TEST_COUNT=$((TEST_COUNT + 1))
    echo -e "\n${BLUE}Test $TEST_COUNT: $test_name${NC}"
    echo "----------------------------------------"
    
    if eval "$test_command"; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        echo -e "${GREEN}✅ PASSED${NC}"
        return 0
    else
        echo -e "${RED}❌ FAILED${NC}"
        return 1
    fi
}

function health_check() {
    curl -f -s -X GET "http://localhost:8080/health" | jq .
}

function register_user() {
    local response=$(curl -s -X POST "http://localhost:8080/api/v1/users/register" \
        -H "Content-Type: application/json" \
        -d '{"username": "automated_test_user", "password": "TestPass123"}')
    echo "$response" | jq .
    echo "$response" > /tmp/register_response.json
}

function login_user() {
    local response=$(curl -s -X POST "http://localhost:8080/api/v1/users/login" \
        -H "Content-Type: application/json" \
        -d '{"username": "automated_test_user", "password": "TestPass123"}')
    echo "$response" | jq .
    echo "$response" > /tmp/login_response.json
}

function get_profile() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    curl -s -X GET "http://localhost:8080/api/v1/users/profile" \
        -H "Authorization: Bearer $token" | jq .
}

function upload_video() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    echo "Test video content for automated testing" > /tmp/test_video.txt
    local response=$(curl -s -X POST "http://localhost:8080/api/v1/videos/upload" \
        -H "Authorization: Bearer $token" \
        -F "file=@/tmp/test_video.txt" \
        -F "name=Automated Test Video")
    echo "$response" | jq .
    echo "$response" > /tmp/upload_response.json
}

function get_user_videos() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    curl -s -X GET "http://localhost:8080/api/v1/users/videos" \
        -H "Authorization: Bearer $token" | jq .
}

function get_video_info() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    local video_id=$(jq -r '.video_id' /tmp/upload_response.json)
    curl -s -X GET "http://localhost:8080/api/v1/videos/$video_id" \
        -H "Authorization: Bearer $token" | jq .
}

function add_comment() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    local video_id=$(jq -r '.video_id' /tmp/upload_response.json)
    local response=$(curl -s -X POST "http://localhost:8080/api/v1/videos/$video_id/comments" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d '{"content": "This is an automated test comment!"}')
    echo "$response" | jq .
    echo "$response" > /tmp/comment_response.json
}

function get_comments() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    local video_id=$(jq -r '.video_id' /tmp/upload_response.json)
    curl -s -X GET "http://localhost:8080/api/v1/videos/$video_id/comments" \
        -H "Authorization: Bearer $token" | jq .
}

function get_single_comment() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    local comment_id=$(jq -r '.comment_id' /tmp/comment_response.json)
    curl -s -X GET "http://localhost:8080/api/v1/comments/$comment_id" \
        -H "Authorization: Bearer $token" | jq .
}

# Main test execution
echo -e "${YELLOW}Starting test sequence...${NC}"

run_test "Health Check" "health_check"
run_test "User Registration" "register_user"
run_test "User Login" "login_user"
run_test "Get User Profile" "get_profile"
run_test "Video Upload" "upload_video"
run_test "Get User Videos" "get_user_videos"
run_test "Get Video Info" "get_video_info"
run_test "Add Comment" "add_comment"
run_test "Get Comments" "get_comments"
run_test "Get Single Comment" "get_single_comment"

# Cleanup
echo -e "\n${YELLOW}Cleaning up test files...${NC}"
rm -f /tmp/*.json /tmp/test_video.txt

# Summary
echo -e "\n${BLUE}======================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}======================================${NC}"
echo -e "Total tests: ${TEST_COUNT}"
echo -e "Passed: ${GREEN}${PASSED_TESTS}${NC}"
echo -e "Failed: ${RED}$((TEST_COUNT - PASSED_TESTS))${NC}"

if [ $PASSED_TESTS -eq $TEST_COUNT ]; then
    echo -e "\n${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed${NC}"
    exit 1
fi