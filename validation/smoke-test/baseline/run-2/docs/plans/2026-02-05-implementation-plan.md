# Flask Multi-Tenant SaaS API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a production-ready Flask multi-tenant SaaS API with file uploads and background processing.

**Architecture:** Session-based authentication with Flask-Login, SQLAlchemy models with tenant isolation, Celery workers for async file validation, Docker Compose for local development.

**Tech Stack:** Flask, SQLAlchemy, Flask-Login, Flask-Session, Celery, Redis, PostgreSQL, Docker

---

## Task 1: Project Foundation

**Files:**
- Create: `requirements.txt`
- Create: `config.py`
- Create: `.env.example`

**Step 1: Create requirements.txt**

```txt
Flask==3.0.0
Flask-SQLAlchemy==3.1.1
Flask-Login==0.6.3
Flask-Session==0.5.0
Flask-WTF==1.2.1
Flask-Migrate==4.0.5
celery==5.3.4
redis==5.0.1
psycopg2-binary==2.9.9
python-dotenv==1.0.0
bcrypt==4.1.2
```

**Step 2: Create config.py**

```python
import os
from datetime import timedelta

class Config:
    SECRET_KEY = os.environ.get('SECRET_KEY') or 'dev-secret-key-change-in-production'
    SQLALCHEMY_TRACK_MODIFICATIONS = False

    # Upload settings
    UPLOAD_FOLDER = os.environ.get('UPLOAD_FOLDER', '/app/uploads')
    MAX_FILE_SIZE_MB = int(os.environ.get('MAX_FILE_SIZE_MB', 100))
    MAX_CONTENT_LENGTH = MAX_FILE_SIZE_MB * 1024 * 1024

    # Session settings
    SESSION_TYPE = 'redis'
    SESSION_PERMANENT = True
    PERMANENT_SESSION_LIFETIME = timedelta(days=7)
    SESSION_COOKIE_HTTPONLY = True
    SESSION_COOKIE_SAMESITE = 'Lax'

    # Celery settings
    CELERY_BROKER_URL = os.environ.get('REDIS_URL', 'redis://localhost:6379/0')
    CELERY_RESULT_BACKEND = os.environ.get('REDIS_URL', 'redis://localhost:6379/0')

    # Storage quota
    STORAGE_QUOTA_MB = int(os.environ.get('STORAGE_QUOTA_MB', 1000))

class DevelopmentConfig(Config):
    DEBUG = True
    SQLALCHEMY_DATABASE_URI = os.environ.get('DATABASE_URL') or 'sqlite:///app.db'
    SESSION_REDIS = os.environ.get('REDIS_URL', 'redis://localhost:6379/0')

class ProductionConfig(Config):
    DEBUG = False
    SQLALCHEMY_DATABASE_URI = os.environ.get('DATABASE_URL')
    SESSION_REDIS = os.environ.get('REDIS_URL')
    SESSION_COOKIE_SECURE = True

config = {
    'development': DevelopmentConfig,
    'production': ProductionConfig,
    'default': DevelopmentConfig
}
```

**Step 3: Create .env.example**

```bash
FLASK_ENV=development
SECRET_KEY=your-secret-key-here
DATABASE_URL=postgresql://flask:flask@db:5432/flaskapp
REDIS_URL=redis://redis:6379/0
UPLOAD_FOLDER=/app/uploads
MAX_FILE_SIZE_MB=100
STORAGE_QUOTA_MB=1000
```

**Step 4: Commit**

```bash
git add requirements.txt config.py .env.example
git commit -m "feat: add project configuration and dependencies"
```

---

## Task 2: Database Models

**Files:**
- Create: `app/__init__.py`
- Create: `app/models.py`

**Step 1: Create app/__init__.py with Flask app factory**

