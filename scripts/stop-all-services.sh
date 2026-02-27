#!/bin/bash
# 停止所有服务脚本

echo "🛑 停止所有服务..."

# 停止所有服务
docker-compose down

echo "✅ 所有服务已停止"
echo "🗑️  如需删除数据卷，请执行: docker-compose down -v"
echo "🧹 清理未使用的资源: docker system prune -f"