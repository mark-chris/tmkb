from app import db, login_manager
from datetime import datetime
import bcrypt
import uuid

class Organization(db.Model):
    """Organization model for multi-tenancy"""
    __tablename__ = 'organizations'

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(255), unique=True, nullable=False, index=True)
    created_at = db.Column(db.DateTime, default=datetime.utcnow, nullable=False)

    # Relationships
    users = db.relationship('User', backref='organization', lazy='dynamic', cascade='all, delete-orphan')
    files = db.relationship('File', backref='organization', lazy='dynamic', cascade='all, delete-orphan')

    def __repr__(self):
        return f'<Organization {self.name}>'

    def to_dict(self):
        return {
            'id': self.id,
            'name': self.name,
            'created_at': self.created_at.isoformat()
        }

class User(db.Model):
    """User model with bcrypt password hashing"""
    __tablename__ = 'users'

    id = db.Column(db.Integer, primary_key=True)
    username = db.Column(db.String(80), unique=True, nullable=False, index=True)
    password_hash = db.Column(db.String(128), nullable=False)
    organization_id = db.Column(db.Integer, db.ForeignKey('organizations.id'), nullable=False, index=True)
    created_at = db.Column(db.DateTime, default=datetime.utcnow, nullable=False)

    # Relationships
    files = db.relationship('File', backref='uploader', lazy='dynamic', foreign_keys='File.uploaded_by')

    def __repr__(self):
        return f'<User {self.username}>'

    def set_password(self, password):
        """Hash password using bcrypt"""
        self.password_hash = bcrypt.hashpw(password.encode('utf-8'), bcrypt.gensalt(12)).decode('utf-8')

    def check_password(self, password):
        """Verify password against hash"""
        return bcrypt.checkpw(password.encode('utf-8'), self.password_hash.encode('utf-8'))

    @property
    def is_active(self):
        """Required by Flask-Login"""
        return True

    @property
    def is_authenticated(self):
        """Required by Flask-Login"""
        return True

    @property
    def is_anonymous(self):
        """Required by Flask-Login"""
        return False

    def get_id(self):
        """Required by Flask-Login"""
        return str(self.id)

    def to_dict(self):
        return {
            'id': self.id,
            'username': self.username,
            'organization_id': self.organization_id
        }

@login_manager.user_loader
def load_user(user_id):
    """User loader for Flask-Login"""
    return User.query.get(int(user_id))

class File(db.Model):
    """File model for uploaded files"""
    __tablename__ = 'files'

    id = db.Column(db.String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    filename = db.Column(db.String(255), nullable=False)
    filepath = db.Column(db.String(512), nullable=True)
    organization_id = db.Column(db.Integer, db.ForeignKey('organizations.id'), nullable=False, index=True)
    uploaded_by = db.Column(db.Integer, db.ForeignKey('users.id'), nullable=False, index=True)
    status = db.Column(db.String(20), default='pending', nullable=False, index=True)
    file_size = db.Column(db.Integer, nullable=True)
    mime_type = db.Column(db.String(100), nullable=True)
    sha256_hash = db.Column(db.String(64), nullable=True)
    celery_task_id = db.Column(db.String(36), nullable=True)
    error_message = db.Column(db.Text, nullable=True)
    created_at = db.Column(db.DateTime, default=datetime.utcnow, nullable=False, index=True)
    processed_at = db.Column(db.DateTime, nullable=True)

    def __repr__(self):
        return f'<File {self.filename} ({self.status})>'

    def to_dict(self, include_uploader=False):
        """Serialize file to dictionary"""
        result = {
            'id': self.id,
            'filename': self.filename,
            'status': self.status,
            'file_size': self.file_size,
            'mime_type': self.mime_type,
            'sha256_hash': self.sha256_hash,
            'created_at': self.created_at.isoformat(),
            'processed_at': self.processed_at.isoformat() if self.processed_at else None
        }

        if include_uploader and self.uploader:
            result['uploaded_by'] = {
                'id': self.uploader.id,
                'username': self.uploader.username
            }

        if self.status == 'failed' and self.error_message:
            result['error_message'] = self.error_message

        return result
