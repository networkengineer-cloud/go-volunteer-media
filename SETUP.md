# Setup Guide

This guide will help you set up and run the Volunteer Media application locally.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.24 or higher**: [Download Go](https://golang.org/dl/)
- **Node.js 20 or higher**: [Download Node.js](https://nodejs.org/)
- **PostgreSQL 15 or higher**: [Download PostgreSQL](https://www.postgresql.org/download/) OR Docker
- **Docker & Docker Compose** (optional, for containerized setup): [Download Docker](https://www.docker.com/get-started)

## Quick Start with Docker

The easiest way to get started is using Docker Compose:

```bash
# 1. Clone the repository
git clone https://github.com/networkengineer-cloud/go-volunteer-media.git
cd go-volunteer-media

# 2. Start the database
docker compose up -d postgres_dev

# 3. In one terminal, start the backend
go run cmd/api/main.go

# 4. In another terminal, start the frontend
cd frontend
npm install
npm run dev
```

Visit http://localhost:5173 to see the application!

## Manual Setup

### Step 1: Set Up the Database

#### Option A: Using Docker
```bash
docker compose up -d postgres_dev
```

#### Option B: Using Local PostgreSQL
```bash
# Create database
createdb volunteer_media_dev

# Or using psql
psql -U postgres -c "CREATE DATABASE volunteer_media_dev;"
```

### Step 2: Configure Environment Variables

Create a `.env` file in the root directory:

```bash
cp .env.example .env
```

Edit `.env` with your settings:

```env
# Application Configuration
ENV=development
PORT=8080

# JWT Secret (CHANGE THIS IN PRODUCTION!)
JWT_SECRET=your-secret-key-change-in-production

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=volunteer_media_dev
DB_SSLMODE=disable
```

### Step 3: Start the Backend

```bash
# Install Go dependencies (if not already installed)
go mod download

# Run the application
go run cmd/api/main.go
```

The backend will:
- Connect to the database
- Run migrations automatically
- Create default groups (dogs, cats, modsquad)
- Start listening on port 8080

You should see output like:
```
2025/10/10 23:35:47 Database connection established
2025/10/10 23:35:47 Running database migrations...
2025/10/10 23:35:47 Migrations completed successfully
2025/10/10 23:35:47 Created default group: dogs
2025/10/10 23:35:47 Created default group: cats
2025/10/10 23:35:47 Created default group: modsquad
2025/10/10 23:37:19 Server starting on port 8080...
```

### Step 4: Start the Frontend

In a new terminal:

```bash
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

The frontend will start on http://localhost:5173

## First-Time Setup

### 1. Register a User

1. Open http://localhost:5173/login
2. Click "Need an account? Register"
3. Fill in the form:
   - Username: `admin`
   - Email: `admin@example.com`
   - Password: `admin123`
4. Click "Register"

### 2. Make Your User an Admin

By default, new users are not admins. To make your first user an admin:

```bash
# Using Docker
docker exec volunteer_media_db_dev psql -U postgres -d volunteer_media_dev \
  -c "UPDATE users SET is_admin = true WHERE username = 'admin';"

# Using local PostgreSQL
psql -U postgres -d volunteer_media_dev \
  -c "UPDATE users SET is_admin = true WHERE username = 'admin';"
```

Log out and log back in to see admin privileges.

### 3. Assign Users to Groups

As an admin, you can assign users to groups via API:

```bash
# Get your token by logging in
TOKEN=$(curl -s -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r .token)

# Add user (ID 1) to dogs group (ID 1)
curl -X POST http://localhost:8080/api/admin/users/1/groups/1 \
  -H "Authorization: Bearer $TOKEN"
```

## Testing the Application

### Test Backend APIs

```bash
# Register a user
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"test123"}'

# Login
TOKEN=$(curl -s -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"test123"}' | jq -r .token)

# Get groups
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/groups

# Create an animal (if user is in the group)
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Buddy","species":"Dog","breed":"Golden Retriever","age":3,"description":"Friendly dog","status":"available"}' \
  http://localhost:8080/api/groups/1/animals
```

### Build for Production

#### Backend
```bash
# Build binary
go build -o api ./cmd/api

# Run binary
./api
```

#### Frontend
```bash
cd frontend
npm run build

# The built files will be in frontend/dist/
```

## Docker Production Deployment

### Build and Run with Docker

```bash
# Build the frontend first
cd frontend
npm install
npm run build
cd ..

# Build Docker image
docker build -t volunteer-media:latest .

# Run with environment variables
docker run -d \
  -p 8080:8080 \
  -e ENV=production \
  -e DB_HOST=your-database-host \
  -e DB_PASSWORD=your-secure-password \
  -e JWT_SECRET=your-secret-key \
  --name volunteer-media \
  volunteer-media:latest
```

### Using Docker Compose for Full Stack

```bash
# Start everything (database + API)
docker compose up -d

# View logs
docker compose logs -f api

# Stop everything
docker compose down
```

## Troubleshooting

### Database Connection Issues

**Error**: `failed to connect to database`

**Solutions**:
- Ensure PostgreSQL is running: `docker compose ps` or `pg_isready`
- Check database credentials in `.env`
- Verify database exists: `psql -U postgres -l`

### Port Already in Use

**Error**: `bind: address already in use`

**Solutions**:
- Change the port in `.env`: `PORT=3000`
- Kill the process using the port:
  ```bash
  # Find process
  lsof -i :8080
  # Kill it
  kill -9 <PID>
  ```

### Frontend Can't Connect to Backend

**Solutions**:
- Ensure backend is running on port 8080
- Check CORS settings in `internal/middleware/middleware.go`
- Verify proxy settings in `frontend/vite.config.ts`

### Migrations Not Running

**Solutions**:
- Delete and recreate the database:
  ```bash
  dropdb volunteer_media_dev
  createdb volunteer_media_dev
  ```
- Run the application again

## Development Tips

### Hot Reload for Backend

Install `air` for automatic reload on file changes:

```bash
go install github.com/cosmtrek/air@latest
air
```

### View Database

```bash
# Using Docker
docker exec -it volunteer_media_db_dev psql -U postgres -d volunteer_media_dev

# Using local PostgreSQL
psql -U postgres -d volunteer_media_dev

# Common queries
\dt                    # List tables
SELECT * FROM users;   # View users
SELECT * FROM groups;  # View groups
```

### Reset Database

```bash
# Drop and recreate
docker compose down -v  # Removes volumes
docker compose up -d postgres_dev

# Or manually
dropdb volunteer_media_dev
createdb volunteer_media_dev
```

## Next Steps

- Read the main [README.md](README.md) for API documentation
- Check the [Dockerfile](Dockerfile) for production deployment details
- Review security considerations in the README

## Getting Help

If you encounter issues:

1. Check the logs: `docker compose logs -f` or view terminal output
2. Verify environment variables are set correctly
3. Ensure all prerequisites are installed
4. Check that ports 8080 and 5173 are available
5. Open an issue on GitHub with error details
