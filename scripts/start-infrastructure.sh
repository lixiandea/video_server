#!/bin/bash
# 启动基础设施服务脚本

set -e

echo "🚀 启动基础设施服务..."

# 创建必要的目录
echo "📁 创建存储目录..."
mkdir -p storage/videos storage/temp logs

# 检查Docker环境
echo "🐳 检查Docker环境..."
if ! command -v docker &> /dev/null; then
    echo "❌ Docker未安装，请先安装Docker"
    exit 1
fi

if ! docker info &> /dev/null; then
    echo "❌ Docker守护进程未运行，请启动Docker"
    exit 1
fi

# 启动基础设施服务 (MySQL, Redis, 监控组件)
echo "🔧 启动基础设施服务..."
docker-compose up -d mysql redis prometheus grafana jaeger

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 15

# 检查服务状态
echo "📋 检查服务状态..."
docker-compose ps

# 验证关键服务
echo "🔍 验证服务可用性..."

# 检查MySQL
echo "🧪 测试MySQL连接..."
if docker-compose exec -T mysql mysqladmin ping -h localhost > /dev/null 2>&1; then
    echo "✅ MySQL服务正常"
else
    echo "❌ MySQL服务异常"
    docker-compose logs mysql
fi

# 检查Redis
echo "🧪 测试Redis连接..."
if docker-compose exec -T redis redis-cli ping > /dev/null 2>&1; then
    echo "✅ Redis服务正常"
else
    echo "❌ Redis服务异常"
    docker-compose logs redis
fi

# 显示服务信息
echo ""
echo "🎉 基础设施服务启动完成！"
echo ""
echo "📊 服务端口映射："
echo "   MySQL:     localhost:3306"
echo "   Redis:     localhost:6379" 
echo "   Prometheus: http://localhost:9090"
echo "   Grafana:    http://localhost:3001 (admin/admin)"
echo "   Jaeger:     http://localhost:16686"
echo ""
echo "📝 下一步操作："
echo "   1. 启动应用服务: ./start-app-services.sh"
echo "   2. 查看日志: docker-compose logs -f <service_name>"
echo "   3. 停止服务: ./stop-infrastructure.sh"