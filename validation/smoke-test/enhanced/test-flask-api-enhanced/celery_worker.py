"""Celery worker entry point."""

import os
from dotenv import load_dotenv

# Load environment variables from .env file
dotenv_path = os.path.join(os.path.dirname(__file__), '.env')
if os.path.exists(dotenv_path):
    load_dotenv(dotenv_path)

from app import create_app
from app.extensions import celery

# Create Flask app to initialize Celery configuration
app = create_app()

# The celery instance is configured in create_app()
# and can be used directly for the worker

if __name__ == '__main__':
    with app.app_context():
        celery.start()
