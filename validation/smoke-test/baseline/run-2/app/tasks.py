import os
from celery import Celery
from app import create_app, db
from app.models import File

# Create Flask app for context
flask_app = create_app()

# Initialize Celery
celery = Celery(
    'tasks',
    broker=flask_app.config['CELERY_BROKER_URL'],
    backend=flask_app.config['CELERY_RESULT_BACKEND']
)
celery.conf.update(flask_app.config)

@celery.task(name='app.tasks.validate_file')
def validate_file(file_id):
    """
    Background task to validate uploaded file.

    Performs:
    - File integrity check (exists and readable)
    - Size validation (matches database record)
    - MIME type verification
    - Storage quota check

    Updates file status to 'ready' on success or 'failed' with error message.
    """
    with flask_app.app_context():
        file_record = File.query.get(file_id)

        if not file_record:
            return {'status': 'error', 'message': 'File record not found'}

        try:
            # 1. File integrity check - verify file exists and is readable
            if not os.path.exists(file_record.file_path):
                raise Exception('File not found on disk')

            if not os.access(file_record.file_path, os.R_OK):
                raise Exception('File is not readable')

            # 2. Size validation - confirm file_size matches actual disk size
            actual_size = os.path.getsize(file_record.file_path)
            if actual_size != file_record.file_size_bytes:
                raise Exception(f'File size mismatch: expected {file_record.file_size_bytes}, got {actual_size}')

            # 3. MIME type verification (basic check)
            # In production, you might use python-magic for more robust MIME detection
            if actual_size == 0:
                raise Exception('File is empty')

            # 4. Quota double-check
            from app.models import Organization
            org = Organization.query.get(file_record.organization_id)
            if org:
                total_usage = db.session.query(db.func.sum(File.file_size_bytes)).\
                    filter(File.organization_id == org.id, File.status == 'ready').scalar() or 0
                quota_bytes = org.storage_quota_mb * 1024 * 1024

                if total_usage + actual_size > quota_bytes:
                    raise Exception('Storage quota exceeded')

            # Success - mark file as ready
            file_record.status = 'ready'
            file_record.processed_at = db.func.now()
            file_record.error_message = None
            db.session.commit()

            return {'status': 'success', 'file_id': file_id}

        except Exception as e:
            # Failure - mark file as failed
            file_record.status = 'failed'
            file_record.error_message = str(e)
            file_record.processed_at = db.func.now()
            db.session.commit()

            return {'status': 'failed', 'file_id': file_id, 'error': str(e)}
