"""
Pytest configuration and fixtures for testing.
"""
import os
import tempfile
import pytest
from app import create_app, db
from app.models import User, Organization, File


@pytest.fixture
def app():
    """Create and configure a test Flask application instance."""
    # Create a temporary file for the test database
    db_fd, db_path = tempfile.mkstemp()

    # Create a temporary directory for test file uploads
    upload_dir = tempfile.mkdtemp()

    # Test configuration
    test_config = {
        'TESTING': True,
        'SQLALCHEMY_DATABASE_URI': f'sqlite:///{db_path}',
        'WTF_CSRF_ENABLED': False,  # Disable CSRF for testing
        'SECRET_KEY': 'test-secret-key',
        'UPLOAD_FOLDER': upload_dir,
        'MAX_FILE_SIZE_MB': 10,
        'MAX_CONTENT_LENGTH': 10 * 1024 * 1024,  # 10MB in bytes
        'STORAGE_QUOTA_MB': 100,
        'CELERY_TASK_ALWAYS_EAGER': True,  # Execute tasks synchronously in tests
        'CELERY_TASK_EAGER_PROPAGATES': True,
        'SESSION_TYPE': 'filesystem',  # Use filesystem sessions for testing instead of Redis
        'SESSION_PERMANENT': False,
    }

    # Create app with test config
    app = create_app(test_config)

    # Create database tables
    with app.app_context():
        db.create_all()

    yield app

    # Cleanup
    with app.app_context():
        db.session.remove()
        db.drop_all()

    os.close(db_fd)
    os.unlink(db_path)

    # Clean up upload directory
    import shutil
    if os.path.exists(upload_dir):
        shutil.rmtree(upload_dir)


@pytest.fixture
def client(app):
    """Create a test client for the Flask application."""
    return app.test_client()


@pytest.fixture
def runner(app):
    """Create a test CLI runner for the Flask application."""
    return app.test_cli_runner()


@pytest.fixture
def organization(app):
    """Create a test organization."""
    with app.app_context():
        org = Organization(name='Test Organization', storage_quota_mb=100)
        db.session.add(org)
        db.session.commit()
        # Refresh to get the ID
        db.session.refresh(org)
        org_id = org.id
        org_name = org.name

    # Return a dict with the data (detached from session)
    return {'id': org_id, 'name': org_name}


@pytest.fixture
def user(app, organization):
    """Create a test user."""
    with app.app_context():
        user = User(
            email='test@example.com',
            organization_id=organization['id']
        )
        user.set_password('testpassword123')
        db.session.add(user)
        db.session.commit()
        # Refresh to get the ID
        db.session.refresh(user)
        user_id = user.id
        user_email = user.email

    # Return a dict with the data (detached from session)
    return {
        'id': user_id,
        'email': user_email,
        'password': 'testpassword123',
        'organization_id': organization['id']
    }


@pytest.fixture
def second_organization(app):
    """Create a second test organization for multi-tenant testing."""
    with app.app_context():
        org = Organization(name='Second Organization', storage_quota_mb=100)
        db.session.add(org)
        db.session.commit()
        db.session.refresh(org)
        org_id = org.id
        org_name = org.name

    return {'id': org_id, 'name': org_name}


@pytest.fixture
def second_user(app, second_organization):
    """Create a user in a different organization for tenant isolation testing."""
    with app.app_context():
        user = User(
            email='other@example.com',
            organization_id=second_organization['id']
        )
        user.set_password('otherpassword123')
        db.session.add(user)
        db.session.commit()
        db.session.refresh(user)
        user_id = user.id
        user_email = user.email

    return {
        'id': user_id,
        'email': user_email,
        'password': 'otherpassword123',
        'organization_id': second_organization['id']
    }


@pytest.fixture
def authenticated_client(client, user):
    """Create an authenticated test client."""
    # Log in the user
    client.post('/api/auth/login', json={
        'email': user['email'],
        'password': user['password']
    })
    return client


@pytest.fixture
def second_authenticated_client(client, second_user):
    """Create an authenticated test client for the second user."""
    # Log in the second user
    client.post('/api/auth/login', json={
        'email': second_user['email'],
        'password': second_user['password']
    })
    return client
