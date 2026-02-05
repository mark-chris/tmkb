# Flask Multi-Tenant SaaS API

A production-ready Flask-based REST API for a multi-tenant SaaS platform with secure file upload capabilities and asynchronous background job processing.

## Features

- **Multi-Tenant Architecture**: Complete tenant isolation with organization-based data partitioning
- **Authentication & Authorization**: Secure session-based authentication using Flask-Login with Redis-backed sessions
- **File Management**: Upload, list, and download files with automatic validation and storage quota enforcement
- **Background Processing**: Asynchronous file validation using Celery workers
- **RESTful API**: Clean, well-documented API endpoints following REST principles
- **Database Migrations**: Managed with Flask-Migrate (Alembic)
- **Docker Support**: Full Docker Compose setup for development and production
- **Comprehensive Tests**: Full test suite with pytest

## Architecture

### Components

- **Flask API Server**: Handles HTTP requests and serves REST API endpoints (port 5000)
- **PostgreSQL Database**: Stores users, organizations, and file metadata
- **Redis**: Message broker for Celery and session storage
- **Celery Workers**: Background task processors for file validation
- **Local File Storage**: Organized by tenant for isolation

### Tech Stack

- Python 3.11+
- Flask 3.0
- PostgreSQL 15
- Redis 7
- Celery 5.3
- SQLAlchemy (via Flask-SQLAlchemy)
- Docker & Docker Compose

## Prerequisites

- Docker and Docker Compose (recommended)
- OR Python 3.11+, PostgreSQL, and Redis (for local development)

## Quick Start with Docker

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd flask-api
   ```

2. **Create environment file**
   ```bash
   cp .env.example .env
   ```

   Edit `.env` and set your configuration values, especially `SECRET_KEY`.

3. **Start all services**
   ```bash
   docker-compose up -d
   ```

4. **Initialize the database**
   ```bash
   docker-compose exec web python init_db.py
   ```

5. **Access the API**

   The API is now running at `http://localhost:5000`

## Local Development Setup

### 1. Install Dependencies

```bash
# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install requirements
pip install -r requirements.txt
```

### 2. Set Up PostgreSQL and Redis

Install PostgreSQL and Redis on your system, then create a database:

```bash
createdb flaskapp
```

### 3. Configure Environment

Create a `.env` file:

```bash
FLASK_ENV=development
SECRET_KEY=your-secret-key-here
DATABASE_URL=postgresql://user:password@localhost:5432/flaskapp
REDIS_URL=redis://localhost:6379/0
UPLOAD_FOLDER=./uploads
MAX_FILE_SIZE_MB=100
STORAGE_QUOTA_MB=1000
```

### 4. Initialize Database

```bash
python init_db.py
```

### 5. Run the Application

In separate terminal windows:

```bash
# Terminal 1: Flask application
flask run

# Terminal 2: Celery worker
celery -A app.tasks worker --loglevel=info
```

## API Documentation

### Base URL

```
http://localhost:5000/api
```

### Authentication Endpoints

#### Register New User

```http
POST /api/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword123",
  "organization_name": "My Company"
}
```

**Response** (201 Created):
```json
{
  "user_id": 1,
  "email": "user@example.com",
  "organization_id": 1
}
```

#### Login

```http
POST /api/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response** (200 OK):
```json
{
  "user_id": 1,
  "email": "user@example.com",
  "organization_id": 1
}
```

Sets a secure session cookie for subsequent requests.

#### Logout

```http
POST /api/auth/logout
```

**Response** (200 OK):
```json
{
  "success": true
}
```

### File Endpoints

All file endpoints require authentication (session cookie).

#### Upload File

```http
POST /api/files/upload
Content-Type: multipart/form-data

