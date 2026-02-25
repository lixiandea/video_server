#!/bin/bash

# 监控组件启动脚本
# 用于单独启动Prometheus和Grafana监控服务

set -e

echo "=== 启动监控组件 ==="

# 检查Docker环境
if ! command -v docker &> /dev/null; then
    echo "错误: Docker未安装"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "错误: Docker Compose未安装"
    exit 1
fi

# 创建必要的目录
mkdir -p deploy/monitoring/dashboards

# 复制仪表板文件（如果存在）
if [ -f "deploy/monitoring/grafana-dashboard.json" ]; then
    cp deploy/monitoring/grafana-dashboard.json deploy/monitoring/dashboards/video-server-dashboard.json
fi

# 启动监控服务
echo "正在启动Prometheus和Grafana..."
docker-compose up -d prometheus grafana

# 等待服务启动
echo "等待服务启动..."
sleep 10

# 检查服务状态
echo "检查服务状态:"
docker-compose ps prometheus grafana

echo ""
echo "=== 监控服务访问地址 ==="
echo "Prometheus: http://localhost:9090"
echo "Grafana: http://localhost:3001 (用户名: admin, 密码: admin)"
echo ""
echo "=== 使用说明 ==="
echo "1. 访问Prometheus查看原始指标数据"
echo "2. 访问Grafana查看可视化监控面板"
echo "3. 如需停止监控服务: docker-compose stop prometheus grafana"