```python
from flask import Flask
from flask_sqlalchemy import SQLAlchemy
from flask_login import LoginManager
from flask_session import Session
from flask_wtf.csrf import CSRFProtect
from flask_migrate import Migrate
from redis import Redis
from config import config

db = SQLAlchemy()
login_manager = LoginManager()
sess = Session()
csrf = CSRFProtect()
migrate = Migrate()

def create_app(config_name='default'):
    app = Flask(__name__)
    app.config.from_object(config[config_name])

    # Initialize extensions
    db.init_app(app)
    login_manager.init_app(app)
    sess.init_app(app)
    csrf.init_app(app)
    migrate.init_app(app, db)

    # Configure Flask-Login
    login_manager.login_view = 'auth.login'
    login_manager.login_message = 'Please log in to access this page.'

    # User loader
    from app.models import User

    @login_manager.user_loader
    def load_user(user_id):
        return User.query.get(int(user_id))

    # Register blueprints
    from app.auth import auth_bp
    from app.files import files_bp

    app.register_blueprint(auth_bp, url_prefix='/api/auth')
    app.register_blueprint(files_bp, url_prefix='/api/files')

    return app
```

**Step 2: Create app/models.py**

```python
import uuid
from datetime import datetime
from flask_login import UserMixin
from werkzeug.security import generate_password_hash, check_password_hash
from app import db

class Organization(db.Model):
    __tablename__ = 'organizations'

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(255), unique=True, nullable=False, index=True)
    storage_quota_mb = db.Column(db.Integer, default=1000)
    created_at = db.Column(db.DateTime, default=datetime.utcnow)

    users = db.relationship('User', backref='organization', lazy=True)
    files = db.relationship('File', backref='organization', lazy=True)

    def get_storage_used_mb(self):
        total_bytes = db.session.query(
            db.func.sum(File.file_size_bytes)
        ).filter(
            File.organization_id == self.id,
            File.status != 'failed'
        ).scalar() or 0
        return total_bytes / (1024 * 1024)

    def has_storage_quota(self, additional_mb):
        return (self.get_storage_used_mb() + additional_mb) <= self.storage_quota_mb

class User(UserMixin, db.Model):
    __tablename__ = 'users'

    id = db.Column(db.Integer, primary_key=True)
    email = db.Column(db.String(255), unique=True, nullable=False, index=True)
    password_hash = db.Column(db.String(255), nullable=False)
    organization_id = db.Column(db.Integer, db.ForeignKey('organizations.id'), nullable=False)
    is_active = db.Column(db.Boolean, default=True)
    created_at = db.Column(db.DateTime, default=datetime.utcnow)

    files = db.relationship('File', backref='uploader', lazy=True)

    def set_password(self, password):
        self.password_hash = generate_password_hash(password)

    def check_password(self, password):
        return check_password_hash(self.password_hash, password)

class File(db.Model):
    __tablename__ = 'files'

    id = db.Column(db.String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    filename = db.Column(db.String(255), nullable=False)
    file_path = db.Column(db.String(512), nullable=False)
    file_size_bytes = db.Column(db.Integer)
    mime_type = db.Column(db.String(127))
    status = db.Column(db.Enum('pending', 'ready', 'failed', name='file_status'), nullable=False, default='pending')
    error_message = db.Column(db.Text)
    organization_id = db.Column(db.Integer, db.ForeignKey('organizations.id'), nullable=False)
    uploaded_by = db.Column(db.Integer, db.ForeignKey('users.id'), nullable=False)
    uploaded_at = db.Column(db.DateTime, default=datetime.utcnow)
    processed_at = db.Column(db.DateTime)

    def to_dict(self):
        return {
            'id': self.id,
            'filename': self.filename,
            'size': self.file_size_bytes,
            'mime_type': self.mime_type,
            'status': self.status,
            'error_message': self.error_message,
            'uploaded_at': self.uploaded_at.isoformat() if self.uploaded_at else None,
            'processed_at': self.processed_at.isoformat() if self.processed_at else None,
            'uploaded_by_email': self.uploader.email
        }
```

**Step 3: Commit**

```bash
git add app/__init__.py app/models.py
git commit -m "feat: add database models and Flask app factory"
```

---

## Task 3: Authentication Blueprint

**Files:**
- Create: `app/auth/__init__.py`
- Create: `app/auth/routes.py`

**Step 1: Create app/auth/__init__.py**

```python
from flask import Blueprint

auth_bp = Blueprint('auth', __name__)

from app.auth import routes
```

**Step 2: Create app/auth/routes.py**