file: <binary-file-data>
```

**Response** (201 Created):
```json
{
  "file_id": "123e4567-e89b-12d3-a456-426614174000",
  "filename": "document.pdf",
  "status": "pending"
}
```

The file status will be "pending" until the background worker validates it.

#### List Files

```http
GET /api/files?page=1&limit=20
```

**Response** (200 OK):
```json
{
  "files": [
    {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "filename": "document.pdf",
      "size": 1048576,
      "status": "ready",
      "uploaded_at": "2026-02-05T10:30:00Z",
      "uploaded_by_email": "user@example.com"
    }
  ],
  "total": 1,
  "page": 1
}
```

#### Get File Details

```http
GET /api/files/{file_id}
```

**Response** (200 OK):
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "filename": "document.pdf",
  "size": 1048576,
  "mime_type": "application/pdf",
  "status": "ready",
  "error_message": null,
  "uploaded_at": "2026-02-05T10:30:00Z",
  "uploaded_by_email": "user@example.com"
}
```

#### Download File

```http
GET /api/files/{file_id}/download
```

**Response** (200 OK):
Returns the file content with appropriate headers. Only works if file status is "ready".

### Error Responses

All endpoints return standard error responses:

```json
{
  "error": "Description of the error"
}
```

Common HTTP status codes:
- `400 Bad Request`: Invalid input or request
- `401 Unauthorized`: Authentication required
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

## Data Model

### Organizations

| Field | Type | Description |
|-------|------|-------------|
| id | Integer | Primary key |
| name | String | Organization name (unique) |
| storage_quota_mb | Integer | Storage limit in MB (default: 1000) |
| created_at | DateTime | Creation timestamp |

### Users

| Field | Type | Description |
|-------|------|-------------|
| id | Integer | Primary key |
| email | String | Email address (unique) |
| password_hash | String | Bcrypt hashed password |
| organization_id | Integer | Foreign key to organizations |
| is_active | Boolean | Account status (default: true) |
| created_at | DateTime | Creation timestamp |

### Files

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Primary key |
| filename | String | Original filename |
| file_path | String | Storage path on disk |
| file_size_bytes | Integer | File size in bytes |
| mime_type | String | MIME type |
| status | Enum | 'pending', 'ready', or 'failed' |
| error_message | Text | Error description (if failed) |
| organization_id | Integer | Foreign key to organizations |
| uploaded_by | Integer | Foreign key to users |
| uploaded_at | DateTime | Upload timestamp |
| processed_at | DateTime | Processing completion timestamp |

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| FLASK_ENV | Environment (development/production) | development |
| SECRET_KEY | Flask secret key (required) | - |
| DATABASE_URL | PostgreSQL connection string | - |
| REDIS_URL | Redis connection string | redis://localhost:6379/0 |
| UPLOAD_FOLDER | File storage directory | /app/uploads |
| MAX_FILE_SIZE_MB | Per-file size limit | 100 |
| STORAGE_QUOTA_MB | Per-tenant storage quota | 1000 |

### Configuration Classes

The application uses environment-based configuration in `config.py`:

- **DevelopmentConfig**: SQLite database, debug mode enabled
- **ProductionConfig**: PostgreSQL database, secure cookies, debug disabled

## Testing

### Run All Tests

```bash
# With Docker
docker-compose exec web pytest

# Local
pytest
```

### Run Specific Test File

```bash
pytest tests/test_auth.py
```

### Run with Coverage

```bash
pytest --cov=app --cov-report=html
```

Coverage report will be generated in `htmlcov/index.html`.

### Test Structure

- `tests/conftest.py`: Pytest fixtures and test configuration
- `tests/test_auth.py`: Authentication endpoint tests
- `tests/test_files.py`: File management and tenant isolation tests

## Security Features

- **Password Hashing**: Bcrypt with automatic salt generation
- **Session Security**: HttpOnly, Secure (HTTPS), and SameSite cookies
- **CSRF Protection**: Flask-WTF CSRF tokens on state-changing requests
- **Tenant Isolation**: Automatic query filtering by organization_id
- **Input Validation**: Request validation and sanitization
- **File Security**: Filename sanitization, size limits, quota enforcement

## Background Tasks

### File Validation Task

