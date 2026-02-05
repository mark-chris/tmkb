from celery import Celery
from datetime import datetime
import os
import hashlib
import magic
import logging
import shutil

# Create Celery instance with config from environment
celery = Celery(__name__)
celery.conf.update(
    broker_url=os.environ.get('CELERY_BROKER_URL', 'redis://localhost:6379/0'),
    result_backend=os.environ.get('CELERY_RESULT_BACKEND', 'redis://localhost:6379/0'),
    task_serializer='json',
    accept_content=['json'],
    result_serializer='json',
    timezone='UTC',
    enable_utc=True,
)

logger = logging.getLogger(__name__)

@celery.task(bind=True, max_retries=3, default_retry_delay=2)
def process_file(self, file_id):
    """
    Background task to process uploaded file

    Steps:
    1. Load file record from database
    2. Read file from temporary location
    3. Extract metadata (size, MIME type, SHA256 hash)
    4. Move file to permanent storage
    5. Update database record
    6. Clean up temporary file
    """
    # Import inside task to avoid circular imports
    from app import create_app, db
    from app.models import File

    flask_app = create_app()

    with flask_app.app_context():
        try:
            # Load file record
            file_record = File.query.get(file_id)
            if not file_record:
                logger.error(f"File {file_id} not found in database")
                return {'status': 'error', 'message': 'File not found'}

            # Update status to processing
            file_record.status = 'processing'
            db.session.commit()

            # Get temp file path
            temp_path = os.path.join(flask_app.config['TEMP_FOLDER'], file_id)

            if not os.path.exists(temp_path):
                raise FileNotFoundError(f"Temporary file not found: {temp_path}")

            # Extract metadata
            file_size = os.path.getsize(temp_path)

            # Detect MIME type
            mime = magic.Magic(mime=True)
            mime_type = mime.from_file(temp_path)

            # Calculate SHA256 hash
            sha256_hash = hashlib.sha256()
            with open(temp_path, 'rb') as f:
                for chunk in iter(lambda: f.read(4096), b''):
                    sha256_hash.update(chunk)
            sha256_digest = sha256_hash.hexdigest()

            # Create permanent storage directory
            org_upload_dir = os.path.join(
                flask_app.config['UPLOAD_FOLDER'],
                str(file_record.organization_id)
            )
            os.makedirs(org_upload_dir, exist_ok=True)

            # Generate permanent filename
            permanent_filename = f"{file_id}_{file_record.filename}"
            permanent_path = os.path.join(org_upload_dir, permanent_filename)

            # Move file to permanent storage
            shutil.move(temp_path, permanent_path)

            # Update database record
            file_record.filepath = os.path.join(str(file_record.organization_id), permanent_filename)
            file_record.file_size = file_size
            file_record.mime_type = mime_type
            file_record.sha256_hash = sha256_digest
            file_record.status = 'completed'
            file_record.processed_at = datetime.utcnow()
            file_record.error_message = None

            db.session.commit()

            logger.info(f"Successfully processed file {file_id}")

            return {
                'status': 'success',
                'file_id': file_id,
                'file_size': file_size,
                'mime_type': mime_type,
                'sha256_hash': sha256_digest
            }

        except Exception as exc:
            logger.error(f"Error processing file {file_id}: {str(exc)}")

            # Update status to failed
            file_record = File.query.get(file_id)
            if file_record:
                file_record.status = 'failed'
                file_record.error_message = str(exc)
                db.session.commit()

            # Retry if within retry limit
            if self.request.retries < self.max_retries:
                raise self.retry(exc=exc, countdown=2 ** self.request.retries)

            return {
                'status': 'failed',
                'file_id': file_id,
                'error': str(exc)
            }
