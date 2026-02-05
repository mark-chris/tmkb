"""
WSGI entry point for Flask application.

This file is used by production WSGI servers (e.g., Gunicorn, uWSGI)
and by the Flask development server via 'flask run'.
"""

import os
from app import create_app

# Determine environment from FLASK_ENV variable (defaults to development)
config_name = os.environ.get('FLASK_ENV', 'development')
app = create_app(config_name)

if __name__ == '__main__':
    app.run()
