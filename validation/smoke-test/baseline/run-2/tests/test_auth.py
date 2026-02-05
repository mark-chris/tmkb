"""
Tests for authentication endpoints.
"""
import pytest
from app import db
from app.models import User, Organization


class TestRegister:
    """Tests for the /api/auth/register endpoint."""

    def test_register_new_user_new_organization(self, client):
        """Test registering a new user with a new organization."""
        response = client.post('/api/auth/register', json={
            'email': 'newuser@example.com',
            'password': 'securepassword123',
            'organization_name': 'New Company'
        })

        assert response.status_code == 201
        data = response.get_json()
        assert 'user_id' in data
        assert data['email'] == 'newuser@example.com'
        assert 'organization_id' in data

        # Verify user was created in database
        with client.application.app_context():
            user = User.query.filter_by(email='newuser@example.com').first()
            assert user is not None
            assert user.check_password('securepassword123')

            # Verify organization was created
            org = Organization.query.filter_by(name='New Company').first()
            assert org is not None
            assert user.organization_id == org.id

    def test_register_new_user_existing_organization(self, client, organization):
        """Test registering a new user to an existing organization."""
        response = client.post('/api/auth/register', json={
            'email': 'newmember@example.com',
            'password': 'password123',
            'organization_name': organization['name']
        })

        assert response.status_code == 201
        data = response.get_json()
        assert data['organization_id'] == organization['id']

        # Verify no duplicate organization was created
        with client.application.app_context():
            org_count = Organization.query.filter_by(name=organization['name']).count()
            assert org_count == 1

    def test_register_duplicate_email(self, client, user):
        """Test that registering with an existing email fails."""
        response = client.post('/api/auth/register', json={
            'email': user['email'],
            'password': 'somepassword',
            'organization_name': 'Any Organization'
        })

        assert response.status_code == 400
        data = response.get_json()
        assert 'error' in data
        assert 'already exists' in data['error'].lower()

    def test_register_missing_fields(self, client):
        """Test that registration fails with missing required fields."""
        # Missing password
        response = client.post('/api/auth/register', json={
            'email': 'test@example.com',
            'organization_name': 'Test Org'
        })
        assert response.status_code == 400

        # Missing email
        response = client.post('/api/auth/register', json={
            'password': 'password123',
            'organization_name': 'Test Org'
        })
        assert response.status_code == 400

        # Missing organization_name
        response = client.post('/api/auth/register', json={
            'email': 'test@example.com',
            'password': 'password123'
        })
        assert response.status_code == 400

    def test_register_invalid_email(self, client):
        """Test that registration fails with invalid email format."""
        response = client.post('/api/auth/register', json={
            'email': 'not-an-email',
            'password': 'password123',
            'organization_name': 'Test Org'
        })

        assert response.status_code == 400
        data = response.get_json()
        assert 'error' in data


class TestLogin:
    """Tests for the /api/auth/login endpoint."""

    def test_login_success(self, client, user):
        """Test successful login with valid credentials."""
        response = client.post('/api/auth/login', json={
            'email': user['email'],
            'password': user['password']
        })

        assert response.status_code == 200
        data = response.get_json()
        assert data['user_id'] == user['id']
        assert data['email'] == user['email']
        assert data['organization_id'] == user['organization_id']

        # Verify session cookie was set
        assert 'session' in [cookie.name for cookie in client.cookie_jar]

    def test_login_wrong_password(self, client, user):
        """Test login fails with incorrect password."""
        response = client.post('/api/auth/login', json={
            'email': user['email'],
            'password': 'wrongpassword'
        })

        assert response.status_code == 401
        data = response.get_json()
        assert 'error' in data
        assert 'invalid' in data['error'].lower()

    def test_login_nonexistent_user(self, client):
        """Test login fails for non-existent user."""
        response = client.post('/api/auth/login', json={
            'email': 'nonexistent@example.com',
            'password': 'password123'
        })

        assert response.status_code == 401
        data = response.get_json()
        assert 'error' in data

    def test_login_inactive_user(self, client, user):
        """Test login fails for inactive user."""
        # Deactivate the user
        with client.application.app_context():
            u = User.query.get(user['id'])
            u.is_active = False
            db.session.commit()

        response = client.post('/api/auth/login', json={
            'email': user['email'],
            'password': user['password']
        })

        assert response.status_code == 401
        data = response.get_json()
        assert 'error' in data
        assert 'inactive' in data['error'].lower() or 'deactivated' in data['error'].lower()

    def test_login_missing_fields(self, client):
        """Test login fails with missing required fields."""
        # Missing password
        response = client.post('/api/auth/login', json={
            'email': 'test@example.com'
        })
        assert response.status_code == 400

        # Missing email
        response = client.post('/api/auth/login', json={
            'password': 'password123'
        })
        assert response.status_code == 400


class TestLogout:
    """Tests for the /api/auth/logout endpoint."""

    def test_logout_success(self, authenticated_client):
        """Test successful logout."""
        response = authenticated_client.post('/api/auth/logout')

        assert response.status_code == 200
        data = response.get_json()
        assert data['success'] is True

        # Verify session was cleared - subsequent authenticated request should fail
        response = authenticated_client.get('/api/files')
        assert response.status_code == 401

    def test_logout_not_authenticated(self, client):
        """Test logout when not authenticated."""
        response = client.post('/api/auth/logout')

        # Should either succeed (idempotent) or return 401
        assert response.status_code in [200, 401]


class TestAuthenticationRequired:
    """Tests for authentication requirements on protected endpoints."""

    def test_files_endpoint_requires_auth(self, client):
        """Test that file endpoints require authentication."""
        response = client.get('/api/files')
        assert response.status_code == 401

    def test_file_upload_requires_auth(self, client):
        """Test that file upload requires authentication."""
        response = client.post('/api/files/upload')
        assert response.status_code == 401

    def test_authenticated_request_includes_user(self, authenticated_client, user):
        """Test that authenticated requests have access to current user."""
        response = authenticated_client.get('/api/files')

        # Should succeed (even if empty list)
        assert response.status_code == 200
