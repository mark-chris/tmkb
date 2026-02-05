# Flask Multi-Tenant SaaS API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a production-ready Flask API with multi-tenant file uploads, background processing via Celery, and secure authentication.

**Architecture:** Flask handles HTTP/auth, Celery processes files asynchronously via Redis, SQLite stores multi-tenant data with organization-scoped queries, Docker Compose orchestrates services.

**Tech Stack:** Flask, Flask-Login, Flask-SQLAlchemy, Flask-Migrate, Celery, Redis, SQLite, Docker, python-magic, bcrypt

---

## Task 1: Project Foundation & Dependencies

**Files:**
- Create: `requirements.txt`
- Create: `.env.example`
- Create: `config.py`

**Step 1: Create requirements.txt**

```txt
Flask==3.0.0
Flask-Login==0.6.3
Flask-SQLAlchemy==3.1.1
Flask-Migrate==4.0.5
Flask-Session==0.5.0
Flask-Limiter==3.5.0
celery==5.3.4
redis==5.0.1
bcrypt==4.1.2
python-magic==0.4.27
python-dotenv==1.0.0
```

**Step 2: Create .env.example**

```env
FLASK_ENV=development
FLASK_SECRET_KEY=change-this-to-random-secret-key
DATABASE_URL=sqlite:///app.db
CELERY_BROKER_URL=redis://redis:6379/0
CELERY_RESULT_BACKEND=redis://redis:6379/0
MAX_CONTENT_LENGTH=52428800
UPLOAD_FOLDER=/app/uploads
TEMP_FOLDER=/tmp/uploads
```

**Step 3: Create config.py**

```python
import os
from dotenv import load_dotenv

load_dotenv()

class Config:
    """Base configuration"""
    SECRET_KEY = os.environ.get('FLASK_SECRET_KEY') or 'dev-secret-key-change-in-production'
    SQLALCHEMY_DATABASE_URI = os.environ.get('DATABASE_URL') or 'sqlite:///app.db'
    SQLALCHEMY_TRACK_MODIFICATIONS = False

    # File upload settings
    MAX_CONTENT_LENGTH = int(os.environ.get('MAX_CONTENT_LENGTH', 52428800))  # 50MB
    UPLOAD_FOLDER = os.environ.get('UPLOAD_FOLDER', 'uploads')
    TEMP_FOLDER = os.environ.get('TEMP_FOLDER', '/tmp/uploads')

    # Session settings
    SESSION_TYPE = 'filesystem'
    SESSION_PERMANENT = True
    PERMANENT_SESSION_LIFETIME = 86400  # 24 hours
    SESSION_COOKIE_HTTPONLY = True
    SESSION_COOKIE_SAMESITE = 'Lax'

    # Celery settings
    CELERY_BROKER_URL = os.environ.get('CELERY_BROKER_URL') or 'redis://localhost:6379/0'
    CELERY_RESULT_BACKEND = os.environ.get('CELERY_RESULT_BACKEND') or 'redis://localhost:6379/0'

    # Rate limiting
    RATELIMIT_STORAGE_URL = os.environ.get('CELERY_BROKER_URL') or 'redis://localhost:6379/0'

class DevelopmentConfig(Config):
    """Development configuration"""
    DEBUG = True
    SESSION_COOKIE_SECURE = False

class ProductionConfig(Config):
    """Production configuration"""
    DEBUG = False
    SESSION_COOKIE_SECURE = True

config = {
    'development': DevelopmentConfig,
    'production': ProductionConfig,
    'default': DevelopmentConfig
}
```

**Step 4: Commit**

```bash
git add requirements.txt .env.example config.py
git commit -m "feat: add project dependencies and configuration"
```

---

## Task 2: Docker Configuration

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `.dockerignore`

**Step 1: Create Dockerfile**

```dockerfile
FROM python:3.11-slim

# Install system dependencies for python-magic
RUN apt-get update && apt-get install -y \
    libmagic1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy requirements first for layer caching
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Create upload directories
RUN mkdir -p /app/uploads /tmp/uploads

# Expose Flask port
EXPOSE 5000

# Default command (overridden in docker-compose for celery)
CMD ["python", "wsgi.py"]
```

**Step 2: Create docker-compose.yml**

