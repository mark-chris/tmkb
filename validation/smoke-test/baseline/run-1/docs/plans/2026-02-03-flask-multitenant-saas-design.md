# Flask Multi-Tenant SaaS File Upload API - Design Document

**Date:** 2026-02-03
**Status:** Approved

## Overview

A Flask-based API for a multi-tenant SaaS application with background file processing capabilities. Users belong to organizations and can upload generic files that are processed asynchronously using Celery.

## Key Requirements

- User authentication (username/password with Flask-Login)
- File upload endpoint with background processing
- List and view files (organization-scoped)
- Multi-tenant support (users belong to organizations)
- RESTful API with JSON responses
- Docker Compose development environment

## Architecture

### System Components

The application consists of five main components running in Docker containers:

1. **Flask API** - Handles HTTP requests, authentication, file uploads
2. **Celery Worker** - Processes background jobs for file metadata extraction
3. **Redis** - Message broker for Celery and result backend
4. **SQLite Database** - Stores users, organizations, files, and job metadata
5. **File Storage Volume** - Persistent local filesystem for uploaded files

### Request Flow

1. User authenticates via `/auth/login`, receives session cookie
2. Upload POST to `/files` returns immediately with file record (status: "pending")
3. Flask saves file temporarily and queues Celery task
4. Celery worker extracts metadata (MIME type, size, SHA256 hash)
5. Worker moves file to permanent storage and updates database (status: "completed")
6. User can list files via `/files` (paginated) or view individual file via `/files/<id>`

### Multi-Tenancy Model

- Each user belongs to exactly one organization (1:1 relationship)
- All database queries are automatically filtered by the user's organization
- Files are stored in organization-specific directories: `uploads/<org_id>/<file_id>`
- Session includes organization context to enforce data isolation

## Database Schema

### Organization Model
- `id` - Primary key (integer)
- `name` - Organization name (string, unique)
- `created_at` - Timestamp

### User Model
- `id` - Primary key (integer)
- `username` - Unique login identifier (string, unique)
- `password_hash` - bcrypt hashed password (string)
- `organization_id` - Foreign key to Organization
- `created_at` - Timestamp

### File Model
- `id` - Primary key (UUID string)
- `filename` - Original filename (string)
- `filepath` - Storage path relative to upload root (string)
- `organization_id` - Foreign key to Organization
- `uploaded_by` - Foreign key to User
- `status` - Enum: "pending", "processing", "completed", "failed"
- `file_size` - Bytes (integer, nullable, populated by background job)
- `mime_type` - Detected content type (string, nullable)
- `sha256_hash` - File hash for integrity/deduplication (string, nullable)
- `celery_task_id` - Task ID for job tracking (string, nullable)
- `error_message` - Error details if processing failed (text, nullable)
- `created_at` - Upload timestamp
- `processed_at` - When background job completed (timestamp, nullable)

### Database Indexes
- `User.username` (unique)
- `File.organization_id` (for tenant filtering)
- `File.uploaded_by` (for user's files)
- `File.status` (for filtering by processing state)

## API Endpoints

### Authentication

#### `POST /auth/login`
Authenticates user and creates session.

**Request:**
```json
{
  "username": "john",
  "password": "secret"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "user": {
    "id": 1,
    "username": "john",
    "organization_id": 1
  }
}
```

**Errors:**
- 401: Invalid credentials

Sets session cookie for subsequent requests.

#### `POST /auth/logout`
Clears user session.

**Response (200 OK):**
```json
{
  "success": true
}
```

### File Operations

#### `POST /files`
Upload a file for processing.

**Request:**
- Content-Type: `multipart/form-data`
- Field: `file` (binary file data)

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "document.pdf",
  "status": "pending",
  "created_at": "2026-02-03T19:30:00Z"
}
```

**Errors:**
- 401: Not authenticated
- 413: File too large (>50MB)
- 400: No file provided or invalid file

Returns immediately; file processing happens in background.

#### `GET /files?limit=20&offset=0`
List files belonging to user's organization.

**Query Parameters:**
- `limit` - Results per page (default: 20, max: 100)
- `offset` - Number of results to skip (default: 0)

**Response (200 OK):**
```json
{
  "files": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "filename": "document.pdf",
      "status": "completed",
      "file_size": 1024000,
      "mime_type": "application/pdf",
      "created_at": "2026-02-03T19:30:00Z",
      "processed_at": "2026-02-03T19:30:05Z"
    }
  ],
  "total": 150,
  "limit": 20,
  "offset": 0
}
```

**Errors:**
- 401: Not authenticated

Sorted by `created_at` descending (newest first).

#### `GET /files/<id>`
Get details for a specific file.

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "document.pdf",
  "file_size": 1024000,
  "mime_type": "application/pdf",
  "sha256_hash": "abc123...",
  "status": "completed",
  "uploaded_by": {
    "id": 1,
    "username": "john"
  },
  "created_at": "2026-02-03T19:30:00Z",
  "processed_at": "2026-02-03T19:30:05Z"
}
```