```python
from flask import request, jsonify
from flask_login import login_user, logout_user, login_required, current_user
from app import db
from app.auth import auth_bp
from app.models import User, Organization

@auth_bp.route('/register', methods=['POST'])
def register():
    data = request.get_json()

    if not data or not data.get('email') or not data.get('password') or not data.get('organization_name'):
        return jsonify({'error': 'Missing required fields'}), 400

    # Check if user already exists
    if User.query.filter_by(email=data['email']).first():
        return jsonify({'error': 'Email already registered'}), 400

    # Get or create organization
    org = Organization.query.filter_by(name=data['organization_name']).first()
    if not org:
        org = Organization(name=data['organization_name'])
        db.session.add(org)
        db.session.flush()

    # Create user
    user = User(
        email=data['email'],
        organization_id=org.id
    )
    user.set_password(data['password'])
    db.session.add(user)
    db.session.commit()

    return jsonify({
        'user_id': user.id,
        'email': user.email,
        'organization_id': user.organization_id
    }), 201

@auth_bp.route('/login', methods=['POST'])
def login():
    data = request.get_json()

    if not data or not data.get('email') or not data.get('password'):
        return jsonify({'error': 'Missing email or password'}), 400

    user = User.query.filter_by(email=data['email']).first()

    if not user or not user.check_password(data['password']):
        return jsonify({'error': 'Invalid email or password'}), 401

    if not user.is_active:
        return jsonify({'error': 'Account is inactive'}), 403

    login_user(user)

    return jsonify({
        'user_id': user.id,
        'email': user.email,
        'organization_id': user.organization_id
    }), 200

@auth_bp.route('/logout', methods=['POST'])
@login_required
def logout():
    logout_user()
    return jsonify({'success': True}), 200

@auth_bp.route('/me', methods=['GET'])
@login_required
def me():
    return jsonify({
        'user_id': current_user.id,
        'email': current_user.email,
        'organization_id': current_user.organization_id
    }), 200
```

**Step 3: Commit**

```bash
git add app/auth/
git commit -m "feat: add authentication endpoints"
```

---

## Task 4: File Management Blueprint

**Files:**
- Create: `app/files/__init__.py`
- Create: `app/files/routes.py`

**Step 1: Create app/files/__init__.py**

```python
from flask import Blueprint

files_bp = Blueprint('files', __name__)

from app.files import routes
```

**Step 2: Create app/files/routes.py**

```python
import os
import uuid
from werkzeug.utils import secure_filename
from flask import request, jsonify, send_file, current_app
from flask_login import login_required, current_user
from app import db
from app.files import files_bp
from app.models import File

@files_bp.route('/upload', methods=['POST'])
@login_required
def upload_file():
    if 'file' not in request.files:
        return jsonify({'error': 'No file provided'}), 400

    file = request.files['file']

    if file.filename == '':
        return jsonify({'error': 'No file selected'}), 400

    # Check file size
    file.seek(0, os.SEEK_END)
    file_size = file.tell()
    file.seek(0)

    file_size_mb = file_size / (1024 * 1024)

    if file_size > current_app.config['MAX_CONTENT_LENGTH']:
        return jsonify({'error': f'File too large. Max size: {current_app.config["MAX_FILE_SIZE_MB"]}MB'}), 413

    # Check organization quota
    if not current_user.organization.has_storage_quota(file_size_mb):
        return jsonify({'error': 'Organization storage quota exceeded'}), 413

    # Generate file ID and secure filename
    file_id = str(uuid.uuid4())
    filename = secure_filename(file.filename)

    # Create directory structure: uploads/{org_id}/{file_id}/
    upload_dir = os.path.join(
        current_app.config['UPLOAD_FOLDER'],
        str(current_user.organization_id),
        file_id
    )
    os.makedirs(upload_dir, exist_ok=True)

    file_path = os.path.join(upload_dir, filename)

    # Save file
    file.save(file_path)

    # Create database record
    file_record = File(
        id=file_id,
        filename=filename,
        file_path=file_path,
        file_size_bytes=file_size,
        mime_type=file.content_type,
        status='pending',
        organization_id=current_user.organization_id,
        uploaded_by=current_user.id
    )
    db.session.add(file_record)
    db.session.commit()

    # Queue Celery task
    from app.tasks import validate_file
    validate_file.delay(file_id)

    return jsonify({
        'file_id': file_id,
        'filename': filename,
        'status': 'pending'
    }), 201

@files_bp.route('', methods=['GET'])
@login_required
def list_files():
    page = request.args.get('page', 1, type=int)
    limit = request.args.get('limit', 20, type=int)

    # Limit pagination
    limit = min(limit, 100)

    # Query files for current organization
    query = File.query.filter_by(organization_id=current_user.organization_id)
    total = query.count()

    files = query.order_by(File.uploaded_at.desc()).paginate(
        page=page, per_page=limit, error_out=False
    )

    return jsonify({
        'files': [f.to_dict() for f in files.items],
        'total': total,
        'page': page,
        'pages': files.pages
    }), 200

@files_bp.route('/<file_id>', methods=['GET'])
@login_required
def get_file(file_id):
    file = File.query.filter_by(
        id=file_id,
        organization_id=current_user.organization_id
    ).first()

    if not file:
        return jsonify({'error': 'File not found'}), 404

    return jsonify(file.to_dict()), 200

@files_bp.route('/<file_id>/download', methods=['GET'])
@login_required
def download_file(file_id):
    file = File.query.filter_by(
        id=file_id,
        organization_id=current_user.organization_id
    ).first()

    if not file:
        return jsonify({'error': 'File not found'}), 404

    if file.status != 'ready':
        return jsonify({'error': f'File is not ready for download. Status: {file.status}'}), 400

    if not os.path.exists(file.file_path):
        return jsonify({'error': 'File not found on disk'}), 404

    return send_file(
        file.file_path,
        as_attachment=True,
        download_name=file.filename,
        mimetype=file.mime_type
    )
```