```yaml
version: '3.8'

services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  app:
    build: .
    ports:
      - "5000:5000"
    volumes:
      - ./uploads:/app/uploads
      - ./app:/app/app
      - ./instance:/app/instance
    depends_on:
      redis:
        condition: service_healthy
    environment:
      - FLASK_ENV=development
      - CELERY_BROKER_URL=redis://redis:6379/0
      - CELERY_RESULT_BACKEND=redis://redis:6379/0
    command: python wsgi.py

  celery:
    build: .
    volumes:
      - ./uploads:/app/uploads
      - ./app:/app/app
      - ./instance:/app/instance
    depends_on:
      redis:
        condition: service_healthy
    environment:
      - CELERY_BROKER_URL=redis://redis:6379/0
      - CELERY_RESULT_BACKEND=redis://redis:6379/0
    command: celery -A celery_worker worker --loglevel=info
```

**Step 3: Create .dockerignore**

```
__pycache__
*.pyc
*.pyo
*.pyd
.Python
env/
venv/
.env
*.db
*.sqlite
*.sqlite3
.git
.gitignore
instance/
uploads/
flask_session/
```

**Step 4: Commit**

```bash
git add Dockerfile docker-compose.yml .dockerignore
git commit -m "feat: add Docker configuration for multi-container setup"
```

---

## Task 3: Database Models

**Files:**
- Create: `app/__init__.py`
- Create: `app/models.py`

**Step 1: Create app/__init__.py (Flask factory)**

```python
from flask import Flask
from flask_sqlalchemy import SQLAlchemy
from flask_migrate import Migrate
from flask_login import LoginManager
from flask_session import Session
from flask_limiter import Limiter
from flask_limiter.util import get_remote_address
from config import config
import os

db = SQLAlchemy()
migrate = Migrate()
login_manager = LoginManager()
sess = Session()
limiter = Limiter(
    key_func=get_remote_address,
    default_limits=["200 per day", "50 per hour"]
)

def create_app(config_name=None):
    """Application factory pattern"""
    if config_name is None:
        config_name = os.environ.get('FLASK_ENV', 'development')

    app = Flask(__name__)
    app.config.from_object(config[config_name])

    # Initialize extensions
    db.init_app(app)
    migrate.init_app(app, db)
    login_manager.init_app(app)
    sess.init_app(app)
    limiter.init_app(app)

    # Create upload directories
    os.makedirs(app.config['UPLOAD_FOLDER'], exist_ok=True)
    os.makedirs(app.config['TEMP_FOLDER'], exist_ok=True)

    # Register blueprints (will add later)
    from app.auth import auth_bp
    from app.files import files_bp

    app.register_blueprint(auth_bp, url_prefix='/auth')
    app.register_blueprint(files_bp, url_prefix='/files')

    return app
```

**Step 2: Create app/models.py**

