#!/bin/bash

# Docker环境检查脚本
# 检查Docker和Docker Compose是否正确安装和运行

echo "Checking Docker environment..."

# 检查Docker是否安装
if ! command -v docker &> /dev/null; then
    echo "✗ Docker is not installed"
    exit 1
else
    echo "✓ Docker is installed"
    docker --version
fi

# 检查Docker Compose是否安装
if ! command -v docker-compose &> /dev/null; then
    echo "✗ Docker Compose is not installed"
    exit 1
else
    echo "✓ Docker Compose is installed"
    docker-compose --version
fi

# 检查Docker守护进程是否运行
if ! docker info &> /dev/null; then
    echo "✗ Docker daemon is not running"
    exit 1
else
    echo "✓ Docker daemon is running"
fi

echo ""
echo "Docker environment is ready!"
echo ""
echo "To start the video server environment, run:"
echo "  ./start-docker.sh"
echo ""
echo "To start only monitoring services, run:"
echo "  ./start-monitoring.sh"
echo ""
echo "To stop the environment, run:"
echo "  ./stop-docker.sh"