**Step 3: Commit**

```bash
git add app/files/
git commit -m "feat: add file upload and management endpoints"
```

---

## Task 5: Celery Background Tasks

**Files:**
- Create: `app/tasks.py`

**Step 1: Create app/tasks.py**

```python
import os
import mimetypes
from datetime import datetime
from celery import Celery
from app import create_app, db
from app.models import File

# Create Flask app
flask_app = create_app(os.environ.get('FLASK_ENV', 'development'))

# Create Celery instance
celery = Celery(
    'tasks',
    broker=flask_app.config['CELERY_BROKER_URL'],
    backend=flask_app.config['CELERY_RESULT_BACKEND']
)

celery.conf.update(flask_app.config)

@celery.task(name='app.tasks.validate_file')
def validate_file(file_id):
    with flask_app.app_context():
        file = File.query.get(file_id)

        if not file:
            return {'error': 'File not found'}

        try:
            # Step 1: File integrity check
            if not os.path.exists(file.file_path):
                raise Exception('File does not exist on disk')

            if not os.path.isfile(file.file_path):
                raise Exception('Path is not a file')

            # Step 2: Size validation
            actual_size = os.path.getsize(file.file_path)
            if actual_size != file.file_size_bytes:
                raise Exception(f'File size mismatch. Expected: {file.file_size_bytes}, Actual: {actual_size}')

            # Step 3: MIME type verification
            guessed_type, _ = mimetypes.guess_type(file.file_path)
            if guessed_type and file.mime_type:
                if not guessed_type.startswith(file.mime_type.split('/')[0]):
                    raise Exception(f'MIME type mismatch. Uploaded: {file.mime_type}, Detected: {guessed_type}')

            # Step 4: Quota double-check
            file_size_mb = actual_size / (1024 * 1024)
            if not file.organization.has_storage_quota(file_size_mb):
                raise Exception('Organization storage quota exceeded')

            # Validation successful
            file.status = 'ready'
            file.processed_at = datetime.utcnow()
            db.session.commit()

            return {'status': 'ready', 'file_id': file_id}

        except Exception as e:
            # Validation failed
            file.status = 'failed'
            file.error_message = str(e)
            file.processed_at = datetime.utcnow()
            db.session.commit()

            # Optionally delete file
            if os.path.exists(file.file_path):
                try:
                    os.remove(file.file_path)
                except:
                    pass

            return {'status': 'failed', 'file_id': file_id, 'error': str(e)}
```

**Step 2: Commit**