```python
from app import db, login_manager
from datetime import datetime
import bcrypt
import uuid

class Organization(db.Model):
    """Organization model for multi-tenancy"""
    __tablename__ = 'organizations'

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(255), unique=True, nullable=False, index=True)
    created_at = db.Column(db.DateTime, default=datetime.utcnow, nullable=False)

    # Relationships
    users = db.relationship('User', backref='organization', lazy='dynamic', cascade='all, delete-orphan')
    files = db.relationship('File', backref='organization', lazy='dynamic', cascade='all, delete-orphan')

    def __repr__(self):
        return f'<Organization {self.name}>'

    def to_dict(self):
        return {
            'id': self.id,
            'name': self.name,
            'created_at': self.created_at.isoformat()
        }

class User(db.Model):
    """User model with bcrypt password hashing"""
    __tablename__ = 'users'

    id = db.Column(db.Integer, primary_key=True)
    username = db.Column(db.String(80), unique=True, nullable=False, index=True)
    password_hash = db.Column(db.String(128), nullable=False)
    organization_id = db.Column(db.Integer, db.ForeignKey('organizations.id'), nullable=False, index=True)
    created_at = db.Column(db.DateTime, default=datetime.utcnow, nullable=False)

    # Relationships
    files = db.relationship('File', backref='uploader', lazy='dynamic', foreign_keys='File.uploaded_by')

    def __repr__(self):
        return f'<User {self.username}>'

    def set_password(self, password):
        """Hash password using bcrypt"""
        self.password_hash = bcrypt.hashpw(password.encode('utf-8'), bcrypt.gensalt(12)).decode('utf-8')

    def check_password(self, password):
        """Verify password against hash"""
        return bcrypt.checkpw(password.encode('utf-8'), self.password_hash.encode('utf-8'))

    @property
    def is_active(self):
        """Required by Flask-Login"""
        return True

    @property
    def is_authenticated(self):
        """Required by Flask-Login"""
        return True

    @property
    def is_anonymous(self):
        """Required by Flask-Login"""
        return False

    def get_id(self):
        """Required by Flask-Login"""
        return str(self.id)

    def to_dict(self):
        return {
            'id': self.id,
            'username': self.username,
            'organization_id': self.organization_id
        }

@login_manager.user_loader
def load_user(user_id):
    """User loader for Flask-Login"""
    return User.query.get(int(user_id))

class File(db.Model):
    """File model for uploaded files"""
    __tablename__ = 'files'

    id = db.Column(db.String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    filename = db.Column(db.String(255), nullable=False)
    filepath = db.Column(db.String(512), nullable=True)
    organization_id = db.Column(db.Integer, db.ForeignKey('organizations.id'), nullable=False, index=True)
    uploaded_by = db.Column(db.Integer, db.ForeignKey('users.id'), nullable=False, index=True)
    status = db.Column(db.String(20), default='pending', nullable=False, index=True)
    file_size = db.Column(db.Integer, nullable=True)
    mime_type = db.Column(db.String(100), nullable=True)
    sha256_hash = db.Column(db.String(64), nullable=True)
    celery_task_id = db.Column(db.String(36), nullable=True)
    error_message = db.Column(db.Text, nullable=True)
    created_at = db.Column(db.DateTime, default=datetime.utcnow, nullable=False, index=True)
    processed_at = db.Column(db.DateTime, nullable=True)

    def __repr__(self):
        return f'<File {self.filename} ({self.status})>'

    def to_dict(self, include_uploader=False):
        """Serialize file to dictionary"""
        result = {
            'id': self.id,
            'filename': self.filename,
            'status': self.status,
            'file_size': self.file_size,
            'mime_type': self.mime_type,
            'sha256_hash': self.sha256_hash,
            'created_at': self.created_at.isoformat(),
            'processed_at': self.processed_at.isoformat() if self.processed_at else None
        }

        if include_uploader and self.uploader:
            result['uploaded_by'] = {
                'id': self.uploader.id,
                'username': self.uploader.username
            }

        if self.status == 'failed' and self.error_message:
            result['error_message'] = self.error_message

        return result
```

**Step 3: Commit**

```bash
git add app/__init__.py app/models.py
git commit -m "feat: add database models for multi-tenant file uploads"
```

---

## Task 4: Authentication Routes

**Files:**
- Create: `app/auth.py`
- Create: `app/utils.py`

**Step 1: Create app/utils.py (helper functions)**

```python
from functools import wraps
from flask import abort
from flask_login import current_user
import os
import re

def require_org_access(model_instance):
    """Verify user has access to resource based on organization"""
    if not hasattr(model_instance, 'organization_id'):
        return True

    if model_instance.organization_id != current_user.organization_id:
        abort(404)  # Don't reveal existence

    return True

def sanitize_filename(filename):
    """Remove path traversal attempts and dangerous characters"""
    # Remove path separators
    filename = os.path.basename(filename)

    # Remove any non-alphanumeric characters except dots, dashes, underscores
    filename = re.sub(r'[^\w\s.-]', '', filename)

    # Limit length
    if len(filename) > 255:
        name, ext = os.path.splitext(filename)
        filename = name[:255-len(ext)] + ext

    return filename

def allowed_file(filename, allowed_extensions=None):
    """Check if file extension is allowed"""
    if allowed_extensions is None:
        # Allow all files by default (MIME validation happens in background)
        return '.' in filename

    return '.' in filename and \
           filename.rsplit('.', 1)[1].lower() in allowed_extensions
```

**Step 2: Create app/auth.py**

