#!/bin/bash

# Start the frontend server for video service testing

echo "Starting frontend server for video service testing..."

# Change to the project directory
cd "$(dirname "$0")"

# Start the frontend server
echo "Starting server on port 3000..."
cd frontend
go run server.go

echo "Frontend server stopped."