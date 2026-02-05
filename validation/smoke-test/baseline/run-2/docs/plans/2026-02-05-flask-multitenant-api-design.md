# Flask Multi-Tenant SaaS API Design

**Date:** 2026-02-05
**Status:** Approved

## Overview

A Flask-based REST API for a multi-tenant SaaS platform with file upload capabilities and background job processing. Users belong to organizations (tenants), can upload files, and background workers validate files asynchronously.

## Architecture Overview

### Core Components

**Flask API Server** - Handles HTTP requests, serves API endpoints for authentication, file uploads, and file listing. Uses Flask-Login for session management with secure cookies. Runs on port 5000.

**PostgreSQL Database** - Stores users, organizations, and file metadata. Uses SQLite in development mode, PostgreSQL in production. Supports tenant isolation through organization_id foreign keys.

**Redis** - Serves two purposes: Celery message broker for task queuing, and session storage for Flask-Login (faster than database sessions, scales better).

**Celery Worker** - Background task processor that picks up file validation jobs from Redis queue. Runs file integrity checks, validates file sizes against limits, and updates file status from "pending" to "ready" or "failed".

**File Storage** - Local filesystem with configurable upload directory (e.g., `/app/uploads/{organization_id}/{file_id}`). Files are organized by tenant to maintain isolation.

**Docker Compose** - Orchestrates all services in development. Includes Flask app, Celery worker, Redis, and PostgreSQL containers with proper networking and volume mounts.

### Request Flow

User authenticates → uploads file via API → Flask saves file and metadata → Celery job queued → Worker validates file → Status updated → User queries file list/details.

## Data Model

### Organizations Table
- `id` (primary key)
- `name` (string, unique, required) - Organization/tenant name
- `created_at` (timestamp)
- `storage_quota_mb` (integer, default 1000) - Per-tenant storage limit

### Users Table
- `id` (primary key)
- `email` (string, unique, required)
- `password_hash` (string, required) - Bcrypt hashed password
- `organization_id` (foreign key to organizations, required)
- `is_active` (boolean, default True) - For soft account deactivation
- `created_at` (timestamp)

Each user belongs to exactly one organization. Authentication checks both credentials and active status.

### Files Table
- `id` (primary key, UUID)
- `filename` (string, required) - Original filename
- `file_path` (string, required) - Storage path on disk
- `file_size_bytes` (integer)
- `mime_type` (string)
- `status` (enum: 'pending', 'ready', 'failed', required)
- `error_message` (text, nullable) - If validation fails
- `organization_id` (foreign key, required) - Tenant isolation
- `uploaded_by` (foreign key to users, required)
- `uploaded_at` (timestamp)
- `processed_at` (timestamp, nullable) - When Celery job completed

All file queries are automatically scoped by `organization_id` to ensure tenant isolation.

## API Endpoints

### Authentication Endpoints

**POST /api/auth/register** - Create new user account
- Request: `{email, password, organization_name}`
- Response: `{user_id, email, organization_id}`
- Creates organization if new, or joins existing by name

**POST /api/auth/login** - Authenticate user
- Request: `{email, password}`
- Response: `{user_id, email, organization_id}`
- Sets secure session cookie via Flask-Login

**POST /api/auth/logout** - End session
- Response: `{success: true}`
- Clears session cookie

### File Endpoints (all require authentication)

**POST /api/files/upload** - Upload a file
- Request: multipart/form-data with file
- Response: `{file_id, filename, status: 'pending'}`
- Saves file, queues Celery validation job
- Enforces organization storage quota

**GET /api/files** - List all files for current user's organization
- Query params: `?page=1&limit=20`
- Response: `{files: [{id, filename, size, status, uploaded_at, uploaded_by_email}], total, page}`
- Filtered by organization_id automatically

**GET /api/files/<file_id>** - Get file details
- Response: `{id, filename, size, mime_type, status, error_message, uploaded_at, uploaded_by_email}`
- Returns 404 if file belongs to different organization

**GET /api/files/<file_id>/download** - Download file content
- Returns file with appropriate headers if status is 'ready'
- Blocks download if status is 'pending' or 'failed'

## Authentication Flow

### Session Management with Flask-Login

**LoginManager Configuration**
- Session cookie named `session`, httponly and secure flags enabled
- Cookie lifetime: 7 days (configurable)
- Sessions stored in Redis for performance and horizontal scaling
- Uses Flask-Session extension with Redis backend

**User Loader**
```python
@login_manager.user_loader
def load_user(user_id):
    return User.query.get(user_id)
```
Called on each request to load user from session cookie.

**Login Process**
1. User submits email/password to `/api/auth/login`
2. Backend queries user by email, verifies password with bcrypt
3. Checks `is_active` flag (reject if False)
4. Calls `login_user(user)` which creates session
5. Session ID stored in Redis with user_id
6. Secure cookie sent to client with session ID

**Request Authentication**
- Every protected endpoint decorated with `@login_required`
- Flask-Login reads session cookie, looks up session in Redis
- Loads user object via user_loader
- User object available as `current_user` in request context
- `current_user.organization_id` used for tenant scoping

**CSRF Protection**
- Flask-WTF provides CSRF tokens for state-changing requests
- GET requests exempt, POST/PUT/DELETE require valid token
- Token embedded in forms or passed as `X-CSRF-Token` header

## File Upload & Processing Flow

### Upload Request Flow

1. **Client uploads file** to `POST /api/files/upload`
   - Flask validates authentication and CSRF token
   - Checks organization's current storage usage against quota
   - Rejects if quota exceeded

2. **File saved to disk**
   - Generate UUID for file_id
   - Save to: `/uploads/{organization_id}/{file_id}/{original_filename}`
   - Directory structure ensures tenant isolation
   - Werkzeug's `secure_filename()` sanitizes filename

