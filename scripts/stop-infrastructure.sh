#!/bin/bash
# 停止基础设施服务脚本

echo "🛑 停止基础设施服务..."

# 停止基础设施相关的服务
SERVICES="mysql redis prometheus grafana jaeger"

for service in $SERVICES; do
    if docker-compose ps $service | grep -q "Up"; then
        echo "⏹️  停止 $service 服务..."
        docker-compose stop $service
    else
        echo "⏭️  $service 服务未运行"
    fi
done

echo "✅ 基础设施服务已停止"
echo "💡 如需完全清理容器和数据卷，请执行: docker-compose down -v"