**Errors:**
- 401: Not authenticated
- 404: File not found or belongs to different organization

## Background Processing

### Celery Task: `process_file`

**Workflow:**

1. Flask saves file to temporary location (`/tmp/uploads/<uuid>`)
2. Creates File record with `status="pending"`
3. Queues Celery task: `process_file.delay(file_id)`
4. Returns response immediately to user

**Task Execution:**

```python
@celery.task
def process_file(file_id):
    # 1. Load File record from database
    # 2. Read file from temporary location
    # 3. Extract metadata:
    #    - Calculate SHA256 hash
    #    - Detect MIME type using python-magic
    #    - Get file size
    # 4. Move file to permanent storage:
    #    uploads/<org_id>/<file_id>_<filename>
    # 5. Update File record:
    #    - Set filepath, file_size, mime_type, sha256_hash
    #    - Set status="completed", processed_at=now()
    # 6. Delete temporary file
```

**Error Handling:**

- Task retries 3 times with exponential backoff (if Redis/DB temporarily unavailable)
- On final failure: set `status="failed"`, store error in `error_message`
- File remains in temp location for debugging (cleaned by daily cron job)
- Log all errors with file_id for traceability

**Configuration:**

- Task timeout: 5 minutes (handles large 50MB files on slow disk)
- Max retries: 3
- Retry delay: 2^retry seconds

## Security & Tenant Isolation

### Authentication (Flask-Login)

- Sessions stored server-side (Flask-Session with filesystem backend)
- Password hashing with bcrypt (cost factor 12)
- Login attempts rate-limited (5 attempts per minute per IP)
- Session timeout: 24 hours of inactivity
- Secure cookie flags: `HttpOnly=True, Secure=True (in production), SameSite=Lax`

### Tenant Isolation

Every database query automatically filters by organization:

```python
@login_required
def get_files():
    org_id = current_user.organization_id
    files = File.query.filter_by(organization_id=org_id).all()
```

### File Access Control

1. All file endpoints check `@login_required`
2. File queries include `organization_id=current_user.organization_id`
3. File paths include org_id: `uploads/<org_id>/<file_id>`
4. Before serving/accessing any file, verify ownership:

```python
file = File.query.get_or_404(file_id)
if file.organization_id != current_user.organization_id:
    abort(404)  # Don't reveal existence
```

### Upload Security

- 50MB size limit enforced by Flask config (`MAX_CONTENT_LENGTH`)
- Filename sanitization (remove path traversal attempts)
- Files stored by UUID, not original filename
- No direct file serving (files not in static directory)
- MIME type validation against detected type vs extension

### Additional Security Measures

- CORS disabled by default
- Rate limiting on upload endpoint (10 uploads/minute per user)
- No directory listing on storage paths
- All errors return generic messages (no information disclosure)

## Project Structure

```
test-flask-api/
├── app/
│   ├── __init__.py          # Flask app factory
│   ├── models.py            # SQLAlchemy models
│   ├── auth.py              # Authentication routes
│   ├── files.py             # File routes
│   ├── tasks.py             # Celery tasks
│   └── utils.py             # Helpers (tenant filter, file validation)
├── migrations/              # Flask-Migrate database migrations
├── uploads/                 # Persistent file storage (Docker volume)
├── config.py                # Configuration classes
├── wsgi.py                  # WSGI entry point
├── celery_worker.py         # Celery worker entry point
├── requirements.txt         # Python dependencies
├── docker-compose.yml       # Multi-container setup
├── Dockerfile               # Flask/Celery image
├── .env.example             # Environment variables template
└── .env                     # Environment variables (gitignored)
```

## Docker Configuration

### docker-compose.yml

```yaml
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  app:
    build: .
    ports:
      - "5000:5000"
    volumes:
      - ./uploads:/app/uploads
      - ./app:/app/app
    depends_on:
      - redis
    environment:
      - FLASK_ENV=development
      - CELERY_BROKER_URL=redis://redis:6379/0
      - CELERY_RESULT_BACKEND=redis://redis:6379/0

  celery:
    build: .
    command: celery -A celery_worker worker --loglevel=info
    volumes:
      - ./uploads:/app/uploads
      - ./app:/app/app
    depends_on:
      - redis
    environment:
      - CELERY_BROKER_URL=redis://redis:6379/0
      - CELERY_RESULT_BACKEND=redis://redis:6379/0
```

