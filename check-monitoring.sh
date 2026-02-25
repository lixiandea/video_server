#!/bin/bash

# 监控服务状态检查脚本

echo "=== 监控服务状态检查 ==="
echo ""

# 检查Docker Compose服务状态
echo "Docker Compose服务状态:"
docker-compose ps

echo ""
echo "=== 端口监听状态 ==="
echo "检查各服务端口是否开放:"

# 检查各个端口
ports=("3306" "6379" "8080" "8089" "3000" "9090" "3001")
services=("MySQL" "Redis" "API Server" "Scheduler" "Frontend" "Prometheus" "Grafana")

for i in "${!ports[@]}"; do
    if lsof -Pi :${ports[$i]} -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "✓ ${services[$i]} (${ports[$i]}) - 运行中"
    else
        echo "✗ ${services[$i]} (${ports[$i]}) - 未运行"
    fi
done

echo ""
echo "=== 访问地址 ==="
echo "API Server: http://localhost:8080"
echo "Prometheus: http://localhost:9090"
echo "Grafana: http://localhost:3001 (admin/admin)"
echo "Frontend: http://localhost:3000"
echo ""
echo "=== 快速测试 ==="
echo "健康检查: curl http://localhost:8080/health"
echo "Metrics: curl http://localhost:8080/metrics"