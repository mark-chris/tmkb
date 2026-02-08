"""Pytest configuration and fixtures."""

import os
import pytest
import tempfile
from app import create_app
from app.extensions import db as _db
from app.models.organization import Organization
from app.models.user import User
from app.models.file import File


class TestConfig:
    """Test configuration."""
    TESTING = True
    SECRET_KEY = 'test-secret-key'
    SQLALCHEMY_DATABASE_URI = 'sqlite:///:memory:'
    SQLALCHEMY_TRACK_MODIFICATIONS = False
    WTF_CSRF_ENABLED = False

    # Use temporary directory for test file uploads
    UPLOAD_FOLDER = tempfile.mkdtemp()
    MAX_CONTENT_LENGTH = 10 * 1024 * 1024

    # Disable Celery for tests
    CELERY_TASK_ALWAYS_EAGER = True
    CELERY_TASK_EAGER_PROPAGATES = True


@pytest.fixture(scope='function')
def app():
    """Create and configure Flask app for testing."""
    app = create_app(TestConfig)

    with app.app_context():
        _db.create_all()
        yield app
        _db.session.remove()
        _db.drop_all()


@pytest.fixture(scope='function')
def db(app):
    """Provide database for tests."""
    return _db


@pytest.fixture(scope='function')
def client(app):
    """Provide test client."""
    return app.test_client()


@pytest.fixture(scope='function')
def org1(db):
    """Create test organization 1."""
    org = Organization(name='Organization 1')
    db.session.add(org)
    db.session.commit()
    return org


@pytest.fixture(scope='function')
def org2(db):
    """Create test organization 2."""
    org = Organization(name='Organization 2')
    db.session.add(org)
    db.session.commit()
    return org


@pytest.fixture(scope='function')
def user1(db, org1):
    """Create test user in organization 1."""
    user = User(
        email='user1@org1.com',
        organization_id=org1.id,
        is_active=True
    )
    user.set_password('password123')
    db.session.add(user)
    db.session.commit()
    return user


@pytest.fixture(scope='function')
def user2(db, org2):
    """Create test user in organization 2."""
    user = User(
        email='user2@org2.com',
        organization_id=org2.id,
        is_active=True
    )
    user.set_password('password123')
    db.session.add(user)
    db.session.commit()
    return user


@pytest.fixture(scope='function')
def authenticated_client1(client, user1):
    """Provide authenticated client for user1."""
    client.post('/auth/login', json={
        'email': 'user1@org1.com',
        'password': 'password123'
    })
    return client


@pytest.fixture(scope='function')
def authenticated_client2(client, user2):
    """Provide authenticated client for user2."""
    client.post('/auth/login', json={
        'email': 'user2@org2.com',
        'password': 'password123'
    })
    return client
