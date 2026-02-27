#!/bin/bash
# 服务状态检查脚本

echo "🔍 检查服务状态..."

echo "=== Docker环境检查 ==="
if ! command -v docker &> /dev/null; then
    echo "❌ Docker未安装"
    exit 1
else
    echo "✅ Docker已安装"
    docker --version
fi

if ! docker info &> /dev/null; then
    echo "❌ Docker守护进程未运行"
    exit 1
else
    echo "✅ Docker守护进程运行中"
fi

echo ""
echo "=== 服务状态 ==="
docker-compose ps

echo ""
echo "=== 端口占用检查 ==="
PORTS=("3306" "6379" "8080" "8089" "3000" "3001" "9090" "16686")

for port in "${PORTS[@]}"; do
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        pid=$(lsof -t -i :$port)
        process=$(lsof -t -i :$port | xargs ps -p 2>/dev/null | tail -1)
        echo "🟡 端口 $port 被占用 (PID: $pid)"
    else
        echo "✅ 端口 $port 可用"
    fi
done

echo ""
echo "=== 健康检查 ==="

# MySQL健康检查
if docker-compose ps mysql | grep -q "Up"; then
    if docker-compose exec -T mysql mysqladmin ping -h localhost > /dev/null 2>&1; then
        echo "✅ MySQL服务健康"
    else
        echo "❌ MySQL服务不健康"
    fi
else
    echo "⏭️  MySQL服务未运行"
fi

# Redis健康检查
if docker-compose ps redis | grep -q "Up"; then
    if docker-compose exec -T redis redis-cli ping > /dev/null 2>&1; then
        echo "✅ Redis服务健康"
    else
        echo "❌ Redis服务不健康"
    fi
else
    echo "⏭️  Redis服务未运行"
fi

# API Server健康检查
if docker-compose ps api-server | grep -q "Up"; then
    if curl -f http://localhost:8080/health > /dev/null 2>&1; then
        echo "✅ API Server服务健康"
    else
        echo "❌ API Server服务不健康"
    fi
else
    echo "⏭️  API Server服务未运行"
fi

echo ""
echo "=== 资源使用情况 ==="
if command -v docker &> /dev/null; then
    echo "Docker容器资源使用:"
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" 2>/dev/null || echo "无法获取资源统计"
fi

echo ""
echo "=== 存储空间检查 ==="
echo "项目目录大小:"
du -sh . 2>/dev/null || echo "无法计算目录大小"

if [ -d "storage" ]; then
    echo "存储目录大小:"
    du -sh storage/* 2>/dev/null || echo "存储目录为空"
fi

echo ""
echo "📋 建议操作:"
echo "   启动基础设施: ./scripts/start-infrastructure.sh"
echo "   启动应用服务: ./scripts/start-app-services.sh"
echo "   查看详细日志: docker-compose logs -f <service_name>"
echo "   停止所有服务: ./scripts/stop-all-services.sh"