"""User model with Flask-Login integration."""

from datetime import datetime
from flask_login import UserMixin
from werkzeug.security import generate_password_hash, check_password_hash
from app.extensions import db


class User(db.Model, UserMixin):
    """
    User model representing application users.

    Security (TMKB-AUTHZ-005):
    - USER-IN-ORG relationship: Each user belongs to exactly one organization
    - Users can only access resources from their organization
    - organization_id is set at user creation and should not change
    """

    __tablename__ = 'users'

    id = db.Column(db.Integer, primary_key=True)
    email = db.Column(db.String(255), nullable=False, unique=True, index=True)
    password_hash = db.Column(db.String(255), nullable=False)

    # USER-IN-ORG relationship (TMKB-AUTHZ-005)
    organization_id = db.Column(
        db.Integer,
        db.ForeignKey('organizations.id'),
        nullable=False,
        index=True
    )

    created_at = db.Column(db.DateTime, nullable=False, default=datetime.utcnow)
    is_active = db.Column(db.Boolean, nullable=False, default=True)

    # Relationships
    organization = db.relationship('Organization', back_populates='users')
    files = db.relationship('File', back_populates='uploaded_by_user', lazy='dynamic')

    def set_password(self, password):
        """Hash and store password."""
        self.password_hash = generate_password_hash(password)

    def check_password(self, password):
        """Verify password against hash."""
        return check_password_hash(self.password_hash, password)

    def __repr__(self):
        return f'<User {self.email}>'

    def to_dict(self):
        """Convert to dictionary representation."""
        return {
            'id': self.id,
            'email': self.email,
            'organization_id': self.organization_id,
            'is_active': self.is_active,
            'created_at': self.created_at.isoformat()
        }
