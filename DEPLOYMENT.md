# Deployment Guide

This guide covers deploying the Haws Volunteers application to production.

## Security Checklist

Before deploying to production, ensure you have:

- [ ] Changed the `JWT_SECRET` to a strong, random value
- [ ] Set strong database passwords
- [ ] Enabled SSL/TLS for database connections (`DB_SSLMODE=require`)
- [ ] Configured CORS to allow only your frontend domain
- [ ] Set `ENV=production`
- [ ] Reviewed and restricted database user permissions
- [ ] Configured firewall rules to restrict access
- [ ] Set up HTTPS for the application
- [ ] Configured backup strategy for the database
- [ ] Set up monitoring and logging

## Environment Variables for Production

Create a production `.env` file or set environment variables:

```env
# Application
ENV=production
PORT=8080

# Security - CHANGE THESE!
JWT_SECRET=<generate-strong-random-secret>

# Production Database
DB_HOST=<your-production-db-host>
DB_PORT=5432
DB_USER=<db-user>
DB_PASSWORD=<strong-password>
DB_NAME=volunteer_media_prod
DB_SSLMODE=require
```

Generate a strong JWT secret:
```bash
openssl rand -base64 32
```

## Deployment Options

### Option 1: Docker Deployment

#### Prerequisites
- Docker installed on the server
- Docker Compose (optional)
- PostgreSQL database (managed service or self-hosted)

#### Steps

1. **Build the frontend:**
```bash
cd frontend
npm install
npm run build
cd ..
```

2. **Build the Docker image:**
```bash
docker build -t volunteer-media:1.0.0 .
```

3. **Run the container:**
```bash
docker run -d \
  --name volunteer-media \
  -p 8080:8080 \
  -e ENV=production \
  -e JWT_SECRET="your-secret-key" \
  -e DB_HOST="your-db-host" \
  -e DB_PORT="5432" \
  -e DB_USER="postgres" \
  -e DB_PASSWORD="your-password" \
  -e DB_NAME="volunteer_media_prod" \
  -e DB_SSLMODE="require" \
  --restart unless-stopped \
  volunteer-media:1.0.0
```

4. **Check logs:**
```bash
docker logs -f volunteer-media
```

#### Using Docker Compose

Create `docker-compose.prod.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: volunteer_media_prod
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped
    
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      ENV: production
      JWT_SECRET: ${JWT_SECRET}
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: ${DB_USER}
      DB_PASSWORD: ${DB_PASSWORD}
      DB_NAME: volunteer_media_prod
      DB_SSLMODE: disable  # Use 'require' if using external DB
    depends_on:
      - postgres
    restart: unless-stopped

volumes:
  postgres_data:
```

Deploy:
```bash
docker compose -f docker-compose.prod.yml up -d
```

### Option 2: Binary Deployment

#### Prerequisites
- Go 1.24+ installed on server
- PostgreSQL database
- systemd (for service management)

#### Steps

1. **Build the backend:**
```bash
# On your development machine or CI/CD
cd /path/to/go-volunteer-media
go build -o volunteer-media-api ./cmd/api

# Or with optimizations
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o volunteer-media-api ./cmd/api
```

2. **Build the frontend:**
```bash
cd frontend
npm install
npm run build
cd ..
```

3. **Copy files to server:**
```bash
# Create directory structure
ssh user@server 'mkdir -p /opt/volunteer-media'

# Copy binary
scp volunteer-media-api user@server:/opt/volunteer-media/

# Copy frontend build
scp -r frontend/dist user@server:/opt/volunteer-media/frontend/

# Copy environment file
scp .env.production user@server:/opt/volunteer-media/.env
```

4. **Create systemd service:**
```bash
sudo nano /etc/systemd/system/volunteer-media.service
```

Content:
```ini
[Unit]
Description=Haws Volunteers API
After=network.target postgresql.service

[Service]
Type=simple
User=volunteer-media
Group=volunteer-media
WorkingDirectory=/opt/volunteer-media
EnvironmentFile=/opt/volunteer-media/.env
ExecStart=/opt/volunteer-media/volunteer-media-api
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

5. **Start the service:**
```bash
# Create user
sudo useradd -r -s /bin/false volunteer-media
sudo chown -R volunteer-media:volunteer-media /opt/volunteer-media

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable volunteer-media
sudo systemctl start volunteer-media

# Check status
sudo systemctl status volunteer-media
```

### Option 3: Cloud Platform Deployment

#### Heroku

1. Create `Procfile`:
```
web: ./volunteer-media-api
```

2. Create `heroku.yml`:
```yaml
build:
  docker:
    web: Dockerfile
```

3. Deploy:
```bash
heroku create your-app-name
heroku addons:create heroku-postgresql:hobby-dev
heroku config:set JWT_SECRET="your-secret"
git push heroku main
```

#### AWS (ECS/Fargate)

1. Push Docker image to ECR
2. Create ECS task definition
3. Configure RDS PostgreSQL
4. Deploy to Fargate
5. Configure ALB for HTTPS

#### Google Cloud Run

1. Build and push to Container Registry:
```bash
gcloud builds submit --tag gcr.io/PROJECT_ID/volunteer-media
```

2. Deploy:
```bash
gcloud run deploy volunteer-media \
  --image gcr.io/PROJECT_ID/volunteer-media \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars JWT_SECRET=your-secret,DB_HOST=your-db
