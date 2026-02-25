#!/bin/bash

# 监控功能测试脚本
# 用于验证监控系统的各项功能是否正常工作

set -e

echo "=== 视频服务监控功能测试 ==="
echo

# 配置
API_BASE_URL="http://localhost:8080"
TEST_USERNAME="monitor_test_user"
TEST_PASSWORD="TestPass123!"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试结果计数器
PASSED=0
FAILED=0

# 打印测试结果
print_result() {
    local test_name=$1
    local result=$2
    
    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}✓ ${test_name}${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ ${test_name}${NC}"
        ((FAILED++))
    fi
}

# 等待服务启动
wait_for_service() {
    local url=$1
    local max_attempts=30
    local attempt=1
    
    echo "等待服务启动..."
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$url/health" >/dev/null 2>&1; then
            echo "服务已就绪"
            return 0
        fi
        echo -n "."
        sleep 2
        ((attempt++))
    done
    
    echo -e "\n${RED}服务启动超时${NC}"
    return 1
}

# 测试健康检查端点
test_health_endpoint() {
    echo "测试健康检查端点..."
    
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/health" -o /tmp/health_response)
    if [ "$response" = "200" ]; then
        print_result "健康检查端点" "PASS"
    else
        print_result "健康检查端点" "FAIL"
    fi
}

# 测试指标端点
test_metrics_endpoint() {
    echo "测试指标端点..."
    
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/metrics" -o /tmp/metrics_response)
    if [ "$response" = "200" ]; then
        # 检查是否包含基本指标
        if grep -q "go_info" /tmp/metrics_response; then
            print_result "指标端点可用性" "PASS"
            
            # 显示一些关键指标
            echo "  关键指标:"
            grep "^go_info\|^process_start_time_seconds" /tmp/metrics_response | head -5 | sed 's/^/    /'
        else
            print_result "指标端点内容" "FAIL"
        fi
    else
        print_result "指标端点" "FAIL"
    fi
}

# 测试用户注册（会产生监控数据）
test_user_registration_monitoring() {
    echo "测试用户注册监控..."
    
    # 先尝试删除可能存在的测试用户
    curl -s -X DELETE "$API_BASE_URL/api/v1/test-users/$TEST_USERNAME" >/dev/null 2>&1
    
    # 注册新用户
    register_data="{\"username\":\"$TEST_USERNAME\",\"password\":\"$TEST_PASSWORD\"}"
    
    response=$(curl -s -w "%{http_code}" \
        -H "Content-Type: application/json" \
        -d "$register_data" \
        "$API_BASE_URL/api/v1/users/register" \
        -o /tmp/register_response)
    
    if [ "$response" = "201" ] || [ "$response" = "409" ]; then
        print_result "用户注册监控" "PASS"
        
        # 如果是409冲突，说明用户已存在，这也可以接受
        if [ "$response" = "409" ]; then
            echo "  用户已存在，测试继续..."
        fi
        
        # 提取token用于后续测试
        TOKEN=$(grep -o '"token":"[^"]*"' /tmp/register_response | cut -d'"' -f4)
        echo "  Token获取: ${TOKEN:0:20}..."
    else
        print_result "用户注册监控" "FAIL"
        echo "  HTTP状态码: $response"
        echo "  响应内容: $(cat /tmp/register_response)"
    fi
}

# 测试登录监控
test_login_monitoring() {
    echo "测试登录监控..."
    
    login_data="{\"username\":\"$TEST_USERNAME\",\"password\":\"$TEST_PASSWORD\"}"
    
    response=$(curl -s -w "%{http_code}" \
        -H "Content-Type: application/json" \
        -d "$login_data" \
        "$API_BASE_URL/api/v1/users/login" \
        -o /tmp/login_response)
    
    if [ "$response" = "200" ]; then
        print_result "登录监控" "PASS"
        
        # 提取登录token
        LOGIN_TOKEN=$(grep -o '"token":"[^"]*"' /tmp/login_response | cut -d'"' -f4)
        echo "  登录Token: ${LOGIN_TOKEN:0:20}..."
    else
        print_result "登录监控" "FAIL"
    fi
}

