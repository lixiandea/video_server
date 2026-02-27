#!/bin/bash
# 启动应用服务脚本

set -e

echo "🚀 启动应用服务..."

# 检查基础设施服务是否运行
echo "🔍 检查基础设施服务状态..."
if ! docker-compose ps mysql | grep -q "Up"; then
    echo "❌ MySQL服务未运行，请先启动基础设施服务"
    echo "   执行: ./scripts/start-infrastructure.sh"
    exit 1
fi

if ! docker-compose ps redis | grep -q "Up"; then
    echo "❌ Redis服务未运行，请先启动基础设施服务"
    echo "   执行: ./scripts/start-infrastructure.sh"
    exit 1
fi

# 构建应用服务镜像
echo "🔨 构建应用服务镜像..."
docker-compose build api-server scheduler worker frontend

# 启动应用服务
echo "🔧 启动应用服务..."
docker-compose up -d api-server scheduler worker frontend

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 10

# 检查服务状态
echo "📋 检查服务状态..."
docker-compose ps

# 验证API服务
echo "🔍 验证API服务..."
MAX_RETRIES=30
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -f http://localhost:8080/health > /dev/null 2>&1; then
        echo "✅ API服务正常"
        break
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
    echo "⏳ 等待API服务启动... ($RETRY_COUNT/$MAX_RETRIES)"
    sleep 2
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo "❌ API服务启动超时"
    docker-compose logs api-server
    exit 1
fi

# 验证前端服务
echo "🔍 验证前端服务..."
if curl -f http://localhost:3000 > /dev/null 2>&1; then
    echo "✅ 前端服务正常"
else
    echo "⚠️  前端服务可能仍在启动中"
fi

# 显示服务信息
echo ""
echo "🎉 应用服务启动完成！"
echo ""
echo "📊 服务端口映射："
echo "   API Server:   http://localhost:8080"
echo "   Scheduler:    http://localhost:8089"
echo "   Frontend:     http://localhost:3000"
echo ""
echo "📝 常用命令："
echo "   查看所有服务状态: docker-compose ps"
echo "   查看服务日志:     docker-compose logs -f <service_name>"
echo "   停止应用服务:     ./scripts/stop-app-services.sh"
echo "   停止所有服务:     ./scripts/stop-all-services.sh"