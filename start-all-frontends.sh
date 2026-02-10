#!/bin/bash

# Start both frontend servers: legacy Go frontend and new Vue SSR frontend

echo "Starting all frontend servers for video service..."

# Get the project directory
PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Function to start Go frontend
start_go_frontend() {
    echo "Starting Go frontend server on port 3000..."
    cd "$PROJECT_DIR/frontend"
    go run server.go &
    GO_FRONTEND_PID=$!
    echo "Go frontend server started with PID: $GO_FRONTEND_PID"
}

# Function to start Vue SSR frontend
start_vue_frontend() {
    echo "Starting Vue SSR frontend server on port 3001..."
    cd "$PROJECT_DIR/frontend-vue"
    npm run dev &
    VUE_FRONTEND_PID=$!
    echo "Vue SSR frontend server started with PID: $VUE_FRONTEND_PID"
}

# Start both frontends
start_go_frontend
sleep 2  # Wait a moment for the first server to start
start_vue_frontend

echo "Both frontend servers are running:"
echo "- Go frontend: http://localhost:3000"
echo "- Vue SSR frontend: http://localhost:3001"

# Keep the script running
wait $GO_FRONTEND_PID $VUE_FRONTEND_PID