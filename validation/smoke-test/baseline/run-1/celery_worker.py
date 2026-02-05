from app.tasks import celery

# This file is the entry point for Celery worker
# Usage: celery -A celery_worker worker --loglevel=info

if __name__ == '__main__':
    celery.start()