When a file is uploaded, a Celery task validates it:

1. **File Integrity Check**: Verify file exists and is readable
2. **Size Validation**: Confirm file size matches metadata
3. **MIME Type Verification**: Re-check MIME type consistency
4. **Quota Check**: Ensure organization stays under quota

On success, file status changes to "ready". On failure, status becomes "failed" with an error message.

### Task Monitoring

Monitor Celery workers:

```bash
# Docker
docker-compose logs -f worker

# Local
celery -A app.tasks worker --loglevel=info
```

## Project Structure

```
flask-api/
├── app/
│   ├── __init__.py           # Flask app factory
│   ├── models.py             # SQLAlchemy models
│   ├── tasks.py              # Celery tasks
│   ├── auth/
│   │   ├── __init__.py
│   │   └── routes.py         # Authentication endpoints
│   └── files/
│       ├── __init__.py
│       └── routes.py         # File management endpoints
├── tests/
│   ├── __init__.py
│   ├── conftest.py           # Pytest fixtures
│   ├── test_auth.py          # Auth tests
│   └── test_files.py         # File tests
├── uploads/                  # File storage (gitignored)
├── config.py                 # Configuration classes
├── init_db.py                # Database initialization script
├── wsgi.py                   # WSGI entry point
├── docker-compose.yml        # Docker orchestration
├── Dockerfile                # Container definition
├── requirements.txt          # Python dependencies
├── pytest.ini                # Pytest configuration
├── .env.example              # Environment template
├── .env                      # Environment variables (gitignored)
└── README.md                 # This file
```

## Deployment

### Production Checklist

1. **Environment Variables**: Set secure values in production `.env`
   - Generate strong `SECRET_KEY`: `python -c "import secrets; print(secrets.token_hex(32))"`
   - Use production database URLs
   - Set `FLASK_ENV=production`

2. **Database**: Use managed PostgreSQL service (AWS RDS, GCP Cloud SQL, etc.)

3. **Redis**: Use managed Redis service (AWS ElastiCache, Redis Cloud, etc.)

4. **File Storage**: Consider cloud storage (S3, GCS) for scalability

5. **SSL/TLS**: Ensure HTTPS is enabled (use reverse proxy like Nginx)

6. **Monitoring**: Set up logging and monitoring (Sentry, DataDog, etc.)

7. **Backups**: Configure automated database and file backups

### Docker Production Deployment

Build and run with production settings:

```bash
# Set production environment variables
export FLASK_ENV=production
export DATABASE_URL=postgresql://...
export REDIS_URL=redis://...

# Build and start
docker-compose -f docker-compose.yml up -d

# Initialize database
docker-compose exec web python init_db.py
```

## Troubleshooting

### Database Connection Issues

- Verify PostgreSQL is running: `docker-compose ps`
- Check connection string in `.env`
- Ensure database exists: `docker-compose exec db psql -U postgres -l`

### Redis Connection Issues

- Check Redis is running: `docker-compose ps`
- Test connection: `docker-compose exec redis redis-cli ping`

### File Upload Failures

- Check upload directory permissions
- Verify storage quota hasn't been exceeded
- Check Celery worker logs: `docker-compose logs worker`

### Session Issues

- Clear browser cookies
- Verify Redis is running and accessible
- Check `SECRET_KEY` is set and consistent

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make changes and add tests
4. Run tests: `pytest`
5. Commit changes: `git commit -am 'Add new feature'`
6. Push to branch: `git push origin feature/my-feature`
7. Submit a pull request

## License

This project is licensed under the MIT License.

## Support

For issues and questions:
- Open an issue on GitHub
- Check existing documentation
- Review test files for usage examples

## Changelog

### Version 1.0.0 (2026-02-05)

- Initial release
- Multi-tenant architecture with organization-based isolation
- User authentication and session management
- File upload, listing, and download functionality
- Background file validation with Celery
- Comprehensive test suite
- Docker Compose setup for development
- Full API documentation
