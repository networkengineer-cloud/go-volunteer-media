# go-volunteer-media

Social Media app for volunteers to get updates, share photos and experiences with animals at the shelter.

## Features

- **User Authentication**: Secure login and registration with JWT tokens
- **Password Reset**: Self-service password reset via email (Resend or SMTP)
- **Group Management**: Organize volunteers into groups (dogs, cats, modsquad, etc.)
- **Animal CRUD**: Create, read, update, and delete animal profiles within groups
- **Image Gallery**: Upload and manage multiple images per animal with profile picture support
- **Protocol Documents**: Upload and share protocol documents (PDF/DOCX) for animals
- **Flexible Storage**: Feature-flagged storage with PostgreSQL (default) or Azure Blob Storage
- **Bulk Animal Management**: Bulk edit, CSV import/export for efficient animal management
- **Updates Feed**: Share experiences and photos with group members
- **Email Notifications**: Announcement emails to users (configurable per user)
- **Admin Controls**: Admin users can create groups and manage user memberships
- **Secure Deployment**: Docker-based deployment with non-root user

## Technology Stack

### Backend
- **Go** (1.24+) - Backend API
- **Gin** - HTTP web framework
- **GORM** - ORM for database operations
- **PostgreSQL** - Database (dev and prod)
- **JWT** - Authentication tokens
- **bcrypt** - Password hashing

### Frontend
- **React** (18+) - UI library
- **TypeScript** - Type safety
- **Vite** - Build tool
- **React Router** - Client-side routing
- **Axios** - HTTP client

## Getting Started

### Prerequisites

- Go 1.24+
- Node.js 20+
- PostgreSQL 15+
- Docker (optional, for containerized deployment)

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/networkengineer-cloud/go-volunteer-media.git
   cd go-volunteer-media
   ```

2. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Start PostgreSQL** (using Docker Compose)
   ```bash
   docker-compose up -d postgres_dev
   ```

4. **Seed the database with demo data** (Optional but recommended for demos)
   ```bash
   make seed
   ```
   This creates demo users, animals, and content. See [SEED_DATA.md](SEED_DATA.md) for details.
   
   Demo credentials:
   - Username: `admin` / Password: `demo1234`
   - See [SEED_DATA.md](SEED_DATA.md) for all demo users

5. **Run the backend**
   ```bash
   go run cmd/api/main.go
   ```
   The API will be available at http://localhost:8080

6. **Run the frontend** (in a separate terminal)
   ```bash
   cd frontend
   npm install
   npm run dev
   ```
   The frontend will be available at http://localhost:5173

### Database Configuration

#### Development Database
- Host: localhost
- Port: 5432
- Database: volunteer_media_dev
- User: postgres
- Password: postgres

#### Production Database
Configure via environment variables:
- `DB_HOST` - Database host
- `DB_PORT` - Database port (default: 5432)
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name (volunteer_media_prod)
- `DB_SSLMODE` - SSL mode (require for production)

### Default Groups

The application creates three default groups on first run:
- **dogs** - Dog volunteers group
- **cats** - Cat volunteers group
- **modsquad** - Moderators group

## API Endpoints

### Authentication
- `POST /api/register` - Register a new user
- `POST /api/login` - Login and get JWT token
- `GET /api/me` - Get current user info (authenticated)

### Groups
- `GET /api/groups` - Get all accessible groups (authenticated)
- `GET /api/groups/:id` - Get group details (authenticated)
- `POST /api/admin/groups` - Create new group (admin only)
- `PUT /api/admin/groups/:id` - Update group (admin only)
- `DELETE /api/admin/groups/:id` - Delete group (admin only)

### Animals
- `GET /api/groups/:groupId/animals` - Get all animals in a group
- `GET /api/groups/:groupId/animals/:id` - Get animal details
- `POST /api/groups/:groupId/animals` - Create new animal
- `PUT /api/groups/:groupId/animals/:id` - Update animal
- `DELETE /api/groups/:groupId/animals/:id` - Delete animal

### Updates
- `GET /api/groups/:groupId/updates` - Get all updates for a group
- `POST /api/groups/:groupId/updates` - Create new update

### User-Group Management (Admin Only)
- `POST /api/admin/users/:userId/groups/:groupId` - Add user to group
- `DELETE /api/admin/users/:userId/groups/:groupId` - Remove user from group

## Docker Deployment

### Build the Docker image
```bash
docker build -t volunteer-media .
```

### Run with Docker Compose
```bash
docker-compose up -d
```

This will start:
- PostgreSQL database (dev)
- API service on port 8080

### Production Deployment

1. Build the frontend:
   ```bash
   cd frontend
   npm run build
   ```

2. Build the Docker image:
   ```bash
   docker build -t volunteer-media:prod .
   ```

3. Run with production environment variables:
   ```bash
   docker run -d \
     -p 8080:8080 \
     -e ENV=production \
     -e DB_HOST=your-prod-db-host \
     -e DB_PASSWORD=your-secure-password \
     -e JWT_SECRET=your-secret-key \
     volunteer-media:prod
   ```

## Security Features

- JWT-based authentication
- Password hashing with bcrypt
- Role-based access control (user/admin)
- Group-based permissions
- Secure Dockerfile with non-root user
- Multi-stage Docker build
- No sensitive data in container image

## Storage Architecture

The application supports flexible storage for images and documents through a feature-flagged system:

### Storage Providers

1. **PostgreSQL (Default)**: Stores binary data directly in the database
   - Zero configuration required
   - Works out of the box
   - Best for small to medium deployments

2. **Azure Blob Storage (Recommended for Production)**: Offloads binary storage to Azure
   - Significantly reduces database size and costs
   - Better performance and scalability
   - Support for future video files
   - CDN integration ready

### Configuration

Set the storage provider via environment variable:

```bash
# Use PostgreSQL (default)
STORAGE_PROVIDER=postgres

