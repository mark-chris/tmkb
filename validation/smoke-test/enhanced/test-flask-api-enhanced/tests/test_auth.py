"""Authentication tests."""

import pytest
from app.models.user import User


def test_login_success(client, user1):
    """Test successful login."""
    response = client.post('/auth/login', json={
        'email': 'user1@org1.com',
        'password': 'password123'
    })

    assert response.status_code == 200
    data = response.get_json()
    assert data['message'] == 'Login successful'
    assert data['user']['email'] == 'user1@org1.com'
    assert data['user']['organization_id'] == user1.organization_id


def test_login_invalid_credentials(client, user1):
    """Test login with invalid password."""
    response = client.post('/auth/login', json={
        'email': 'user1@org1.com',
        'password': 'wrongpassword'
    })

    assert response.status_code == 401
    data = response.get_json()
    assert 'error' in data


def test_login_missing_fields(client):
    """Test login with missing fields."""
    response = client.post('/auth/login', json={
        'email': 'user1@org1.com'
    })

    assert response.status_code == 400
    data = response.get_json()
    assert 'error' in data


def test_login_inactive_user(client, user1, db):
    """Test login with deactivated account."""
    user1.is_active = False
    db.session.commit()

    response = client.post('/auth/login', json={
        'email': 'user1@org1.com',
        'password': 'password123'
    })

    assert response.status_code == 403
    data = response.get_json()
    assert 'disabled' in data['error'].lower()


def test_logout(authenticated_client1):
    """Test logout."""
    response = authenticated_client1.post('/auth/logout')

    assert response.status_code == 200
    data = response.get_json()
    assert data['message'] == 'Logout successful'


def test_get_current_user(authenticated_client1, user1):
    """Test getting current user info."""
    response = authenticated_client1.get('/auth/me')

    assert response.status_code == 200
    data = response.get_json()
    assert data['email'] == 'user1@org1.com'
    assert data['organization_id'] == user1.organization_id


def test_get_current_user_unauthorized(client):
    """Test getting current user without authentication."""
    response = client.get('/auth/me')
    assert response.status_code == 401