# 测试受保护路由监控
test_protected_route_monitoring() {
    echo "测试受保护路由监控..."
    
    if [ -z "$LOGIN_TOKEN" ]; then
        print_result "受保护路由监控" "SKIP"
        echo "  跳过: 未获取到登录token"
        return
    fi
    
    response=$(curl -s -w "%{http_code}" \
        -H "Authorization: Bearer $LOGIN_TOKEN" \
        "$API_BASE_URL/api/v1/users/profile" \
        -o /tmp/profile_response)
    
    if [ "$response" = "200" ]; then
        print_result "受保护路由监控" "PASS"
    else
        print_result "受保护路由监控" "FAIL"
    fi
}

# 测试错误监控
test_error_monitoring() {
    echo "测试错误监控..."
    
    # 发送无效请求来触发错误
    response=$(curl -s -w "%{http_code}" \
        -H "Content-Type: application/json" \
        -d '{"invalid":"data"}' \
        "$API_BASE_URL/api/v1/users/register" \
        -o /tmp/error_response)
    
    # 任何4xx或5xx响应都被视为错误监控测试通过
    if [[ "$response" =~ ^[45][0-9][0-9]$ ]]; then
        print_result "错误监控" "PASS"
        echo "  错误状态码: $response"
    else
        print_result "错误监控" "FAIL"
    fi
}

# 测试日志文件生成
test_logging() {
    echo "测试日志文件生成..."
    
    # 检查日志目录是否存在
    if [ -d "./logs" ]; then
        log_files=$(find ./logs -name "*.log" -type f 2>/dev/null)
        if [ -n "$log_files" ]; then
            print_result "日志文件生成" "PASS"
            echo "  日志文件数量: $(echo "$log_files" | wc -l | tr -d ' ')"
            echo "  最新日志文件: $(ls -t ./logs/*.log 2>/dev/null | head -1)"
        else
            print_result "日志文件生成" "FAIL"
            echo "  日志目录为空"
        fi
    else
        print_result "日志文件生成" "FAIL"
        echo "  日志目录不存在"
    fi
}

# 显示监控指标摘要
show_metrics_summary() {
    echo
    echo "=== 监控指标摘要 ==="
    
    if curl -s "$API_BASE_URL/metrics" >/dev/null 2>&1; then
        echo "HTTP请求数统计:"
        curl -s "$API_BASE_URL/metrics" | grep "http_requests_total" | head -10 | sed 's/^/  /'
        
        echo
        echo "活跃用户数:"
        curl -s "$API_BASE_URL/metrics" | grep "active_users_total" | sed 's/^/  /'
        
        echo
        echo "数据库连接数:"
        curl -s "$API_BASE_URL/metrics" | grep "db_connections_open" | sed 's/^/  /'
    else
        echo "无法获取指标数据"
    fi
}

# 主测试流程
main() {
    echo "开始监控功能测试..."
    echo
    
    # 等待服务启动
    if ! wait_for_service "$API_BASE_URL"; then
        echo -e "${RED}服务未启动，请先启动API服务${NC}"
        exit 1
    fi
    
    echo
    
    # 执行各项测试
    test_health_endpoint
    test_metrics_endpoint
    test_user_registration_monitoring
    test_login_monitoring
    test_protected_route_monitoring
    test_error_monitoring
    test_logging
    
    echo
    echo "=== 测试结果汇总 ==="
    echo -e "${GREEN}通过: $PASSED${NC}"
    echo -e "${RED}失败: $FAILED${NC}"
    echo "总计: $((PASSED + FAILED))"
    
    if [ $FAILED -eq 0 ]; then
        echo -e "${GREEN}所有监控测试通过！${NC}"
        show_metrics_summary
        exit 0
    else
        echo -e "${RED}部分测试失败，请检查监控配置${NC}"
        exit 1
    fi
}

# 清理函数
cleanup() {
    echo
    echo "清理测试数据..."
    # 清理临时文件
    rm -f /tmp/health_response /tmp/metrics_response /tmp/register_response \
          /tmp/login_response /tmp/profile_response /tmp/error_response
}

# 注册清理函数
trap cleanup EXIT

# 运行主函数
main