```python
from flask import Blueprint, request, jsonify, current_app
from flask_login import login_user, logout_user, login_required, current_user
from app import db, limiter
from app.models import User
import logging

auth_bp = Blueprint('auth', __name__)
logger = logging.getLogger(__name__)

@auth_bp.route('/login', methods=['POST'])
@limiter.limit("5 per minute")
def login():
    """Authenticate user and create session"""
    data = request.get_json()

    if not data or not data.get('username') or not data.get('password'):
        return jsonify({'error': 'Username and password required'}), 400

    username = data['username']
    password = data['password']

    # Find user
    user = User.query.filter_by(username=username).first()

    if not user or not user.check_password(password):
        logger.warning(f"Failed login attempt for username: {username}")
        return jsonify({'error': 'Invalid credentials'}), 401

    # Create session
    login_user(user, remember=False)

    logger.info(f"User {username} logged in successfully")

    return jsonify({
        'success': True,
        'user': user.to_dict()
    }), 200

@auth_bp.route('/logout', methods=['POST'])
@login_required
def logout():
    """Clear user session"""
    username = current_user.username
    logout_user()

    logger.info(f"User {username} logged out")

    return jsonify({'success': True}), 200

@auth_bp.route('/me', methods=['GET'])
@login_required
def get_current_user():
    """Get current authenticated user info"""
    return jsonify({
        'user': current_user.to_dict()
    }), 200
```

**Step 3: Commit**

```bash
git add app/auth.py app/utils.py
git commit -m "feat: add authentication routes with rate limiting"
```

---

## Task 5: File Upload Endpoint

**Files:**
- Create: `app/files.py`

**Step 1: Create app/files.py**

```python
from flask import Blueprint, request, jsonify, current_app
from flask_login import login_required, current_user
from werkzeug.utils import secure_filename
from app import db, limiter
from app.models import File
from app.utils import sanitize_filename, require_org_access
from app.tasks import process_file
import os
import uuid
import logging

files_bp = Blueprint('files', __name__)
logger = logging.getLogger(__name__)

@files_bp.route('', methods=['POST'])
@login_required
@limiter.limit("10 per minute")
def upload_file():
    """Upload file and queue for background processing"""
    # Check if file is in request
    if 'file' not in request.files:
        return jsonify({'error': 'No file provided'}), 400

    file = request.files['file']

    # Check if filename is empty
    if file.filename == '':
        return jsonify({'error': 'No file selected'}), 400

    try:
        # Sanitize filename
        original_filename = sanitize_filename(file.filename)

        # Generate unique file ID
        file_id = str(uuid.uuid4())

        # Save to temporary location
        temp_path = os.path.join(current_app.config['TEMP_FOLDER'], file_id)
        file.save(temp_path)

        # Create database record
        file_record = File(
            id=file_id,
            filename=original_filename,
            organization_id=current_user.organization_id,
            uploaded_by=current_user.id,
            status='pending'
        )

        db.session.add(file_record)
        db.session.commit()

        # Queue background task
        task = process_file.delay(file_id)

        # Update with task ID
        file_record.celery_task_id = task.id
        db.session.commit()

        logger.info(f"File {file_id} uploaded by user {current_user.username}, task {task.id} queued")

        return jsonify(file_record.to_dict()), 201

    except Exception as e:
        logger.error(f"Error uploading file: {str(e)}")
        db.session.rollback()
        return jsonify({'error': 'Failed to upload file'}), 500

@files_bp.route('', methods=['GET'])
@login_required
def list_files():
    """List files for user's organization with pagination"""
    # Parse pagination parameters
    try:
        limit = min(int(request.args.get('limit', 20)), 100)
        offset = int(request.args.get('offset', 0))
    except ValueError:
        return jsonify({'error': 'Invalid pagination parameters'}), 400

    # Query files for organization
    query = File.query.filter_by(organization_id=current_user.organization_id)

    # Get total count
    total = query.count()

    # Get paginated results (newest first)
    files = query.order_by(File.created_at.desc()).limit(limit).offset(offset).all()

    return jsonify({
        'files': [f.to_dict() for f in files],
        'total': total,
        'limit': limit,
        'offset': offset
    }), 200

@files_bp.route('/<file_id>', methods=['GET'])
@login_required
def get_file(file_id):
    """Get details for a specific file"""
    file = File.query.get_or_404(file_id)

    # Verify organization access
    require_org_access(file)

    return jsonify(file.to_dict(include_uploader=True)), 200
```

**Step 2: Commit**

```bash
git add app/files.py
git commit -m "feat: add file upload and list endpoints with pagination"
```

---

## Task 6: Celery Background Tasks

**Files:**
- Create: `app/tasks.py`
- Create: `celery_worker.py`