## Dependencies

### Core Dependencies
- Flask - Web framework
- Flask-Login - Session management
- Flask-SQLAlchemy - ORM
- Flask-Migrate - Database migrations
- Celery - Background task processing
- Redis - Message broker
- python-magic - MIME type detection
- bcrypt - Password hashing
- python-dotenv - Environment variable management

### Development Dependencies
- Flask debug mode for hot-reload
- SQLite for simple database setup

## Initialization & Development Workflow

### Initial Setup

```bash
# 1. Clone/create project
# 2. Copy environment template
cp .env.example .env

# 3. Start services
docker-compose up -d

# 4. Initialize database
docker-compose exec app flask db init
docker-compose exec app flask db migrate -m "Initial schema"
docker-compose exec app flask db upgrade

# 5. Seed initial data
docker-compose exec app flask seed-data
```

### Seed Data

Custom Flask CLI command creates sample data for testing:
- Organization: "Acme Corp"
- User: username="admin", password="password"

### Development Workflow

- Code changes in `app/` hot-reload automatically (Flask debug mode)
- Celery worker needs restart after code changes: `docker-compose restart celery`
- View logs: `docker-compose logs -f app celery`
- Access app: `http://localhost:5000`
- Redis CLI: `docker-compose exec redis redis-cli`

### Database Migrations

```bash
# After model changes
docker-compose exec app flask db migrate -m "description"
docker-compose exec app flask db upgrade
```

### Testing the API

```bash
# Login
curl -X POST http://localhost:5000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}' \
  -c cookies.txt

# Upload file
curl -X POST http://localhost:5000/files \
  -F "file=@test.pdf" \
  -b cookies.txt

# List files
curl http://localhost:5000/files?limit=10 -b cookies.txt

# View specific file
curl http://localhost:5000/files/<file-id> -b cookies.txt
```

## Future Considerations

### Not Included in MVP (but could be added)

- File deletion endpoint
- Job status endpoint to check processing progress
- User management (create/update/delete users)
- Organization management endpoints
- File download endpoint
- File search/filtering by name, type, date
- Storage quota limits per organization
- Email notifications when processing completes/fails
- Webhook support for processing events
- S3-compatible storage backend option
- PostgreSQL for production deployments
- API authentication via tokens (JWT) instead of sessions
- Chunked uploads for files >50MB
- Virus scanning integration (ClamAV)
- File preview generation (thumbnails for images/PDFs)

## Design Decisions & Trade-offs

### SQLite vs PostgreSQL
**Decision:** SQLite for MVP
**Rationale:** Simpler setup, single file database, sufficient for development and small deployments
**Trade-off:** Limited concurrency, not suitable for high-traffic production

### Local Filesystem vs S3
**Decision:** Local filesystem
**Rationale:** Simpler implementation, no cloud dependencies, easier debugging
**Trade-off:** Not horizontally scalable, requires shared storage for multi-server deployments

### Username/Password Only
**Decision:** No email, no password reset
**Rationale:** Simplifies implementation, reduces dependencies (no email service)
**Trade-off:** Less user-friendly, admin must reset passwords manually

### Users Belong to One Organization
**Decision:** 1:1 user-organization relationship
**Rationale:** Simpler data model, easier to enforce isolation, common SaaS pattern
**Trade-off:** Users can't collaborate across organizations

### 50MB Upload Limit
**Decision:** Single-request upload with 50MB limit
**Rationale:** Covers most use cases, simpler than chunked uploads
**Trade-off:** Large files may timeout on slow connections, no resumable uploads

### Minimal API Surface
**Decision:** Only login, upload, list, view endpoints
**Rationale:** Focus on core functionality, YAGNI principle
**Trade-off:** Missing convenience features like file deletion, job status checks

## Success Criteria

The implementation is successful if:

1. Users can log in with username/password
2. Authenticated users can upload files up to 50MB
3. Files are processed in the background (metadata extraction)
4. Users can list their organization's files with pagination
5. Users can view individual file details
6. Tenant isolation is enforced (users only see their org's files)
7. System runs via `docker-compose up`
8. Code is structured and maintainable
9. Database migrations work correctly
10. Basic error handling is in place