```bash
git add app/tasks.py
git commit -m "feat: add Celery file validation task"
```

---

## Task 6: Docker Configuration

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `.dockerignore`

**Step 1: Create Dockerfile**

```dockerfile
FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Create uploads directory
RUN mkdir -p /app/uploads

EXPOSE 5000

CMD ["flask", "run", "--host=0.0.0.0"]
```

**Step 2: Create docker-compose.yml**

```yaml
version: '3.8'

services:
  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: flaskapp
      POSTGRES_USER: flask
      POSTGRES_PASSWORD: flask
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - flask_network

  redis:
    image: redis:7-alpine
    networks:
      - flask_network

  web:
    build: .
    ports:
      - "5000:5000"
    volumes:
      - ./app:/app/app
      - ./uploads:/app/uploads
    environment:
      FLASK_APP: wsgi:app
      FLASK_ENV: development
      DATABASE_URL: postgresql://flask:flask@db:5432/flaskapp
      REDIS_URL: redis://redis:6379/0
      SECRET_KEY: dev-secret-key-change-me
      UPLOAD_FOLDER: /app/uploads
    depends_on:
      - db
      - redis
    networks:
      - flask_network
    command: flask run --host=0.0.0.0

  worker:
    build: .
    volumes:
      - ./app:/app/app
      - ./uploads:/app/uploads
    environment:
      FLASK_ENV: development
      DATABASE_URL: postgresql://flask:flask@db:5432/flaskapp
      REDIS_URL: redis://redis:6379/0
      SECRET_KEY: dev-secret-key-change-me
      UPLOAD_FOLDER: /app/uploads
    depends_on:
      - db
      - redis
      - web
    networks:
      - flask_network
    command: celery -A app.tasks.celery worker --loglevel=info

volumes:
  postgres_data:

networks:
  flask_network:
    driver: bridge
```

**Step 3: Create .dockerignore**

```
.worktrees/
__pycache__/
*.pyc
*.pyo
*.pyd
.Python
env/
venv/
.venv/
*.egg-info/
.eggs/
dist/
build/
*.db
*.sqlite
*.sqlite3
.env
.git/
.gitignore
uploads/
migrations/
```

**Step 4: Commit**

```bash
git add Dockerfile docker-compose.yml .dockerignore
git commit -m "feat: add Docker configuration for development"
```

---

## Task 7: Application Entry Points

**Files:**
- Create: `wsgi.py`

**Step 1: Create wsgi.py**

```python
import os
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

from app import create_app, db
from app.models import User, Organization, File

app = create_app(os.environ.get('FLASK_ENV', 'development'))

@app.shell_context_processor
def make_shell_context():
    return {
        'db': db,
        'User': User,
        'Organization': Organization,
        'File': File
    }

if __name__ == '__main__':
    app.run()
```

**Step 2: Commit**

```bash
git add wsgi.py
git commit -m "feat: add WSGI entry point"
```

---

## Task 8: Database Initialization Script

**Files:**
- Create: `init_db.py`

**Step 1: Create init_db.py**

```python
import os
from dotenv import load_dotenv

load_dotenv()

from app import create_app, db

def init_database():
    app = create_app(os.environ.get('FLASK_ENV', 'development'))

    with app.app_context():
        # Create all tables
        db.create_all()
        print("Database tables created successfully!")

if __name__ == '__main__':
    init_database()
```

**Step 2: Commit**

```bash
git add init_db.py
git commit -m "feat: add database initialization script"
```

---

## Task 9: Test Suite Setup

**Files:**
- Create: `tests/__init__.py`
- Create: `tests/conftest.py`
- Create: `tests/test_auth.py`
- Create: `pytest.ini`

**Step 1: Add pytest to requirements.txt**

```bash
echo "pytest==7.4.3" >> requirements.txt
echo "pytest-flask==1.3.0" >> requirements.txt
```

**Step 2: Create tests/__init__.py**

```python
# Tests package
```

**Step 3: Create tests/conftest.py**