**Step 1: Create app/tasks.py**

```python
from celery import Celery
from app import create_app, db
from app.models import File
from datetime import datetime
import os
import hashlib
import magic
import logging
import shutil

# Create Celery instance
celery = Celery(__name__)

# Configure Celery from Flask config
flask_app = create_app()
celery.conf.update(
    broker_url=flask_app.config['CELERY_BROKER_URL'],
    result_backend=flask_app.config['CELERY_RESULT_BACKEND'],
    task_serializer='json',
    accept_content=['json'],
    result_serializer='json',
    timezone='UTC',
    enable_utc=True,
)

logger = logging.getLogger(__name__)

@celery.task(bind=True, max_retries=3, default_retry_delay=2)
def process_file(self, file_id):
    """
    Background task to process uploaded file

    Steps:
    1. Load file record from database
    2. Read file from temporary location
    3. Extract metadata (size, MIME type, SHA256 hash)
    4. Move file to permanent storage
    5. Update database record
    6. Clean up temporary file
    """
    with flask_app.app_context():
        try:
            # Load file record
            file_record = File.query.get(file_id)
            if not file_record:
                logger.error(f"File {file_id} not found in database")
                return {'status': 'error', 'message': 'File not found'}

            # Update status to processing
            file_record.status = 'processing'
            db.session.commit()

            # Get temp file path
            temp_path = os.path.join(flask_app.config['TEMP_FOLDER'], file_id)

            if not os.path.exists(temp_path):
                raise FileNotFoundError(f"Temporary file not found: {temp_path}")

            # Extract metadata
            file_size = os.path.getsize(temp_path)

            # Detect MIME type
            mime = magic.Magic(mime=True)
            mime_type = mime.from_file(temp_path)

            # Calculate SHA256 hash
            sha256_hash = hashlib.sha256()
            with open(temp_path, 'rb') as f:
                for chunk in iter(lambda: f.read(4096), b''):
                    sha256_hash.update(chunk)
            sha256_digest = sha256_hash.hexdigest()

            # Create permanent storage directory
            org_upload_dir = os.path.join(
                flask_app.config['UPLOAD_FOLDER'],
                str(file_record.organization_id)
            )
            os.makedirs(org_upload_dir, exist_ok=True)

            # Generate permanent filename
            permanent_filename = f"{file_id}_{file_record.filename}"
            permanent_path = os.path.join(org_upload_dir, permanent_filename)

            # Move file to permanent storage
            shutil.move(temp_path, permanent_path)

            # Update database record
            file_record.filepath = os.path.join(str(file_record.organization_id), permanent_filename)
            file_record.file_size = file_size
            file_record.mime_type = mime_type
            file_record.sha256_hash = sha256_digest
            file_record.status = 'completed'
            file_record.processed_at = datetime.utcnow()
            file_record.error_message = None

            db.session.commit()

            logger.info(f"Successfully processed file {file_id}")

            return {
                'status': 'success',
                'file_id': file_id,
                'file_size': file_size,
                'mime_type': mime_type,
                'sha256_hash': sha256_digest
            }

        except Exception as exc:
            logger.error(f"Error processing file {file_id}: {str(exc)}")

            # Update status to failed
            file_record = File.query.get(file_id)
            if file_record:
                file_record.status = 'failed'
                file_record.error_message = str(exc)
                db.session.commit()

            # Retry if within retry limit
            if self.request.retries < self.max_retries:
                raise self.retry(exc=exc, countdown=2 ** self.request.retries)

            return {
                'status': 'failed',
                'file_id': file_id,
                'error': str(exc)
            }
```

**Step 2: Create celery_worker.py**

```python
from app.tasks import celery, flask_app

# This file is the entry point for Celery worker
# Usage: celery -A celery_worker worker --loglevel=info

if __name__ == '__main__':
    with flask_app.app_context():
        celery.start()
```

**Step 3: Commit**

```bash
git add app/tasks.py celery_worker.py
git commit -m "feat: add Celery background processing for file metadata"
```

---

## Task 7: Flask Application Entry Points

**Files:**
- Create: `wsgi.py`
- Create: `app/cli.py`
- Modify: `app/__init__.py`

**Step 1: Create wsgi.py**

