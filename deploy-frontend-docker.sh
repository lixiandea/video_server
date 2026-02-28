#!/bin/bash

# Frontend Docker deployment script

echo "🐳 Building and Deploying Frontend with Docker..."

cd "$(dirname "$0")"

# Build the frontend image
echo "📦 Building Docker image..."
docker build -f Dockerfile.frontend.nginx -t video-frontend:latest ./frontend-new

if [ $? -eq 0 ]; then
    echo "✅ Docker image built successfully!"
    
    # Stop existing container
    echo "🛑 Stopping existing container..."
    docker stop video_server_frontend 2>/dev/null || true
    docker rm video_server_frontend 2>/dev/null || true
    
    # Start new container
    echo "🚀 Starting new container..."
    docker run -d \
        --name video_server_frontend \
        --network video_server_video_network \
        -p 80:80 \
        --restart always \
        video-frontend:latest
    
    echo "✅ Frontend deployed successfully!"
    echo "🌐 Access the application at http://localhost"
else
    echo "❌ Docker build failed!"
    exit 1
fi