```python
import os
import pytest
from app import create_app, db
from app.models import User, Organization

@pytest.fixture
def app():
    app = create_app('development')
    app.config.update({
        'TESTING': True,
        'SQLALCHEMY_DATABASE_URI': 'sqlite:///:memory:',
        'WTF_CSRF_ENABLED': False
    })

    with app.app_context():
        db.create_all()
        yield app
        db.session.remove()
        db.drop_all()

@pytest.fixture
def client(app):
    return app.test_client()

@pytest.fixture
def runner(app):
    return app.test_cli_runner()

@pytest.fixture
def organization(app):
    with app.app_context():
        org = Organization(name='Test Org')
        db.session.add(org)
        db.session.commit()
        return org

@pytest.fixture
def user(app, organization):
    with app.app_context():
        user = User(email='test@example.com', organization_id=organization.id)
        user.set_password('password123')
        db.session.add(user)
        db.session.commit()
        return user
```

**Step 4: Create tests/test_auth.py**

```python
import pytest
from app.models import User, Organization

def test_register_new_user(client):
    response = client.post('/api/auth/register', json={
        'email': 'newuser@example.com',
        'password': 'password123',
        'organization_name': 'New Org'
    })

    assert response.status_code == 201
    data = response.get_json()
    assert data['email'] == 'newuser@example.com'
    assert 'user_id' in data
    assert 'organization_id' in data

def test_register_duplicate_email(client, user):
    response = client.post('/api/auth/register', json={
        'email': 'test@example.com',
        'password': 'password123',
        'organization_name': 'Test Org'
    })

    assert response.status_code == 400
    assert 'already registered' in response.get_json()['error']

def test_login_success(client, user):
    response = client.post('/api/auth/login', json={
        'email': 'test@example.com',
        'password': 'password123'
    })

    assert response.status_code == 200
    data = response.get_json()
    assert data['email'] == 'test@example.com'

def test_login_invalid_password(client, user):
    response = client.post('/api/auth/login', json={
        'email': 'test@example.com',
        'password': 'wrongpassword'
    })

    assert response.status_code == 401
    assert 'Invalid' in response.get_json()['error']

def test_logout(client, user):
    # Login first
    client.post('/api/auth/login', json={
        'email': 'test@example.com',
        'password': 'password123'
    })

    # Logout
    response = client.post('/api/auth/logout')
    assert response.status_code == 200
    assert response.get_json()['success'] is True
```

**Step 5: Create pytest.ini**

```ini
[pytest]
testpaths = tests
python_files = test_*.py
python_classes = Test*
python_functions = test_*
```

**Step 6: Run tests**

```bash
pytest tests/test_auth.py -v
```

Expected: All tests pass

**Step 7: Commit**

```bash
git add tests/ pytest.ini requirements.txt
git commit -m "test: add authentication test suite"
```

---

## Task 10: File Management Tests

**Files:**
- Create: `tests/test_files.py`

**Step 1: Create tests/test_files.py**