```

## Database Setup

### PostgreSQL Production Setup

1. **Create production database:**
```sql
CREATE DATABASE volunteer_media_prod;
CREATE USER volunteer_media WITH ENCRYPTED PASSWORD 'strong-password';
GRANT ALL PRIVILEGES ON DATABASE volunteer_media_prod TO volunteer_media;
```

2. **Configure SSL:**
```sql
ALTER SYSTEM SET ssl = on;
SELECT pg_reload_conf();
```

3. **Tune for production:**
```sql
-- In postgresql.conf
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
max_connections = 100
```

### Managed Database Services

Consider using managed PostgreSQL services:

- **AWS RDS**: Automated backups, read replicas
- **Google Cloud SQL**: High availability, automatic updates
- **Digital Ocean Managed Databases**: Simple setup
- **Heroku Postgres**: Easy integration

## Reverse Proxy Setup

### Nginx Configuration

```nginx
server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}
```

### SSL Certificate with Let's Encrypt

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.com
```

## Monitoring and Logging

### Application Logs

View logs based on deployment method:

```bash
# Docker
docker logs -f volunteer-media

# Systemd
sudo journalctl -u volunteer-media -f

# Cloud platforms
gcloud logging read "resource.type=cloud_run_revision"  # Google Cloud
heroku logs --tail  # Heroku
```

### Health Checks

Add a health check endpoint (optional enhancement):

```go
// In cmd/api/main.go
router.GET("/health", func(c *gin.Context) {
    c.JSON(200, gin.H{"status": "healthy"})
})
```

### Monitoring Tools

Consider implementing:

- **Prometheus**: Metrics collection
- **Grafana**: Visualization
- **Sentry**: Error tracking
- **DataDog**: Full-stack monitoring

## Backup Strategy

### Database Backups

#### Automated Daily Backups
```bash
#!/bin/bash
# backup.sh
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/var/backups/postgres"
mkdir -p $BACKUP_DIR

pg_dump -h localhost -U volunteer_media -d volunteer_media_prod | \
  gzip > $BACKUP_DIR/backup_$DATE.sql.gz

# Keep only last 30 days
find $BACKUP_DIR -name "backup_*.sql.gz" -mtime +30 -delete
```

Add to crontab:
```bash
0 2 * * * /path/to/backup.sh
```

#### Managed Service Backups
- Enable automated backups in your managed database service
- Configure backup retention period
- Test restore procedures regularly

## Scaling

### Horizontal Scaling

The application is stateless and can be scaled horizontally:

1. Run multiple instances behind a load balancer
2. Share the same database
3. Use session storage for JWT (currently stateless)

### Database Scaling

- **Read Replicas**: For read-heavy workloads
- **Connection Pooling**: Use PgBouncer
- **Vertical Scaling**: Increase database resources

## Troubleshooting

### Application Won't Start

Check logs and verify:
- Database is accessible
- Environment variables are set
- Port 8080 is available
- File permissions are correct

### Database Connection Issues

```bash
# Test database connection
psql "postgresql://user:pass@host:5432/dbname?sslmode=require"

# Check network connectivity
telnet db-host 5432
```

### High Memory Usage

- Monitor with `docker stats` or `htop`
- Consider increasing container memory limits
- Check for database connection leaks

## Security Best Practices

1. **Keep Software Updated**
   - Regularly update Go dependencies
   - Update base Docker images
   - Apply PostgreSQL security patches

2. **Principle of Least Privilege**
   - Database user should only have necessary permissions
   - Use non-root user in containers
   - Restrict network access

3. **Secure Secrets Management**
   - Use environment variables, not hardcoded values
   - Consider secret management tools (HashiCorp Vault, AWS Secrets Manager)
   - Rotate credentials regularly

4. **Network Security**
   - Use firewall rules to restrict access
   - Enable database SSL/TLS
   - Use VPC/private networks when possible

5. **Regular Security Audits**
   - Run `go list -m all | nancy sleuth` for vulnerability scanning
   - Review access logs
   - Monitor for suspicious activity

## CI/CD Pipeline

Example GitHub Actions workflow:

```yaml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Build Docker image
        run: docker build -t volunteer-media:${{ github.sha }} .
      
      - name: Push to registry
        run: |
          echo ${{ secrets.REGISTRY_PASSWORD }} | docker login -u ${{ secrets.REGISTRY_USER }} --password-stdin
          docker push volunteer-media:${{ github.sha }}
      
      - name: Deploy to production
        run: |
          # Your deployment commands here
```

## Rollback Procedure

If issues occur after deployment:

1. **Docker:**
```bash
docker stop volunteer-media
docker run -d --name volunteer-media volunteer-media:previous-version
```

2. **Systemd:**
```bash
sudo systemctl stop volunteer-media
# Replace binary with previous version
sudo systemctl start volunteer-media
```

3. **Cloud platforms:**
```bash
# Heroku
heroku releases:rollback

# Google Cloud Run
gcloud run services update-traffic volunteer-media --to-revisions=previous-revision=100
```

## Support

For deployment issues:
- Check application logs
- Review this guide
- Open an issue on GitHub with deployment details
