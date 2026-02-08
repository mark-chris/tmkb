"""File management routes with tenant isolation."""

import os
from flask import Blueprint, request, jsonify, send_file, current_app
from flask_login import login_required, current_user
from werkzeug.utils import secure_filename
from app.models.file import File
from app.files.storage import FileStorage
from app.extensions import db

files_bp = Blueprint('files', __name__, url_prefix='/files')

# Allowed file extensions
ALLOWED_EXTENSIONS = {'txt', 'pdf', 'png', 'jpg', 'jpeg', 'gif', 'doc', 'docx', 'csv', 'xlsx'}
MAX_FILE_SIZE = 10 * 1024 * 1024  # 10MB


def allowed_file(filename):
    """Check if file extension is allowed."""
    return '.' in filename and \
           filename.rsplit('.', 1)[1].lower() in ALLOWED_EXTENSIONS


@files_bp.route('', methods=['POST'])
@login_required
def upload_file():
    """
    Upload file endpoint with automatic tenant assignment.

    Security (TMKB-AUTHZ-006):
    - File automatically assigned to current_user.organization_id
    - organization_id NEVER accepted from request body
    - Tenant isolation enforced at storage and database level

    Request:
        Content-Type: multipart/form-data
        Body: file (binary)

    Returns:
        201: File uploaded successfully
        400: Invalid request or file type
        500: Upload failed
    """
    # Validate file in request
    if 'file' not in request.files:
        return jsonify({'error': 'No file part in request'}), 400

    file = request.files['file']

    if file.filename == '':
        return jsonify({'error': 'No file selected'}), 400

    if not allowed_file(file.filename):
        return jsonify({
            'error': f'File type not allowed. Allowed types: {", ".join(ALLOWED_EXTENSIONS)}'
        }), 400

    # Check file size
    file.seek(0, os.SEEK_END)
    size = file.tell()
    file.seek(0)

    if size > MAX_FILE_SIZE:
        return jsonify({'error': f'File too large. Maximum size: {MAX_FILE_SIZE / 1024 / 1024}MB'}), 400

    try:
        # Save file to organization-scoped directory
        # SECURITY: organization_id from current_user, NOT from request
        absolute_path, relative_path = FileStorage.save_file(
            file,
            current_user.organization_id,
            file.filename
        )

        # Create database record
        file_record = File(
            filename=os.path.basename(relative_path),
            original_filename=file.filename,
            filepath=relative_path,
            mimetype=file.mimetype or 'application/octet-stream',
            size_bytes=size,
            organization_id=current_user.organization_id,  # EXPLICIT tenant assignment
            uploaded_by_user_id=current_user.id,           # Track uploader
            status='pending'
        )

        db.session.add(file_record)
        db.session.commit()

        # Queue background processing (TMKB-AUTHZ-001: pass authorization context)
        from app.tasks.file_processing import process_file_task
        process_file_task.delay(
            file_id=file_record.id,
            user_id=current_user.id,
            organization_id=current_user.organization_id
        )

        return jsonify({
            'message': 'File uploaded successfully',
            'file': file_record.to_dict()
        }), 201

    except Exception as e:
        db.session.rollback()
        # Clean up file if database insert failed
        if 'absolute_path' in locals() and os.path.exists(absolute_path):
            try:
                os.remove(absolute_path)
            except:
                pass
        current_app.logger.error(f"File upload failed: {e}")
        return jsonify({'error': 'Upload failed'}), 500


@files_bp.route('', methods=['GET'])
@login_required
def list_files():
    """
    List all files for current user's organization.

    Security (TMKB-AUTHZ-004):
    - Uses tenant_query() for automatic filtering
    - Only returns files from current_user.organization_id
    - Same authorization logic as get_file() (TMKB-AUTHZ-002)

    Returns:
        200: List of files
    """
    # Automatic tenant filtering via TenantScopedMixin
    files = File.tenant_query().order_by(File.created_at.desc()).all()

    return jsonify({
        'files': [f.to_dict() for f in files]
    }), 200


@files_bp.route('/<int:file_id>', methods=['GET'])
@login_required
def get_file(file_id):
    """
    Get file metadata by ID.

    Security (TMKB-AUTHZ-004):
    - Uses get_for_tenant() for automatic tenant verification
    - Returns 404 if file doesn't exist or belongs to different org
    - Same authorization logic as list_files() (TMKB-AUTHZ-002)

    Args:
        file_id: File ID

    Returns:
        200: File metadata
        404: File not found or access denied
    """
    # Automatic tenant verification via TenantScopedMixin
    file_record = File.get_for_tenant(file_id)

    return jsonify(file_record.to_dict()), 200


@files_bp.route('/<int:file_id>/download', methods=['GET'])
@login_required
def download_file(file_id):
    """
    Download file content.

    Security:
    - Tenant verification before file access
    - File served from organization-scoped directory
    - Original filename preserved for user convenience

    Args:
        file_id: File ID

    Returns:
        200: File content
        404: File not found or access denied
    """
    # Automatic tenant verification via TenantScopedMixin
    file_record = File.get_for_tenant(file_id)

    # Get absolute path
    absolute_path = FileStorage.get_file_path(file_record.filepath)

    if not os.path.exists(absolute_path):
        current_app.logger.error(f"File not found on disk: {absolute_path}")
        return jsonify({'error': 'File not found on disk'}), 404

    return send_file(
        absolute_path,
        mimetype=file_record.mimetype,
        as_attachment=True,
        download_name=file_record.original_filename
    )


@files_bp.route('/<int:file_id>', methods=['DELETE'])
@login_required
def delete_file(file_id):
    """
    Soft delete a file.

    Security:
    - Tenant verification before deletion
    - Soft delete (sets deleted_at timestamp)
    - File physically removed from storage

    Args:
        file_id: File ID

    Returns:
        200: File deleted
        404: File not found or access denied
    """
    from datetime import datetime

    # Automatic tenant verification via TenantScopedMixin
    file_record = File.get_for_tenant(file_id)

    # Soft delete
    file_record.deleted_at = datetime.utcnow()
    db.session.commit()

    # Optionally delete physical file
    try:
        FileStorage.delete_file(file_record.filepath)
    except Exception as e:
        current_app.logger.warning(f"Failed to delete physical file: {e}")

    return jsonify({'message': 'File deleted successfully'}), 200
