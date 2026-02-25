#!/bin/bash

# Docker国内镜像源配置脚本
# 适用于macOS Docker Desktop

echo "=== Docker国内镜像源配置 ==="

# 检查Docker Desktop是否运行
if ! pgrep -f "Docker Desktop" > /dev/null; then
    echo "请先启动Docker Desktop应用"
    exit 1
fi

echo "正在配置国内镜像源..."

# 创建Docker配置目录
mkdir -p ~/.docker

# 备份现有配置
if [ -f ~/.docker/daemon.json ]; then
    cp ~/.docker/daemon.json ~/.docker/daemon.json.backup.$(date +%Y%m%d_%H%M%S)
    echo "已备份原配置文件"
fi

# 写入新的配置
cat > ~/.docker/daemon.json << EOF
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com",
    "https://dockerproxy.com"
  ],
  "experimental": false,
  "features": {
    "buildkit": true
  }
}
EOF

echo "配置完成！"
echo ""
echo "请执行以下操作："
echo "1. 重启Docker Desktop应用"
echo "2. 或者在Docker Desktop中手动重启："
echo "   Docker Desktop -> Preferences -> Resources -> Restart"
echo ""
echo "配置的镜像源："
echo "- 中科大镜像: https://docker.mirrors.ustc.edu.cn"
echo "- 网易镜像: https://hub-mirror.c.163.com"  
echo "- 百度镜像: https://mirror.baidubce.com"
echo "- DockerProxy: https://dockerproxy.com"