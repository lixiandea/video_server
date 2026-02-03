#!/bin/bash

# 安装依赖
echo "Installing dependencies..."
npm install

# 构建客户端和服务端资源
echo "Building client assets..."
npx webpack --config webpack.client.js

echo "Building server bundle..."
npx webpack --config webpack.server.js

echo "Build completed!"