```python
import io
import pytest
from app.models import File

@pytest.fixture
def authenticated_client(client, user):
    client.post('/api/auth/login', json={
        'email': 'test@example.com',
        'password': 'password123'
    })
    return client

def test_upload_file(authenticated_client):
    data = {
        'file': (io.BytesIO(b"test file content"), 'test.txt')
    }

    response = authenticated_client.post(
        '/api/files/upload',
        data=data,
        content_type='multipart/form-data'
    )

    assert response.status_code == 201
    data = response.get_json()
    assert 'file_id' in data
    assert data['filename'] == 'test.txt'
    assert data['status'] == 'pending'

def test_upload_no_file(authenticated_client):
    response = authenticated_client.post('/api/files/upload')

    assert response.status_code == 400
    assert 'No file provided' in response.get_json()['error']

def test_list_files(authenticated_client, app):
    # Upload a file first
    with app.app_context():
        data = {
            'file': (io.BytesIO(b"test content"), 'test.txt')
        }
        authenticated_client.post(
            '/api/files/upload',
            data=data,
            content_type='multipart/form-data'
        )

    # List files
    response = authenticated_client.get('/api/files')

    assert response.status_code == 200
    data = response.get_json()
    assert 'files' in data
    assert len(data['files']) > 0

def test_get_file_details(authenticated_client, app):
    # Upload a file
    with app.app_context():
        data = {
            'file': (io.BytesIO(b"test content"), 'test.txt')
        }
        upload_response = authenticated_client.post(
            '/api/files/upload',
            data=data,
            content_type='multipart/form-data'
        )
        file_id = upload_response.get_json()['file_id']

    # Get file details
    response = authenticated_client.get(f'/api/files/{file_id}')

    assert response.status_code == 200
    data = response.get_json()
    assert data['id'] == file_id
    assert data['filename'] == 'test.txt'

def test_get_nonexistent_file(authenticated_client):
    response = authenticated_client.get('/api/files/nonexistent-id')

    assert response.status_code == 404

def test_tenant_isolation(client, app, organization):
    # Create two users in different organizations
    with app.app_context():
        from app.models import Organization, User

        org2 = Organization(name='Org 2')
        app.db.session.add(org2)
        app.db.session.commit()

        user2 = User(email='user2@example.com', organization_id=org2.id)
        user2.set_password('password123')
        app.db.session.add(user2)
        app.db.session.commit()

    # User 1 uploads file
    client.post('/api/auth/login', json={
        'email': 'test@example.com',
        'password': 'password123'
    })

    data = {
        'file': (io.BytesIO(b"user1 content"), 'user1.txt')
    }
    upload_response = client.post(
        '/api/files/upload',
        data=data,
        content_type='multipart/form-data'
    )
    file_id = upload_response.get_json()['file_id']

    # Logout and login as user 2
    client.post('/api/auth/logout')
    client.post('/api/auth/login', json={
        'email': 'user2@example.com',
        'password': 'password123'
    })

    # User 2 tries to access user 1's file
    response = client.get(f'/api/files/{file_id}')

    assert response.status_code == 404
```

**Step 2: Run tests**

```bash
pytest tests/test_files.py -v
```

Expected: All tests pass

**Step 3: Commit**

```bash
git add tests/test_files.py
git commit -m "test: add file management test suite"
```

---

## Task 11: Documentation

**Files:**
- Create: `README.md`

**Step 1: Create README.md**

```markdown
# Flask Multi-Tenant SaaS API

A production-ready Flask REST API with multi-tenant support, file uploads, and background job processing.

## Features

- Session-based authentication with Flask-Login
- Multi-tenant architecture with organization-based isolation
- File upload with background validation via Celery
- PostgreSQL database with SQLAlchemy ORM
- Redis for session storage and job queue
- Docker Compose for local development
- Comprehensive test suite

## Tech Stack

- **Backend:** Flask 3.0, SQLAlchemy, Flask-Login
- **Database:** PostgreSQL (production), SQLite (development)
- **Cache/Queue:** Redis
- **Background Jobs:** Celery
- **Containerization:** Docker, Docker Compose

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Python 3.11+ (for local development without Docker)

### Running with Docker

1. Clone the repository:
```bash
git clone <repo-url>
cd test2-flask-api
```

2. Start all services:
```bash
docker-compose up --build
```

3. Initialize the database (first time only):
```bash
docker-compose exec web python init_db.py
```

4. Access the API at `http://localhost:5000`

### Running Tests

```bash
pytest tests/ -v
```

## API Endpoints

### Authentication

**POST /api/auth/register**
```json
{
  "email": "user@example.com",
  "password": "password123",
  "organization_name": "My Company"
}
```

**POST /api/auth/login**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**POST /api/auth/logout**
```json
{}
```

### Files (Authentication Required)

**POST /api/files/upload**
- Content-Type: multipart/form-data
- Body: file field with file data

**GET /api/files**
- Query params: `?page=1&limit=20`

**GET /api/files/{file_id}**
- Returns file metadata

**GET /api/files/{file_id}/download**
- Downloads the file

## Architecture

### Multi-Tenant Isolation

All data is scoped by `organization_id`. Users can only access files belonging to their organization.

### Background Processing

File uploads trigger Celery tasks that:
1. Verify file integrity
2. Validate file size
3. Check MIME type
4. Confirm storage quota
5. Update file status to 'ready' or 'failed'

### Security

- Passwords hashed with bcrypt
- Session cookies: httponly, secure (HTTPS in production), SameSite=Lax
- CSRF protection on state-changing requests
- SQL injection protection via SQLAlchemy ORM
- File size limits and storage quotas enforced

