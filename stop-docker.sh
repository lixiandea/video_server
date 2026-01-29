#!/bin/bash

# Script to stop the video server environment with Docker

echo "Stopping video server environment..."

# Stop all services
docker-compose down

echo "Services stopped."