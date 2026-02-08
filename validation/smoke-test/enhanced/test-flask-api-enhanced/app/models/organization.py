"""Organization model for multi-tenant support."""

from datetime import datetime
from app.extensions import db


class Organization(db.Model):
    """
    Organization model representing tenants in multi-tenant architecture.

    Each organization is completely isolated - users belong to one organization
    and can only access resources owned by their organization.
    """

    __tablename__ = 'organizations'

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(255), nullable=False, unique=True, index=True)
    created_at = db.Column(db.DateTime, nullable=False, default=datetime.utcnow)

    # Relationships
    users = db.relationship('User', back_populates='organization', lazy='dynamic', cascade='all, delete-orphan')
    files = db.relationship('File', back_populates='organization', lazy='dynamic', cascade='all, delete-orphan')

    def __repr__(self):
        return f'<Organization {self.name}>'

    def to_dict(self):
        """Convert to dictionary representation."""
        return {
            'id': self.id,
            'name': self.name,
            'created_at': self.created_at.isoformat()
        }
