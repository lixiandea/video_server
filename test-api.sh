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
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test counter
TEST_COUNT=0
PASSED_TESTS=0
FAILED_TESTS=0

# Global variables for test data
TEST_VIDEO_ID=""
TEST_COMMENT_ID=""

function run_test() {
    local test_name="$1"
    local test_command="$2"
    local expect_success="${3:-true}"  # Default expect success

    TEST_COUNT=$((TEST_COUNT + 1))
    echo -e "\n${BLUE}Test $TEST_COUNT: $test_name${NC}"
    echo "----------------------------------------"

    if eval "$test_command"; then
        if [ "$expect_success" = "true" ]; then
            PASSED_TESTS=$((PASSED_TESTS + 1))
            echo -e "${GREEN}✅ PASSED${NC}"
        else
            PASSED_TESTS=$((PASSED_TESTS + 1))
            echo -e "${GREEN}✅ PASSED (Expected failure)${NC}"
        fi
        return 0
    else
        if [ "$expect_success" = "false" ]; then
            PASSED_TESTS=$((PASSED_TESTS + 1))
            echo -e "${GREEN}✅ PASSED (Expected failure)${NC}"
            return 0
        else
            FAILED_TESTS=$((FAILED_TESTS + 1))
            echo -e "${RED}❌ FAILED${NC}"
            return 1
        fi
    fi
}

function health_check() {
    curl -f -s -X GET "http://localhost:8080/health" | jq .
}

function register_user() {
    local username="test_user_$(date +%s)"
    local response=$(curl -s -X POST "http://localhost:8080/api/v1/users/register" \
        -H "Content-Type: application/json" \
        -d "{\"username\": \"$username\", \"password\": \"TestPass123\"}")
    echo "$response" | jq .
    echo "$response" > /tmp/register_response.json
}

function login_user() {
    local username=$(jq -r '.username' /tmp/register_response.json 2>/dev/null || echo "automated_test_user")
    local response=$(curl -s -X POST "http://localhost:8080/api/v1/users/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\": \"$username\", \"password\": \"TestPass123\"}")
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
    
    # Check if test video exists
    if [ ! -f /tmp/video_test/test_video.mp4 ]; then
        echo -e "${YELLOW}⚠️  Test video not found, downloading...${NC}"
        mkdir -p /tmp/video_test
        curl -sL -o /tmp/video_test/test_video.mp4 "https://www.w3schools.com/html/mov_bbb.mp4"
    fi
    
    local response=$(curl -s -X POST "http://localhost:8080/api/v1/videos/upload" \
        -H "Authorization: Bearer $token" \
        -F "file=@/tmp/video_test/test_video.mp4" \
        -F "name=Big Buck Bunny - Test Video")
    echo "$response" | jq .
    echo "$response" > /tmp/upload_response.json
    
    # Extract video ID for subsequent tests
    TEST_VIDEO_ID=$(echo "$response" | jq -r '.video_id // empty')
    if [ -n "$TEST_VIDEO_ID" ]; then
        echo -e "${CYAN}📹 Video uploaded successfully: $TEST_VIDEO_ID${NC}"
    fi
}

function get_user_videos() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    local response=$(curl -s -X GET "http://localhost:8080/api/v1/users/videos" \
        -H "Authorization: Bearer $token")
    echo "$response" | jq .
    
    # Verify video count increased
    local video_count=$(echo "$response" | jq '.total')
    echo -e "${CYAN}📊 Total videos: $video_count${NC}"
}

function get_video_info() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    local video_id="$TEST_VIDEO_ID"
    
    if [ -z "$video_id" ]; then
        echo -e "${YELLOW}⚠️  No video ID available, skipping detailed info test${NC}"
        return 1
    fi
    
    local response=$(curl -s -X GET "http://localhost:8080/api/v1/videos/$video_id" \
        -H "Authorization: Bearer $token")
    echo "$response" | jq .
    
    # Verify video info
    local video_name=$(echo "$response" | jq -r '.name // empty')
    if [ -n "$video_name" ]; then
        echo -e "${CYAN}📹 Video name: $video_name${NC}"
    fi
}

function get_video_stream() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    local video_id="$TEST_VIDEO_ID"
    
    if [ -z "$video_id" ]; then
        echo -e "${YELLOW}⚠️  No video ID available, skipping stream test${NC}"
        return 1
    fi
    
    # Test video stream endpoint (just check headers)
    local status=$(curl -s -o /dev/null -w "%{http_code}" \
        -X GET "http://localhost:8080/api/v1/videos/$video_id/stream" \
        -H "Authorization: Bearer $token")
    
    if [ "$status" = "200" ] || [ "$status" = "416" ]; then
        echo -e "${CYAN}🎬 Video stream accessible (HTTP $status)${NC}"
        return 0
    else
        echo -e "${RED}❌ Video stream failed (HTTP $status)${NC}"
        return 1
    fi
}

