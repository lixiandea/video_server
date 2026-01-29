# Optimized Video Server

An optimized distributed microservices video platform built with Go.

## Architecture Overview

This project implements a modern microservices architecture with the following components:

- **API Gateway Layer** (`/cmd/api-server`) - Main entry point handling user requests
- **Scheduler Service** (`/cmd/scheduler`) - Background job processor for cleanup tasks
- **Worker Service** (`/cmd/worker`) - General background processing
- **Core Services** (`/internal/services`) - Business logic layer
- **Data Models** (`/internal/models`) - ORM models using GORM
- **Storage Layer** (`/pkg/storage`) - File storage abstraction
- **Authentication** (`/pkg/auth`) - JWT-based authentication
- **Database Layer** (`/pkg/database`) - Database connection and ORM
- **Validation Layer** (`/pkg/validation`) - Input validation utilities
- **Handlers** (`/internal/handlers`) - HTTP request handlers
- **Middleware** (`/internal/middleware`) - Authentication and other middleware
- **API Definitions** (`/api`) - API contract definitions

## Features

- JWT-based authentication
- Video upload and streaming
- User management
- Comment system
- Automatic video cleanup
- Configurable storage
- Rate limiting
- Input validation
- Comprehensive error handling

## Setup

1. Install dependencies:
   ```bash
   go mod tidy
   ```

2. Configure your database in `config.yaml`

3. Build the services:
   ```bash
   ./build.sh
   ```

4. Run the services:
   ```bash
   ./start.sh
   ```

## API Endpoints

### User Management
- POST `/api/v1/users/register` - Register a new user
- POST `/api/v1/users/login` - Login to the system
- GET `/api/v1/users/profile` - Get user profile (requires auth)
- PUT `/api/v1/users/profile` - Update user profile (requires auth)
- DELETE `/api/v1/users/account` - Delete user account (requires auth)

### Video Management
- POST `/api/v1/videos/upload` - Upload a video (requires auth)
- GET `/api/v1/videos/{id}` - Get video information (requires auth)
- GET `/api/v1/videos/{id}/stream` - Stream video content (requires auth)
- GET `/api/v1/users/videos` - Get user's videos (requires auth)
- DELETE `/api/v1/videos/{id}` - Delete a video (requires auth)

### Comments
- POST `/api/v1/videos/{id}/comments` - Add a comment to a video (requires auth)
- GET `/api/v1/videos/{id}/comments` - Get comments for a video (requires auth)
- GET `/api/v1/comments/{id}` - Get specific comment (requires auth)
- PUT `/api/v1/comments/{id}` - Update a comment (requires auth)
- DELETE `/api/v1/comments/{id}` - Delete a comment (requires auth)

## Testing with Postman

We've provided a Postman collection for easy API testing:

1. Import the `postman_collection.json` file into Postman
2. Import the `postman_environment.json` file into Postman
3. Update the environment variables as needed
4. The collection includes all API endpoints organized by service

The collection includes:
- User service endpoints (registration, login, profile management)
- Video service endpoints (upload, stream, management)
- Comment service endpoints (create, read, update, delete)

## Frontend Test Interface

We've included a simple frontend interface to test the API functionality:

1. Make sure your backend services are running
2. Start the frontend server:
   ```bash
   ./start-frontend.sh
   ```
3. Open your browser and go to `http://localhost:3000`
4. Use the interface to test various API endpoints

The frontend interface includes:
- User management (register, login, profile management)
- Video management (upload, retrieve, delete)
- Comment management (create, read, update, delete)
- Real-time API response display
- Configurable API base URL and authentication token

## Docker Environment

The project includes a complete Docker environment with all necessary services:

1. Ensure Docker and Docker Compose are installed
2. Check if Docker environment is ready:
   ```bash
   ./check-docker.sh
   ```
3. Start the environment:
   ```bash
   ./start-docker.sh
   ```
4. Access the services:
   - API Server: http://localhost:8080
   - Frontend: http://localhost:3000
   - MySQL: localhost:3306
   - Redis: localhost:6379

To stop the environment:
```bash
./stop-docker.sh
```

The Docker environment includes:
- MySQL database with video server schema
- Redis for caching and session storage
- API server with proper configurations
- Scheduler service for background tasks
- Worker service for background processing
- Frontend test interface

## Security

- Passwords are securely hashed using bcrypt
- JWT tokens for session management
- Input validation for all user inputs
- SQL injection protection through GORM
- File upload validation to prevent malicious uploads

## Scalability Considerations

- Database connection pooling
- Configurable rate limiting
- Separate services for different concerns
- File storage abstraction for cloud storage integration
- Background job processing for heavy tasks

## Directory Structure

```
video_server/
├── api/                     # API definitions (by service)
│   ├── comments/            # Comment API definitions
│   ├── user/                # User API definitions
│   └── videos/              # Video API definitions
├── bin/                     # Compiled binaries
├── cmd/                     # Application main packages
│   ├── api-server/          # Main API service
│   ├── scheduler/           # Task scheduler service
│   └── worker/              # Background worker service
├── docker/                  # Docker configurations
│   └── mysql/               # MySQL initialization
├── frontend/                # Frontend test interface
│   ├── css/                 # Stylesheets
│   ├── js/                  # JavaScript files
│   ├── server.go            # Frontend server
│   └── index.html           # Main page
├── internal/                # Internal application code
│   ├── config/              # Configuration management
│   ├── handlers/            # HTTP request handlers
│   ├── middleware/          # HTTP middleware
│   ├── models/              # Data models (GORM)
│   ├── services/            # Business logic services
│   └── utils/               # Utility functions
├── pkg/                     # Shared libraries
│   ├── auth/                # Authentication utilities
│   ├── database/            # Database utilities
│   ├── storage/             # File storage abstraction
│   └── validation/          # Input validation
├── storage/                 # Runtime storage directories
│   ├── videos/              # Video files
│   └── temp/                # Temporary files
├── config.yaml              # Configuration file
├── docker-compose.yml       # Docker Compose configuration
├── Dockerfile.apiserver     # API server Dockerfile
├── Dockerfile.scheduler     # Scheduler Dockerfile
├── Dockerfile.worker        # Worker Dockerfile
├── Dockerfile.frontend      # Frontend Dockerfile
├── .env                     # Environment variables
├── check-docker.sh          # Docker environment checker
├── build.sh                 # Build script
├── start.sh                 # Startup script for backend services
├── start-frontend.sh        # Startup script for frontend server
├── start-docker.sh          # Startup script for Docker environment
├── stop-docker.sh           # Shutdown script for Docker environment
├── cleanup.sh               # Cleanup script
├── postman_collection.json  # Postman API collection
├── postman_environment.json # Postman environment file
├── go.mod                   # Go modules definition
└── README.md                # Project documentation
```

## Scripts

- `build.sh`: Compiles all services into binaries
- `start.sh`: Starts all backend services in the background
- `start-frontend.sh`: Starts the frontend test interface
- `check-docker.sh`: Checks if Docker environment is ready
- `start-docker.sh`: Starts the complete Docker environment
- `stop-docker.sh`: Stops the Docker environment
- `cleanup.sh`: Stops services and cleans temporary files