"""File model with tenant isolation."""

from datetime import datetime
from app.extensions import db
from app.models.base import TenantScopedMixin


class File(db.Model, TenantScopedMixin):
    """
    File model representing uploaded files with multi-tenant isolation.

    Security (TMKB-AUTHZ-005):
    - ORG-OWNS-RESOURCE: organization_id (inherited from TenantScopedMixin)
    - USER-OWNS-RESOURCE: uploaded_by_user_id
    - Tenant isolation enforced automatically via TenantScopedMixin
    """

    __tablename__ = 'files'

    id = db.Column(db.Integer, primary_key=True)
    filename = db.Column(db.String(255), nullable=False)  # UUID-based unique filename
    original_filename = db.Column(db.String(255), nullable=False)  # User's original filename
    filepath = db.Column(db.String(512), nullable=False)  # Relative to storage root
    mimetype = db.Column(db.String(127), nullable=False)
    size_bytes = db.Column(db.Integer, nullable=False)

    # ORG-OWNS-RESOURCE relationship (inherited from TenantScopedMixin)
    # organization_id is defined in TenantScopedMixin

    # USER-OWNS-RESOURCE relationship (TMKB-AUTHZ-005)
    uploaded_by_user_id = db.Column(
        db.Integer,
        db.ForeignKey('users.id'),
        nullable=False,
        index=True
    )

    # Processing status
    status = db.Column(
        db.String(50),
        nullable=False,
        default='pending',
        index=True
    )  # pending, processing, completed, failed

    processed_at = db.Column(db.DateTime, nullable=True)
    metadata = db.Column(db.JSON, nullable=True)  # Extracted metadata from processing

    created_at = db.Column(db.DateTime, nullable=False, default=datetime.utcnow)

    # Soft delete support
    deleted_at = db.Column(db.DateTime, nullable=True, index=True)

    # Relationships
    organization = db.relationship('Organization', back_populates='files')
    uploaded_by_user = db.relationship('User', back_populates='files')

    def __repr__(self):
        return f'<File {self.original_filename}>'

    def to_dict(self):
        """Convert to dictionary representation."""
        return {
            'id': self.id,
            'filename': self.original_filename,
            'size': self.size_bytes,
            'mimetype': self.mimetype,
            'status': self.status,
            'uploaded_by': self.uploaded_by_user_id,
            'created_at': self.created_at.isoformat(),
            'processed_at': self.processed_at.isoformat() if self.processed_at else None,
            'metadata': self.metadata
        }
