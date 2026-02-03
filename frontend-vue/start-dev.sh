#!/bin/bash

# 开发模式启动 Vue SSR 应用
echo "Starting Vue SSR application in development mode..."

# 安装依赖（如果需要）
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install
fi

# 启动开发服务器
npm run dev