```python
from app import create_app, db
from app.models import Organization, User, File
import os

app = create_app()

# Make shell context for flask shell
@app.shell_context_processor
def make_shell_context():
    return {
        'db': db,
        'Organization': Organization,
        'User': User,
        'File': File
    }

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000, debug=True)
```

**Step 2: Create app/cli.py**

```python
import click
from flask.cli import with_appcontext
from app import db
from app.models import Organization, User

@click.command('seed-data')
@with_appcontext
def seed_data():
    """Seed database with initial test data"""
    # Create organization
    org = Organization.query.filter_by(name='Acme Corp').first()
    if not org:
        org = Organization(name='Acme Corp')
        db.session.add(org)
        db.session.flush()

    # Create user
    user = User.query.filter_by(username='admin').first()
    if not user:
        user = User(username='admin', organization_id=org.id)
        user.set_password('password')
        db.session.add(user)

    db.session.commit()

    click.echo(f'Created organization: {org.name} (ID: {org.id})')
    click.echo(f'Created user: {user.username} (password: password)')
    click.echo('Seed data created successfully!')

def init_app(app):
    """Register CLI commands"""
    app.cli.add_command(seed_data)
```

**Step 3: Modify app/__init__.py to register CLI commands**

Add this import at the top:
```python
from app import cli
```

Add this line before the return statement in `create_app()`:
```python
    # Register CLI commands
    cli.init_app(app)
```

**Step 4: Update app/__init__.py**

```python
from flask import Flask
from flask_sqlalchemy import SQLAlchemy
from flask_migrate import Migrate
from flask_login import LoginManager
from flask_session import Session
from flask_limiter import Limiter
from flask_limiter.util import get_remote_address
from config import config
import os

db = SQLAlchemy()
migrate = Migrate()
login_manager = LoginManager()
sess = Session()
limiter = Limiter(
    key_func=get_remote_address,
    default_limits=["200 per day", "50 per hour"]
)

def create_app(config_name=None):
    """Application factory pattern"""
    if config_name is None:
        config_name = os.environ.get('FLASK_ENV', 'development')

    app = Flask(__name__)
    app.config.from_object(config[config_name])

    # Initialize extensions
    db.init_app(app)
    migrate.init_app(app, db)
    login_manager.init_app(app)
    sess.init_app(app)
    limiter.init_app(app)

    # Create upload directories
    os.makedirs(app.config['UPLOAD_FOLDER'], exist_ok=True)
    os.makedirs(app.config['TEMP_FOLDER'], exist_ok=True)

    # Register blueprints
    from app.auth import auth_bp
    from app.files import files_bp

    app.register_blueprint(auth_bp, url_prefix='/auth')
    app.register_blueprint(files_bp, url_prefix='/files')

    # Register CLI commands
    from app import cli
    cli.init_app(app)

    return app
```

**Step 5: Commit**

```bash
git add wsgi.py app/cli.py app/__init__.py
git commit -m "feat: add Flask entry points and seed data CLI command"
```

---

## Task 8: Database Initialization Setup

**Files:**
- Create: `init_db.sh`

**Step 1: Create init_db.sh script**

```bash
#!/bin/bash
set -e

echo "Waiting for services to be ready..."
sleep 5

echo "Initializing database..."
flask db init || echo "Database already initialized"

echo "Creating migration..."
flask db migrate -m "Initial schema with multi-tenant support"

echo "Applying migrations..."
flask db upgrade

echo "Seeding initial data..."
flask seed-data

echo "Database setup complete!"
```

**Step 2: Make script executable**

Run: `chmod +x init_db.sh`

**Step 3: Commit**

```bash
git add init_db.sh
git commit -m "feat: add database initialization script"
```

---

## Task 9: Documentation and README

**Files:**
- Create: `README.md`
- Create: `docs/API.md`

**Step 1: Create README.md**

```markdown
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
```

**Step 2: Create docs/API.md**

```markdown
# API Documentation

Base URL: `http://localhost:5000`

## Authentication

All file endpoints require authentication via session cookie.

### POST /auth/login

Authenticate user and create session.

**Request:**
```json
{
  "username": "admin",
  "password": "password"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "user": {
    "id": 1,
    "username": "admin",
    "organization_id": 1
  }
}
```

**Errors:**
- `400` - Missing username or password
- `401` - Invalid credentials

**Rate Limit:** 5 requests per minute

---

### POST /auth/logout

Clear user session.

