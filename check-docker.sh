#!/bin/bash

# Script to check if Docker environment is ready

echo "Checking Docker environment..."

# Check if Docker is installed
if ! [ -x "$(command -v docker)" ]; then
  echo "Error: Docker is not installed." >&2
  echo "Please install Docker first:"
  echo "  On macOS: brew install --cask docker"
  echo "  Or download from: https://www.docker.com/products/docker-desktop"
  exit 1
else
  echo "✓ Docker is installed"
  docker --version
fi

# Check if Docker Compose is installed
if ! [ -x "$(command -v docker-compose)" ]; then
  echo "Error: Docker Compose is not installed." >&2
  echo "Please install Docker Compose first:"
  echo "  Usually comes with Docker Desktop"
  echo "  Or install standalone: https://docs.docker.com/compose/install/"
  exit 1
else
  echo "✓ Docker Compose is installed"
  docker-compose --version
fi

# Check if Docker daemon is running
if ! docker info >/dev/null 2>&1; then
  echo "Error: Docker daemon is not running." >&2
  echo "Please start Docker Desktop or the Docker service."
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
echo "To stop the environment, run:"
echo "  ./stop-docker.sh"