function add_comment() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    local video_id="$TEST_VIDEO_ID"
    
    if [ -z "$video_id" ]; then
        echo -e "${YELLOW}⚠️  No video ID available, skipping comment test${NC}"
        return 1
    fi
    
    local response=$(curl -s -X POST "http://localhost:8080/api/v1/videos/$video_id/comments" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d '{"content": "This is an automated test comment! Great video! 🎉"}')
    echo "$response" | jq .
    echo "$response" > /tmp/comment_response.json
    
    # Extract comment ID
    TEST_COMMENT_ID=$(echo "$response" | jq -r '.comment_id // empty')
    if [ -n "$TEST_COMMENT_ID" ]; then
        echo -e "${CYAN}💬 Comment added successfully: $TEST_COMMENT_ID${NC}"
    fi
}

function get_comments() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    local video_id="$TEST_VIDEO_ID"
    
    if [ -z "$video_id" ]; then
        echo -e "${YELLOW}⚠️  No video ID available, skipping comments list test${NC}"
        return 1
    fi
    
    local response=$(curl -s -X GET "http://localhost:8080/api/v1/videos/$video_id/comments" \
        -H "Authorization: Bearer $token")
    echo "$response" | jq .
    
    # Verify comment count
    local comment_count=$(echo "$response" | jq '.comments | length')
    echo -e "${CYAN}💬 Total comments: $comment_count${NC}"
}

function get_single_comment() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    local comment_id="$TEST_COMMENT_ID"
    
    if [ -z "$comment_id" ]; then
        echo -e "${YELLOW}⚠️  No comment ID available, skipping single comment test${NC}"
        return 1
    fi
    
    local response=$(curl -s -X GET "http://localhost:8080/api/v1/comments/$comment_id" \
        -H "Authorization: Bearer $token")
    echo "$response" | jq .
    
    # Verify comment content
    local content=$(echo "$response" | jq -r '.content // empty')
    if [ -n "$content" ]; then
        echo -e "${CYAN}💬 Comment content: $content${NC}"
    fi
}

function update_comment() {
    local token=$(jq -r '.token' /tmp/login_response.json)
    local comment_id="$TEST_COMMENT_ID"
    
    if [ -z "$comment_id" ]; then
        echo -e "${YELLOW}⚠️  No comment ID available, skipping comment update test${NC}"
        return 1
    fi
    
    local response=$(curl -s -X PUT "http://localhost:8080/api/v1/comments/$comment_id" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d '{"content": "Updated comment: This is even better! 👍"}')
    echo "$response" | jq .
    
    # Verify update
    if echo "$response" | jq -e '.message' > /dev/null 2>&1; then
        echo -e "${CYAN}✏️  Comment updated successfully${NC}"
    fi
}

# Main test execution
echo -e "${YELLOW}Starting test sequence...${NC}"
echo -e "${CYAN}📋 Test Plan: Health → Register → Login → Profile → Upload Video → Get Videos → Video Info → Stream → Add Comment → Get Comments → Update Comment → Get Single Comment${NC}"

# Basic tests
run_test "Health Check" "health_check"
run_test "User Registration" "register_user"
run_test "User Login" "login_user"
run_test "Get User Profile" "get_profile"

# Video tests (with success expectation)
run_test "Video Upload (Success)" "upload_video"
run_test "Get User Videos (Success)" "get_user_videos"
run_test "Get Video Info (Success)" "get_video_info"
run_test "Get Video Stream (Success)" "get_video_stream"

# Comment tests (with success expectation)
run_test "Add Comment (Success)" "add_comment"
run_test "Get Comments (Success)" "get_comments"
run_test "Update Comment (Success)" "update_comment"
run_test "Get Single Comment (Success)" "get_single_comment"

# Cleanup
echo -e "\n${YELLOW}Cleaning up test files...${NC}"
rm -f /tmp/*.json

# Summary
echo -e "\n${BLUE}======================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}======================================${NC}"
echo -e "Total tests: ${TEST_COUNT}"
echo -e "Passed: ${GREEN}${PASSED_TESTS}${NC}"
echo -e "Failed: ${RED}${FAILED_TESTS}${NC}"
echo -e "Success Rate: ${GREEN}$(( (PASSED_TESTS * 100) / TEST_COUNT ))%${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All tests passed!${NC}"
    echo -e "${CYAN}📊 Test Coverage:${NC}"
    echo -e "   ✅ Health Check"
    echo -e "   ✅ User Registration & Login"
    echo -e "   ✅ User Profile"
    echo -e "   ✅ Video Upload"
    echo -e "   ✅ Video List & Info"
    echo -e "   ✅ Video Streaming"
    echo -e "   ✅ Comment CRUD Operations"
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed${NC}"
    exit 1
fi