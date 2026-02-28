#!/bin/bash

# Frontend development startup script

echo "🚀 Starting Video Server Frontend Development Server..."

# Change to the frontend directory
cd "$(dirname "$0")/frontend-new"

# Check if node_modules exists
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
fi

# Start development server
echo "🌐 Starting development server on http://localhost:3000"
npm run dev
