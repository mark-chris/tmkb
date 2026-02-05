import re
from flask import request, jsonify
from flask_login import login_user, logout_user, login_required, current_user
from sqlalchemy.exc import IntegrityError
from app import db
from app.auth import auth_bp
from app.models import User, Organization


def validate_email(email):
    """Basic email format validation"""
    pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    return re.match(pattern, email) is not None


def validate_password(password):
    """Password must be at least 8 characters"""
    return len(password) >= 8

@auth_bp.route('/register', methods=['POST'])
def register():
    data = request.get_json()

    if not data or not data.get('email') or not data.get('password') or not data.get('organization_name'):
        return jsonify({'error': 'Missing required fields'}), 400

    # Validate email format
    if not validate_email(data['email']):
        return jsonify({'error': 'Invalid email format'}), 400

    # Validate password strength
    if not validate_password(data['password']):
        return jsonify({'error': 'Password must be at least 8 characters'}), 400

    # Check if user already exists
    if User.query.filter_by(email=data['email']).first():
        return jsonify({'error': 'Email already registered'}), 400

    # Get or create organization with race condition handling
    org = Organization.query.filter_by(name=data['organization_name']).first()
    if not org:
        org = Organization(name=data['organization_name'])
        db.session.add(org)
        try:
            db.session.flush()
        except IntegrityError:
            # Another request created the org in parallel
            db.session.rollback()
            org = Organization.query.filter_by(name=data['organization_name']).first()

    # Create user
    user = User(
        email=data['email'],
        organization_id=org.id
    )
    user.set_password(data['password'])
    db.session.add(user)

    try:
        db.session.commit()
    except IntegrityError:
        db.session.rollback()
        return jsonify({'error': 'Failed to create user'}), 500

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

    # Validate email format
    if not validate_email(data['email']):
        return jsonify({'error': 'Invalid email format'}), 400

    # Validate password strength
    if not validate_password(data['password']):
        return jsonify({'error': 'Password must be at least 8 characters'}), 400

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
