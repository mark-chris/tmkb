# Flask Multi-Tenant SaaS API

A secure Flask-based multi-tenant SaaS API with file uploads and background job processing using Celery. Built with security-first design to prevent cross-tenant data access.

## Features

- **Multi-tenant architecture** with strict tenant isolation
- **User authentication** using Flask-Login
- **File upload** with organization-scoped storage
- **Background job processing** with Celery for file metadata extraction
- **Comprehensive security** addressing OWASP and TMKB threats

## Security Architecture

This application implements multiple layers of security to prevent tenant data leakage:

### 1. TenantScopedMixin (TMKB-AUTHZ-004)
- Automatic tenant filtering on all queries
- Background jobs require explicit `tenant_id`
- Prevents accidental cross-tenant queries

### 2. Background Job Authorization (TMKB-AUTHZ-001)
- Re-validates all authorization in Celery tasks
- Checks user still exists and belongs to organization
- Verifies file ownership and deletion status

### 3. Explicit Relationship Modeling (TMKB-AUTHZ-005)
- Separate tracking of user-in-org, org-owns-resource, user-owns-resource
- Each relationship explicitly checked

### 4. Organization-Scoped File Storage
- Files stored in isolated directories per organization
- UUID-based filenames prevent conflicts
- Path traversal protection

## Tech Stack

- **Flask** - Web framework
- **Flask-SQLAlchemy** - ORM
- **Flask-Login** - Authentication
- **PostgreSQL** - Database
- **Celery** - Background jobs
- **Redis** - Celery broker
- **Pillow** - Image processing

## Prerequisites

- Python 3.9+
- PostgreSQL
- Redis

## Installation

1. Clone the repository:
```bash
cd /home/mark/Projects/test-flask-api-tmkb
```

2. Install dependencies:
```bash
pip install -r requirements.txt
```

3. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Create PostgreSQL database:
```bash
createdb flask_saas
```

5. Initialize database:
```bash
export FLASK_APP=wsgi.py
flask db init
flask db migrate -m "Initial migration"
flask db upgrade
```

6. Create storage directories:
```bash
mkdir -p storage/organizations
```

## Running the Application

### Start Redis
```bash
redis-server
```

### Start Celery Worker (in separate terminal)
```bash
celery -A celery_worker.celery worker --loglevel=info
```

### Start Flask Application
```bash
python wsgi.py
```

The API will be available at `http://localhost:5000`

## API Endpoints

### Authentication

**POST /auth/login**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**POST /auth/logout**

**GET /auth/me**

### Files

**POST /files** - Upload file
- Content-Type: multipart/form-data
- Body: file (binary)

**GET /files** - List all files for current organization

**GET /files/{id}** - Get file metadata

**GET /files/{id}/download** - Download file

**DELETE /files/{id}** - Soft delete file

## Testing

Run the test suite:
```bash
pytest -v
```

Run security tests only:
```bash
pytest tests/test_security.py -v
```

## Database Setup Script

Create initial organizations and users:

```python
from app import create_app
from app.extensions import db
from app.models.organization import Organization
from app.models.user import User

app = create_app()

with app.app_context():
    # Create organizations
    org1 = Organization(name='Acme Corp')
    org2 = Organization(name='TechStart Inc')
    db.session.add_all([org1, org2])
    db.session.commit()

    # Create users
    user1 = User(
        email='alice@acmecorp.com',
        organization_id=org1.id,
        is_active=True
    )
    user1.set_password('password123')

    user2 = User(
        email='bob@techstart.com',
        organization_id=org2.id,
        is_active=True
    )
    user2.set_password('password123')

    db.session.add_all([user1, user2])
    db.session.commit()

    print("Organizations and users created successfully!")
```

Save as `setup_db.py` and run:
```bash
python setup_db.py
```

## Example Usage

1. **Login**
```bash
curl -X POST http://localhost:5000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "alice@acmecorp.com", "password": "password123"}' \
  -c cookies.txt
```

2. **Upload File**
```bash
curl -X POST http://localhost:5000/files \
  -b cookies.txt \
  -F "file=@/path/to/image.jpg"
```

3. **List Files**
```bash
curl http://localhost:5000/files -b cookies.txt
```

4. **Download File**
```bash
curl http://localhost:5000/files/1/download -b cookies.txt -o downloaded_file.jpg
```

## Security Testing

The application includes comprehensive security tests covering:

- **Tenant Isolation**: Users cannot access files from other organizations
- **Background Job Security**: Celery tasks re-validate all authorization
- **Input Validation**: File type and size restrictions
- **Path Traversal Protection**: Malicious filenames sanitized

Run security tests:
```bash
pytest tests/test_security.py -v
```

## Project Structure

```
test-flask-api-tmkb/
├── app/
│   ├── __init__.py              # Flask app factory
│   ├── config.py                # Configuration
│   ├── extensions.py            # Flask extensions
│   ├── models/
│   │   ├── base.py             # TenantScopedMixin (critical security)
│   │   ├── user.py             # User model
│   │   ├── organization.py      # Organization model
│   │   └── file.py             # File model
│   ├── auth/
│   │   └── routes.py           # Authentication endpoints
│   ├── files/
│   │   ├── routes.py           # File endpoints
│   │   └── storage.py          # File storage utilities
│   └── tasks/
│       └── file_processing.py  # Celery tasks
├── tests/
│   ├── conftest.py             # Test fixtures
│   ├── test_auth.py            # Auth tests
│   ├── test_files.py           # File tests
│   └── test_security.py        # Security tests (critical)
├── storage/                     # File storage
├── requirements.txt
├── wsgi.py                     # WSGI entry point
└── celery_worker.py            # Celery worker
```

## Security Principles

1. **Never trust client input** - `organization_id` always from `current_user`
2. **Defense in depth** - Multiple layers enforce tenant isolation
3. **Explicit relationships** - Separate user-in-org, org-owns-resource, user-owns-resource
4. **Re-validate in jobs** - Background tasks check authorization independently
5. **Fail securely** - Errors block access rather than allowing it

## License

MIT