# Use Azure Blob Storage
STORAGE_PROVIDER=azure
AZURE_STORAGE_ACCOUNT_NAME=youraccount
AZURE_STORAGE_ACCOUNT_KEY=yourkey
AZURE_STORAGE_CONTAINER_NAME=volunteer-media-storage
```

### Local Testing with Azurite

Test Azure Blob Storage locally using Azurite:

```bash
# Start Azurite
docker run -p 10000:10000 mcr.microsoft.com/azure-storage/azurite

# Configure environment
export STORAGE_PROVIDER=azure
export AZURE_STORAGE_ACCOUNT_NAME=devstoreaccount1
export AZURE_STORAGE_ACCOUNT_KEY=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==
export AZURE_STORAGE_CONTAINER_NAME=volunteer-media-storage
export AZURE_STORAGE_ENDPOINT=http://127.0.0.1:10000/devstoreaccount1

# Start application
make dev-backend
```

For detailed documentation, see [STORAGE.md](STORAGE.md).

## Development

### Project Structure
```
.
├── cmd/
│   └── api/              # Application entry point
├── internal/
│   ├── auth/             # Authentication logic
│   ├── database/         # Database connection and migrations
│   ├── handlers/         # HTTP request handlers
│   ├── middleware/       # HTTP middleware
│   └── models/           # Data models
├── frontend/
│   ├── src/
│   │   ├── api/          # API client
│   │   ├── components/   # React components
│   │   ├── contexts/     # React contexts
│   │   └── pages/        # Page components
│   └── ...
├── Dockerfile            # Production Docker image
├── docker-compose.yml    # Development environment
└── README.md
```

### Testing the Application

#### With Demo Data (Recommended)

1. Seed the database: `make seed`
2. Login at http://localhost:5173/login with demo credentials:
   - Admin: `admin` / `demo1234`
   - Or any other demo user (see [SEED_DATA.md](SEED_DATA.md))
3. Explore the pre-populated groups, animals, and content

#### Without Demo Data

1. Register a new user at http://localhost:5173/login
2. Login with your credentials
3. As a new user, you won't see any groups (contact admin to be added)
4. Admin users can:
   - Create new groups
   - Add users to groups
   - Manage all content

### Making the First Admin

#### With Demo Data
The seed command creates an admin user automatically:
- Username: `admin`
- Password: `demo1234`

#### Without Demo Data
To make a user an admin, update the database directly:
```sql
UPDATE users SET is_admin = true WHERE username = 'your-username';
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

### Third-Party Licenses

This project uses various open-source libraries. For detailed information about third-party licenses and attributions, see [THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md).

All dependencies use licenses compatible with the MIT license (MIT, Apache-2.0, BSD, ISC). No GPL or copyleft licenses are used.
