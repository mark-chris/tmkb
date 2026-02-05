from flask import Flask
from flask_sqlalchemy import SQLAlchemy
from flask_login import LoginManager
from flask_session import Session
from flask_wtf.csrf import CSRFProtect
from flask_migrate import Migrate
from redis import Redis
from config import config

db = SQLAlchemy()
login_manager = LoginManager()
sess = Session()
csrf = CSRFProtect()
migrate = Migrate()

def create_app(config_name='default'):
    app = Flask(__name__)

    # Handle both string config names and dict configurations
    if isinstance(config_name, dict):
        # Direct configuration dict (used in testing)
        app.config.update(config_name)
    else:
        # Configuration name (used in production)
        app.config.from_object(config[config_name])

    # Configure Redis connection for Flask-Session (if using Redis session type)
    if app.config.get('SESSION_TYPE') == 'redis' or not app.config.get('SESSION_TYPE'):
        import redis
        redis_url = app.config.get('SESSION_REDIS', 'redis://localhost:6379/0')
        app.config['SESSION_REDIS'] = redis.from_url(redis_url)

    # Initialize extensions
    db.init_app(app)
    login_manager.init_app(app)
    sess.init_app(app)
    csrf.init_app(app)
    migrate.init_app(app, db)

    # Configure Flask-Login
    login_manager.login_view = 'auth.login'
    login_manager.login_message = 'Please log in to access this page.'

    # User loader
    from app.models import User

    @login_manager.user_loader
    def load_user(user_id):
        return User.query.get(int(user_id))

    # Register blueprints
    from app.auth import auth_bp
    from app.files import files_bp

    app.register_blueprint(auth_bp, url_prefix='/api/auth')
    app.register_blueprint(files_bp, url_prefix='/api/files')

    return app
