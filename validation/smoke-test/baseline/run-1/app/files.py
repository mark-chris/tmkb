from flask import Blueprint, request, jsonify, current_app
from flask_login import login_required, current_user
from werkzeug.utils import secure_filename
from app import db, limiter
from app.models import File
from app.utils import sanitize_filename, require_org_access
from app.tasks import process_file
import os
import uuid
import logging

files_bp = Blueprint('files', __name__)
logger = logging.getLogger(__name__)

@files_bp.route('', methods=['POST'])
@login_required
@limiter.limit("10 per minute")
def upload_file():
    """Upload file and queue for background processing"""
    # Check if file is in request
    if 'file' not in request.files:
        return jsonify({'error': 'No file provided'}), 400

    file = request.files['file']

    # Check if filename is empty
    if file.filename == '':
        return jsonify({'error': 'No file selected'}), 400

    try:
        # Sanitize filename
        original_filename = sanitize_filename(file.filename)

        # Generate unique file ID
        file_id = str(uuid.uuid4())

        # Save to temporary location
        temp_path = os.path.join(current_app.config['TEMP_FOLDER'], file_id)
        file.save(temp_path)

        # Create database record
        file_record = File(
            id=file_id,
            filename=original_filename,
            organization_id=current_user.organization_id,
            uploaded_by=current_user.id,
            status='pending'
        )

        db.session.add(file_record)
        db.session.commit()

        # Queue background task
        task = process_file.delay(file_id)

        # Update with task ID
        file_record.celery_task_id = task.id
        db.session.commit()

        logger.info(f"File {file_id} uploaded by user {current_user.username}, task {task.id} queued")

        return jsonify(file_record.to_dict()), 201

    except Exception as e:
        logger.error(f"Error uploading file: {str(e)}")
        db.session.rollback()
        return jsonify({'error': 'Failed to upload file'}), 500

@files_bp.route('', methods=['GET'])
@login_required
def list_files():
    """List files for user's organization with pagination"""
    # Parse pagination parameters
    try:
        limit = min(int(request.args.get('limit', 20)), 100)
        offset = int(request.args.get('offset', 0))
    except ValueError:
        return jsonify({'error': 'Invalid pagination parameters'}), 400

    # Query files for organization
    query = File.query.filter_by(organization_id=current_user.organization_id)

    # Get total count
    total = query.count()

    # Get paginated results (newest first)
    files = query.order_by(File.created_at.desc()).limit(limit).offset(offset).all()

    return jsonify({
        'files': [f.to_dict() for f in files],
        'total': total,
        'limit': limit,
        'offset': offset
    }), 200

@files_bp.route('/<file_id>', methods=['GET'])
@login_required
def get_file(file_id):
    """Get details for a specific file"""
    file = File.query.get_or_404(file_id)

    # Verify organization access
    require_org_access(file)

    return jsonify(file.to_dict(include_uploader=True)), 200
