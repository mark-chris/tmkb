"""File processing Celery task with comprehensive authorization."""

import os
import logging
from datetime import datetime
from celery import Task
from app.extensions import celery, db
from app.models.file import File
from app.models.user import User

logger = logging.getLogger(__name__)


class AuthorizationError(Exception):
    """Raised when background job authorization fails."""
    pass


@celery.task(bind=True, max_retries=3)
def process_file_task(self, file_id, user_id, organization_id):
    """
    Process uploaded file with complete authorization re-validation.

    Security (TMKB-AUTHZ-001):
    - Re-validates ALL authorization checks from endpoint
    - Verifies tenant_id matches at every step
    - Checks user still exists and belongs to organization
    - Verifies file not deleted
    - Does NOT trust authorization from original request

    Args:
        file_id: File record ID
        user_id: User who uploaded (for audit and authorization)
        organization_id: Tenant ID for authorization

    Returns:
        dict: Processing result

    Raises:
        AuthorizationError: If any authorization check fails
    """
    logger.info(f"Processing file {file_id} for org {organization_id}")

    try:
        # AUTHORIZATION CHECK 1: Load file with EXPLICIT tenant_id (TMKB-AUTHZ-004)
        file_record = File.get_for_tenant(file_id, tenant_id=organization_id)

        if not file_record:
            logger.warning(f"File {file_id} not found for org {organization_id}")
            return {'status': 'error', 'message': 'File not found'}

        # AUTHORIZATION CHECK 2: File belongs to claimed organization
        if file_record.organization_id != organization_id:
            logger.error(
                f"SECURITY: Tenant mismatch for file {file_id}. "
                f"File belongs to org {file_record.organization_id}, "
                f"job claimed org {organization_id}"
            )
            raise AuthorizationError("Tenant mismatch in background job")

        # AUTHORIZATION CHECK 3: User still exists and belongs to organization
        user = User.query.get(user_id)
        if not user:
            logger.error(f"SECURITY: User {user_id} no longer exists")
            raise AuthorizationError("Requesting user no longer exists")

        if user.organization_id != organization_id:
            logger.error(
                f"SECURITY: User {user_id} no longer in org {organization_id}. "
                f"Now in org {user.organization_id}"
            )
            raise AuthorizationError("User organization changed")

        if not user.is_active:
            logger.error(f"SECURITY: User {user_id} is no longer active")
            raise AuthorizationError("User account disabled")

        # AUTHORIZATION CHECK 4: File not soft-deleted
        if file_record.deleted_at:
            logger.warning(f"Attempted to process deleted file {file_id}")
            raise AuthorizationError("File has been deleted")

        # AUTHORIZATION CHECK 5: File uploaded by claimed user
        if file_record.uploaded_by_user_id != user_id:
            logger.error(
                f"SECURITY: File {file_id} uploaded by user "
                f"{file_record.uploaded_by_user_id}, job claimed user {user_id}"
            )
            raise AuthorizationError("User mismatch in background job")

        # All authorization checks passed - safe to process
        logger.info(f"Authorization checks passed for file {file_id}")

        file_record.status = 'processing'
        db.session.commit()

        # Get file path (already organization-scoped)
        from app.files.storage import FileStorage
        absolute_path = FileStorage.get_file_path(file_record.filepath)

        if not os.path.exists(absolute_path):
            logger.error(f"File {file_id} not found on disk: {absolute_path}")
            file_record.status = 'failed'
            db.session.commit()
            return {'status': 'error', 'message': 'File not found on disk'}

        # Extract metadata based on file type
        metadata = {}

        if file_record.mimetype.startswith('image/'):
            # Extract image metadata
            try:
                from PIL import Image
                with Image.open(absolute_path) as img:
                    metadata['width'] = img.width
                    metadata['height'] = img.height
                    metadata['format'] = img.format
                    metadata['mode'] = img.mode
                logger.info(f"Extracted image metadata for file {file_id}: {metadata}")
            except Exception as e:
                logger.warning(f"Failed to extract image metadata: {e}")
                metadata['error'] = 'Failed to extract image metadata'

        elif file_record.mimetype == 'application/pdf':
            # Could add PDF metadata extraction here
            metadata['type'] = 'pdf'
            logger.info(f"Identified PDF file {file_id}")

        elif file_record.mimetype == 'text/plain':
            # Extract text file metadata
            try:
                with open(absolute_path, 'r', encoding='utf-8', errors='ignore') as f:
                    content = f.read()
                    metadata['line_count'] = content.count('\n') + 1
                    metadata['char_count'] = len(content)
                    metadata['word_count'] = len(content.split())
                logger.info(f"Extracted text metadata for file {file_id}: {metadata}")
            except Exception as e:
                logger.warning(f"Failed to extract text metadata: {e}")
                metadata['error'] = 'Failed to extract text metadata'

        else:
            metadata['type'] = 'generic'

        # Update file record with processing results
        file_record.metadata = metadata
        file_record.status = 'completed'
        file_record.processed_at = datetime.utcnow()
        db.session.commit()

        logger.info(f"Successfully processed file {file_id}")

        return {
            'status': 'success',
            'file_id': file_id,
            'metadata': metadata
        }

    except AuthorizationError as e:
        # Don't retry authorization failures
        logger.error(f"Authorization error for file {file_id}: {e}")
        try:
            file_record = File.query.get(file_id)
            if file_record:
                file_record.status = 'failed'
                db.session.commit()
        except:
            pass
        raise

    except Exception as e:
        logger.error(f"Error processing file {file_id}: {e}", exc_info=True)

        # Update status on final retry
        if self.request.retries >= self.max_retries:
            try:
                file_record = File.query.get(file_id)
                if file_record and file_record.organization_id == organization_id:
                    file_record.status = 'failed'
                    db.session.commit()
            except:
                pass

        # Retry on non-authorization errors
        raise self.retry(exc=e, countdown=60 * (self.request.retries + 1))
