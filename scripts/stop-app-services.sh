#!/bin/bash
# 停止应用服务脚本

echo "🛑 停止应用服务..."

# 停止应用相关的服务
SERVICES="api-server scheduler worker frontend"

for service in $SERVICES; do
    if docker-compose ps $service | grep -q "Up"; then
        echo "⏹️  停止 $service 服务..."
        docker-compose stop $service
    else
        echo "⏭️  $service 服务未运行"
    fi
done

echo "✅ 应用服务已停止"
echo "💡 基础设施服务仍在运行，如需停止请执行: ./scripts/stop-infrastructure.sh"