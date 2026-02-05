# Flask Multi-Tenant SaaS File Upload API

A production-ready Flask API with multi-tenant file uploads, background processing via Celery, and secure authentication.

## Features

- **Multi-tenant architecture** - Organizations with isolated data
- **User authentication** - Username/password with Flask-Login
- **File uploads** - Up to 50MB with background processing
- **Background jobs** - Celery workers extract file metadata (size, MIME type, SHA256)
- **RESTful API** - JSON responses with pagination
- **Docker-based** - Easy deployment with docker-compose

## Tech Stack

- Flask 3.0 - Web framework
- SQLAlchemy - ORM with SQLite
- Celery - Background task processing
- Redis - Message broker
- Docker - Containerization

## Quick Start

### 1. Setup Environment

```bash
# Copy environment template
cp .env.example .env

# Edit .env and set FLASK_SECRET_KEY to a random string
```

### 2. Start Services

```bash
# Build and start containers
docker-compose up -d

# Check logs
docker-compose logs -f
```

### 3. Initialize Database

```bash
# Run initialization script
docker-compose exec app bash init_db.sh
```

This creates:
- Database schema with migrations
- Sample organization: "Acme Corp"
- Sample user: username=`admin`, password=`password`

### 4. Test the API

```bash
# Login
curl -X POST http://localhost:5000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}' \
  -c cookies.txt

# Upload a file
curl -X POST http://localhost:5000/files \
  -F "file=@README.md" \
  -b cookies.txt

# List files
curl http://localhost:5000/files \
  -b cookies.txt

# Get file details
curl http://localhost:5000/files/<file-id> \
  -b cookies.txt
```

## Architecture

### Components

- **Flask App** (port 5000) - HTTP API, authentication, file uploads
- **Celery Worker** - Background file processing
- **Redis** (port 6379) - Message broker for Celery

### Request Flow

1. User authenticates → receives session cookie
2. User uploads file → saved to temp, queued for processing
3. Celery worker extracts metadata → moves to permanent storage
4. User can list/view files (scoped to organization)

## Development

### Project Structure

```
test-flask-api/
├── app/
│   ├── __init__.py      # Flask factory
│   ├── models.py        # SQLAlchemy models
│   ├── auth.py          # Auth routes
│   ├── files.py         # File routes
│   ├── tasks.py         # Celery tasks
│   ├── utils.py         # Helpers
│   └── cli.py           # CLI commands
├── config.py            # Configuration
├── wsgi.py              # Flask entry point
├── celery_worker.py     # Celery entry point
├── docker-compose.yml   # Multi-container setup
└── Dockerfile           # Container image
```

### Database Migrations

```bash
# Create migration after model changes
docker-compose exec app flask db migrate -m "description"

# Apply migrations
docker-compose exec app flask db upgrade
```

### Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f app
docker-compose logs -f celery
```

### Restart Services

```bash
# Restart after code changes
docker-compose restart app celery

# Rebuild after dependency changes
docker-compose up -d --build
```

## API Documentation

See [docs/API.md](docs/API.md) for detailed API documentation.

## Security Features

- **Password hashing** - bcrypt with cost factor 12
- **Session management** - Server-side sessions with 24hr timeout
- **Rate limiting** - Login (5/min), Upload (10/min)
- **Tenant isolation** - Organization-scoped queries
- **File sanitization** - Filename validation, path traversal prevention
- **Secure uploads** - 50MB limit, MIME type validation

## License

MIT