3. **Database record created**
   - Insert Files row with status='pending'
   - Record file_size_bytes, mime_type, uploaded_by

4. **Celery task queued**
   - `validate_file.delay(file_id)` sends task to Redis
   - Returns immediately to client with file_id and status='pending'
   - Client can poll `GET /api/files/{file_id}` for status updates

### Background Processing (Celery Worker)

The `validate_file` task performs:

1. **File integrity check** - Verify file exists and is readable
2. **Size validation** - Confirm file_size matches actual disk size
3. **MIME type verification** - Re-check MIME type matches extension
4. **Quota double-check** - Ensure organization still under quota

**On Success:**
- Update status='ready', set processed_at timestamp
- File becomes available for download

**On Failure:**
- Update status='failed', set error_message
- Optionally delete file from disk (configurable)

## Configuration & Environment

### Environment-based Configuration

**Base Config** - Shared settings
- `SECRET_KEY` - From environment variable (required)
- `UPLOAD_FOLDER` - File storage path (default: `/app/uploads`)
- `MAX_FILE_SIZE_MB` - Per-file limit (default: 100)
- `SESSION_TYPE` - 'redis'
- `PERMANENT_SESSION_LIFETIME` - 7 days
- `CELERY_BROKER_URL` - Redis connection string
- `CELERY_RESULT_BACKEND` - Redis connection string

**Development Config** (extends Base)
- `DEBUG = True`
- `SQLALCHEMY_DATABASE_URI` - SQLite file path
- `REDIS_URL` - `redis://localhost:6379/0`

**Production Config** (extends Base)
- `DEBUG = False`
- `SQLALCHEMY_DATABASE_URI` - PostgreSQL connection string from env
- `REDIS_URL` - Production Redis URL from env
- `SESSION_COOKIE_SECURE = True` - HTTPS only
- `SESSION_COOKIE_HTTPONLY = True`
- `SESSION_COOKIE_SAMESITE = 'Lax'`

### Environment Variables (.env file)
```bash
FLASK_ENV=development  # or production
SECRET_KEY=your-secret-key-here
DATABASE_URL=postgresql://user:pass@db:5432/flaskapp
REDIS_URL=redis://redis:6379/0
UPLOAD_FOLDER=/app/uploads
MAX_FILE_SIZE_MB=100
STORAGE_QUOTA_MB=1000
```

Docker Compose loads `.env` automatically. The app selects config based on `FLASK_ENV`.

## Project Structure

```
flask-api/
├── app/
│   ├── __init__.py           # Flask app factory, extensions init
│   ├── models.py             # SQLAlchemy models (User, Organization, File)
│   ├── auth/
│   │   ├── __init__.py
│   │   └── routes.py         # Auth endpoints (login, register, logout)
│   ├── files/
│   │   ├── __init__.py
│   │   └── routes.py         # File endpoints (upload, list, download)
│   └── tasks.py              # Celery tasks (validate_file)
├── config.py                 # Config classes (Base, Dev, Prod)
├── migrations/               # Flask-Migrate database migrations
├── tests/                    # Unit and integration tests
│   ├── test_auth.py
│   ├── test_files.py
│   └── test_tasks.py
├── uploads/                  # Local file storage (gitignored)
├── docker-compose.yml        # All services orchestration
├── Dockerfile                # Flask app container
├── requirements.txt          # Python dependencies
├── .env.example              # Environment variable template
├── .env                      # Actual env vars (gitignored)
└── wsgi.py                   # WSGI entry point
```

**Key Design Decisions:**
- Blueprint-based routing (auth, files) for modularity
- App factory pattern in `app/__init__.py` for testability
- Separate tasks.py for Celery to avoid circular imports
- Migrations managed by Flask-Migrate (Alembic wrapper)
- Uploads folder volume-mounted in Docker for persistence

## Docker Setup

### Docker Compose Services

**PostgreSQL Service**
- Image: `postgres:15-alpine`
- Environment: `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`
- Volume: `postgres_data:/var/lib/postgresql/data` (persistence)
- Port: 5432 (internal only, not exposed to host)

**Redis Service**
- Image: `redis:7-alpine`
- Port: 6379 (internal only)
- No persistence needed (sessions can be rebuilt)

**Flask App Service**
- Build: Custom Dockerfile with Python 3.11-slim
- Ports: `5000:5000` (exposed to host)
- Volumes:
  - `./app:/app/app` (code hot-reload in dev)
  - `./uploads:/app/uploads` (persistent file storage)
- Depends on: postgres, redis
- Command: `flask run --host=0.0.0.0`
- Environment: All config from .env file

**Celery Worker Service**
- Build: Same Dockerfile as Flask app
- Volumes: Same as Flask (needs access to uploads)
- Depends on: postgres, redis, web
- Command: `celery -A app.tasks worker --loglevel=info`
- No exposed ports

**Dockerfile Details**
```dockerfile
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
```

**Networking**
- All services on bridge network `flask_network`
- Services communicate via service names (e.g., `redis://redis:6379`)

## Security Considerations

- Passwords hashed with bcrypt
- Session cookies: httponly, secure (HTTPS only in prod), SameSite=Lax
- CSRF protection on state-changing requests
- File uploads: sanitized filenames, size limits, quota enforcement
- Tenant isolation: all queries scoped by organization_id
- SQL injection protection: SQLAlchemy ORM parameterized queries

## Next Steps for Implementation

1. Initialize git repository
2. Set up project structure and files
3. Create Docker Compose configuration
4. Implement models and database schema
5. Implement authentication endpoints
6. Implement file upload/download endpoints
7. Implement Celery background tasks
8. Write tests
9. Create README with setup instructions
