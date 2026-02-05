import uuid
from datetime import datetime
from flask_login import UserMixin
from werkzeug.security import generate_password_hash, check_password_hash

# Import db - this is safe because app/__init__.py creates db before importing models
from app import db

class Organization(db.Model):
    __tablename__ = 'organizations'

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(255), unique=True, nullable=False, index=True)
    storage_quota_mb = db.Column(db.Integer, default=1000)
    created_at = db.Column(db.DateTime, default=datetime.utcnow)

    users = db.relationship('User', backref='organization', lazy=True, cascade='all, delete-orphan')
    files = db.relationship('File', backref='organization', lazy=True, cascade='all, delete-orphan')

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
    organization_id = db.Column(db.Integer, db.ForeignKey('organizations.id'), nullable=False, index=True)
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
    file_size_bytes = db.Column(db.BigInteger)
    mime_type = db.Column(db.String(127))
    status = db.Column(db.Enum('pending', 'ready', 'failed', name='file_status'), nullable=False, default='pending')
    error_message = db.Column(db.Text)
    organization_id = db.Column(db.Integer, db.ForeignKey('organizations.id'), nullable=False, index=True)
    uploaded_by = db.Column(db.Integer, db.ForeignKey('users.id'), nullable=False, index=True)
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