**Response (200 OK):**
```json
{
  "success": true
}
```

**Errors:**
- `401` - Not authenticated

---

### GET /auth/me

Get current authenticated user.

**Response (200 OK):**
```json
{
  "user": {
    "id": 1,
    "username": "admin",
    "organization_id": 1
  }
}
```

**Errors:**
- `401` - Not authenticated

---

## File Operations

### POST /files

Upload a file for background processing.

**Request:**
- Content-Type: `multipart/form-data`
- Field: `file` (binary file data)

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "document.pdf",
  "status": "pending",
  "file_size": null,
  "mime_type": null,
  "sha256_hash": null,
  "created_at": "2026-02-03T19:30:00.000000",
  "processed_at": null
}
```

**Errors:**
- `400` - No file provided or invalid file
- `401` - Not authenticated
- `413` - File too large (>50MB)

**Rate Limit:** 10 requests per minute

---

### GET /files

List files for authenticated user's organization.

**Query Parameters:**
- `limit` (optional) - Results per page (default: 20, max: 100)
- `offset` (optional) - Number of results to skip (default: 0)

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
      "sha256_hash": "abc123...",
      "created_at": "2026-02-03T19:30:00.000000",
      "processed_at": "2026-02-03T19:30:05.000000"
    }
  ],
  "total": 150,
  "limit": 20,
  "offset": 0
}
```

**Errors:**
- `400` - Invalid pagination parameters
- `401` - Not authenticated

**Notes:**
- Results sorted by `created_at` descending (newest first)
- Only shows files belonging to user's organization

---

### GET /files/<file_id>

Get details for a specific file.

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "document.pdf",
  "status": "completed",
  "file_size": 1024000,
  "mime_type": "application/pdf",
  "sha256_hash": "abc123...",
  "created_at": "2026-02-03T19:30:00.000000",
  "processed_at": "2026-02-03T19:30:05.000000",
  "uploaded_by": {
    "id": 1,
    "username": "admin"
  }
}
```

**Errors:**
- `401` - Not authenticated
- `404` - File not found or belongs to different organization

---

## File Status Values

- `pending` - File uploaded, waiting for processing
- `processing` - Background job is processing file
- `completed` - File processed successfully
- `failed` - Processing failed (see `error_message`)

---

## Error Responses

All errors follow this format:

```json
{
  "error": "Human-readable error message"
}
```

Common HTTP status codes:
- `400` - Bad Request (invalid input)
- `401` - Unauthorized (not authenticated)
- `404` - Not Found
- `413` - Payload Too Large (file >50MB)
- `429` - Too Many Requests (rate limited)
- `500` - Internal Server Error
```

**Step 3: Commit**

```bash
git add README.md docs/API.md
git commit -m "docs: add comprehensive README and API documentation"
```

---

## Task 10: Testing and Validation

**Files:**
- Create: `test_data/sample.txt`
- Create: `test_api.sh`

**Step 1: Create test file**

```bash
mkdir -p test_data
echo "This is a sample file for testing the upload API." > test_data/sample.txt
```

**Step 2: Create test_api.sh**

```bash
#!/bin/bash

BASE_URL="http://localhost:5000"
COOKIE_FILE="cookies.txt"

echo "=== Testing Flask Multi-Tenant API ==="
echo ""

echo "1. Login as admin user..."
curl -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}' \
  -c "$COOKIE_FILE" \
  -s | jq .

echo ""
echo "2. Get current user info..."
curl -X GET "$BASE_URL/auth/me" \
  -b "$COOKIE_FILE" \
  -s | jq .

echo ""
echo "3. Upload sample file..."
FILE_RESPONSE=$(curl -X POST "$BASE_URL/files" \
  -F "file=@test_data/sample.txt" \
  -b "$COOKIE_FILE" \
  -s)

echo "$FILE_RESPONSE" | jq .
FILE_ID=$(echo "$FILE_RESPONSE" | jq -r '.id')

echo ""
echo "4. Wait for background processing (3 seconds)..."
sleep 3

echo ""
echo "5. Get file details..."
curl -X GET "$BASE_URL/files/$FILE_ID" \
  -b "$COOKIE_FILE" \
  -s | jq .

echo ""
echo "6. List all files..."
curl -X GET "$BASE_URL/files?limit=5" \
  -b "$COOKIE_FILE" \
  -s | jq .

echo ""
echo "7. Logout..."
curl -X POST "$BASE_URL/auth/logout" \
  -b "$COOKIE_FILE" \
  -s | jq .

echo ""
echo "8. Test unauthorized access (should fail)..."
curl -X GET "$BASE_URL/files" \
  -s | jq .

echo ""
echo "=== Tests Complete ==="
```

