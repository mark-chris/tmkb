import os
from datetime import timedelta

class Config:
    SECRET_KEY = os.environ.get('SECRET_KEY', 'INSECURE-DEV-KEY-CHANGE-IN-PRODUCTION')
    SQLALCHEMY_TRACK_MODIFICATIONS = False

    # Upload settings
    UPLOAD_FOLDER = os.environ.get('UPLOAD_FOLDER', '/app/uploads')
    MAX_FILE_SIZE_MB = int(os.environ.get('MAX_FILE_SIZE_MB', 100))
    MAX_CONTENT_LENGTH = MAX_FILE_SIZE_MB * 1024 * 1024

    # Session settings
    SESSION_TYPE = 'redis'
    SESSION_PERMANENT = True
    PERMANENT_SESSION_LIFETIME = timedelta(days=7)
    SESSION_COOKIE_HTTPONLY = True
    SESSION_COOKIE_SAMESITE = 'Lax'

    # Celery settings
    CELERY_BROKER_URL = os.environ.get('REDIS_URL', 'redis://localhost:6379/0')
    CELERY_RESULT_BACKEND = os.environ.get('REDIS_URL', 'redis://localhost:6379/0')

    # Storage quota
    STORAGE_QUOTA_MB = int(os.environ.get('STORAGE_QUOTA_MB', 1000))

class DevelopmentConfig(Config):
    DEBUG = True
    SQLALCHEMY_DATABASE_URI = os.environ.get('DATABASE_URL') or 'sqlite:///app.db'
    SESSION_REDIS = os.environ.get('REDIS_URL', 'redis://localhost:6379/0')

class ProductionConfig(Config):
    DEBUG = False
    SESSION_COOKIE_SECURE = True

    SQLALCHEMY_DATABASE_URI = os.environ.get('DATABASE_URL')
    SESSION_REDIS = os.environ.get('REDIS_URL')

config = {
    'development': DevelopmentConfig,
    'production': ProductionConfig,
    'default': DevelopmentConfig
}
