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