**Step 3: Make script executable**

Run: `chmod +x test_api.sh`

**Step 4: Commit**

```bash
git add test_data/sample.txt test_api.sh
git commit -m "test: add API testing script and sample data"
```

---

## Execution Instructions

### Build and Start Services

```bash
# Build containers
docker-compose build

# Start all services
docker-compose up -d

# Check service health
docker-compose ps
```

### Initialize Database

```bash
# Run database setup
docker-compose exec app bash init_db.sh
```

Expected output:
- Database initialized
- Migrations created and applied
- Seed data created (Acme Corp organization, admin user)

### Verify Services

```bash
# Check Flask app logs
docker-compose logs app

# Check Celery worker logs
docker-compose logs celery

# Verify Redis is running
docker-compose exec redis redis-cli ping
# Should return: PONG
```

### Run API Tests

```bash
# Install jq if not present (for JSON formatting)
# sudo apt-get install jq  # Linux
# brew install jq          # macOS

# Run tests
./test_api.sh
```

Expected results:
1. Login succeeds, returns user info
2. File upload returns file record with "pending" status
3. After 3 seconds, file status should be "completed"
4. File details include metadata (size, MIME type, SHA256 hash)
5. List shows uploaded files for organization
6. Logout succeeds
7. Unauthorized request fails with 401

### Manual Testing

```bash
# Login
curl -X POST http://localhost:5000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}' \
  -c cookies.txt

# Upload file
curl -X POST http://localhost:5000/files \
  -F "file=@test_data/sample.txt" \
  -b cookies.txt

# List files
curl http://localhost:5000/files -b cookies.txt

# Get specific file (replace <file-id>)
curl http://localhost:5000/files/<file-id> -b cookies.txt
```

### Verify File Processing

```bash
# Check uploads directory structure
docker-compose exec app ls -lR uploads/

# Should show: uploads/1/<file-id>_sample.txt

# Check database
docker-compose exec app flask shell
>>> from app.models import File
>>> File.query.all()
>>> exit()
```

### Troubleshooting

**Services not starting:**
```bash
docker-compose logs app
docker-compose logs celery
docker-compose logs redis
```

**Database errors:**
```bash
# Reset database
docker-compose exec app rm instance/app.db
docker-compose exec app bash init_db.sh
```

**Celery not processing:**
```bash
# Check Redis connection
docker-compose exec app python -c "import redis; r = redis.Redis(host='redis'); print(r.ping())"

# Restart Celery worker
docker-compose restart celery
```

### Cleanup

```bash
# Stop services
docker-compose down

# Remove volumes (deletes database and uploads)
docker-compose down -v

# Remove all (including images)
docker-compose down -v --rmi all
```

---

## Success Criteria

✅ The implementation is complete when:

1. **Authentication works**
   - Users can login with username/password
   - Sessions persist across requests
   - Logout clears sessions

2. **File uploads work**
   - Files up to 50MB can be uploaded
   - Upload returns immediately with pending status
   - Files are saved to temporary location

3. **Background processing works**
   - Celery worker picks up tasks
   - Metadata extracted (size, MIME type, SHA256)
   - Files moved to permanent storage by organization
   - Database updated with completed status

4. **File listing works**
   - Users can list their organization's files
   - Pagination works correctly
   - Results sorted by newest first

5. **File details work**
   - Individual file details can be retrieved
   - Includes uploader information
   - Shows processing status and metadata

6. **Tenant isolation works**
   - Users only see their organization's files
   - Accessing other org's files returns 404
   - File storage organized by organization ID

7. **Docker setup works**
   - All services start with docker-compose up
   - Services can communicate
   - Logs are accessible

8. **Database migrations work**
   - Initial schema created
   - Seed data loads successfully
   - Future migrations can be applied

9. **Error handling works**
   - Invalid credentials return 401
   - Missing files return 400
   - Large files return 413
   - Rate limits enforced

10. **API test script passes**
    - All 8 test steps complete successfully
    - No errors in logs
    - Files processed within expected time
