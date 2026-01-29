#!/bin/bash

# Script to start the video server environment with Docker

echo "Starting video server environment with Docker..."

# Create necessary directories
mkdir -p storage/videos storage/temp

# Start all services with Docker Compose
echo "Starting services..."
docker-compose up -d

echo "Waiting for services to start..."
sleep 10

# Show running containers
echo "Running containers:"
docker-compose ps

echo ""
echo "Services are now running:"
echo "- MySQL: localhost:3306"
echo "- Redis: localhost:6379"
echo "- API Server: http://localhost:8080"
echo "- Scheduler: http://localhost:8089"
echo "- Frontend: http://localhost:3000"
echo ""
echo "To view logs: docker-compose logs -f"
echo "To stop services: docker-compose down"