## Configuration

Environment variables (see `.env.example`):

- `FLASK_ENV` - development or production
- `SECRET_KEY` - Flask secret key
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `UPLOAD_FOLDER` - File storage directory
- `MAX_FILE_SIZE_MB` - Maximum file size (default: 100)
- `STORAGE_QUOTA_MB` - Per-organization storage quota (default: 1000)

## Development

### Project Structure

```
.
├── app/
│   ├── __init__.py       # Flask app factory
│   ├── models.py         # Database models
│   ├── tasks.py          # Celery tasks
│   ├── auth/             # Authentication blueprint
│   └── files/            # File management blueprint
├── tests/                # Test suite
├── config.py             # Configuration classes
├── wsgi.py               # WSGI entry point
├── docker-compose.yml    # Docker orchestration
└── requirements.txt      # Python dependencies
```

### Running Locally (without Docker)

1. Install dependencies:
```bash
pip install -r requirements.txt
```

2. Set environment variables:
```bash
cp .env.example .env
# Edit .env with your settings
```

3. Initialize database:
```bash
python init_db.py
```

4. Start Flask:
```bash
flask run
```

5. Start Celery worker (separate terminal):
```bash
celery -A app.tasks.celery worker --loglevel=info
```

## License

MIT
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add comprehensive README"
```

---

## Task 12: Final Integration Test

**Files:**
- Modify: `docker-compose.yml` (add healthchecks)

**Step 1: Run full Docker stack**

```bash
docker-compose up --build
```

Expected: All services start successfully

**Step 2: Initialize database**

```bash
docker-compose exec web python init_db.py
```

Expected: "Database tables created successfully!"

**Step 3: Test registration endpoint**

```bash
curl -X POST http://localhost:5000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","organization_name":"Test Org"}'
```

Expected: 201 response with user data

**Step 4: Test login endpoint**

```bash
curl -X POST http://localhost:5000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' \
  -c cookies.txt
```

Expected: 200 response with user data and session cookie

**Step 5: Test file upload**

```bash
echo "Test file content" > test.txt
curl -X POST http://localhost:5000/api/files/upload \
  -b cookies.txt \
  -F "file=@test.txt"
```

Expected: 201 response with file_id

**Step 6: Verify Celery processed the file**

```bash
docker-compose logs worker
```

Expected: Log showing file validation task completed

**Step 7: Test file listing**

```bash
curl -X GET http://localhost:5000/api/files \
  -b cookies.txt
```

Expected: 200 response with file list including uploaded file

**Step 8: Document verification**

Create a test verification checklist in `docs/verification.md`:

```markdown
# Verification Checklist

## Functional Tests
- [x] User registration creates organization and user
- [x] User login returns session cookie
- [x] File upload saves file and creates database record
- [x] Celery worker processes file validation task
- [x] File listing returns files for user's organization
- [x] File download works for ready files
- [x] Tenant isolation prevents cross-organization access

## Integration Tests
- [x] All Docker services start
- [x] Database migrations work
- [x] Redis connection successful
- [x] Celery worker connects to broker
- [x] Flask app serves requests

## Security Tests
- [x] Passwords are hashed
- [x] Session cookies are httponly
- [x] CSRF protection enabled
- [x] File size limits enforced
- [x] Storage quotas enforced
- [x] Tenant isolation verified
```

**Step 9: Commit**

```bash
git add docs/verification.md
git commit -m "docs: add verification checklist"
```

---

## Completion Criteria

- All endpoints respond correctly
- Tests pass (authentication and file management)
- Docker Compose stack runs successfully
- Celery workers process background tasks
- Database schema created and migrations work
- README documents setup and usage
- Security features implemented (password hashing, CSRF, tenant isolation)
- File uploads and downloads work end-to-end

## Next Steps (Out of Scope)

- Add Flask-Migrate migrations instead of direct db.create_all()
- Add rate limiting with Flask-Limiter
- Add logging with structured logs
- Add monitoring and metrics (Prometheus/Grafana)
- Add API documentation (Swagger/OpenAPI)
- Add user roles and permissions within organizations
- Add file sharing between users in same organization
- Add S3 storage backend option
- Add file versioning
- Add webhook notifications on file processing completion
