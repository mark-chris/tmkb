import os
import uuid
from werkzeug.utils import secure_filename
from flask import request, jsonify, send_file, current_app
from flask_login import login_required, current_user
from app import db
from app.files import files_bp
from app.models import File

ALLOWED_EXTENSIONS = {'txt', 'pdf', 'png', 'jpg', 'jpeg', 'gif', 'doc', 'docx', 'xls', 'xlsx', 'csv', 'zip'}

def allowed_file(filename):
    return '.' in filename and filename.rsplit('.', 1)[1].lower() in ALLOWED_EXTENSIONS

@files_bp.route('/upload', methods=['POST'])
@login_required
def upload_file():
    if 'file' not in request.files:
        return jsonify({'error': 'No file provided'}), 400

    file = request.files['file']

    if file.filename == '':
        return jsonify({'error': 'No file selected'}), 400

    if not allowed_file(file.filename):
        return jsonify({'error': 'File type not allowed'}), 400

    # Check file size
    # Note: There is a minor race condition here as we re-read the file size,
    # but this is acceptable since the Celery task will re-validate the saved file
    file.seek(0, os.SEEK_END)
    file_size = file.tell()
    file.seek(0)

    file_size_mb = file_size / (1024 * 1024)

    if file_size > current_app.config['MAX_CONTENT_LENGTH']:
        return jsonify({'error': f'File too large. Max size: {current_app.config["MAX_FILE_SIZE_MB"]}MB'}), 413

    # Check organization quota
    if not current_user.organization.has_storage_quota(file_size_mb):
        return jsonify({'error': 'Organization storage quota exceeded'}), 413

    # Generate file ID and secure filename
    file_id = str(uuid.uuid4())
    filename = secure_filename(file.filename)

    # Create directory structure: uploads/{org_id}/{file_id}/
    upload_dir = os.path.join(
        current_app.config['UPLOAD_FOLDER'],
        str(current_user.organization_id),
        file_id
    )
    os.makedirs(upload_dir, exist_ok=True)

    file_path = os.path.join(upload_dir, filename)

    # Save file
    file.save(file_path)

    # Create database record
    file_record = File(
        id=file_id,
        filename=filename,
        file_path=file_path,
        file_size_bytes=file_size,
        mime_type=file.content_type,
        status='pending',
        organization_id=current_user.organization_id,
        uploaded_by=current_user.id
    )
    db.session.add(file_record)
    db.session.commit()

    # Queue Celery task
    from app.tasks import validate_file
    validate_file.delay(file_id)

    return jsonify({
        'file_id': file_id,
        'filename': filename,
        'status': 'pending'
    }), 201

@files_bp.route('', methods=['GET'])
@login_required
def list_files():
    page = request.args.get('page', 1, type=int)
    limit = request.args.get('limit', 20, type=int)

    # Limit pagination
    limit = min(limit, 100)

    # Query files for current organization
    query = File.query.filter_by(organization_id=current_user.organization_id)
    total = query.count()

    files = query.order_by(File.uploaded_at.desc()).paginate(
        page=page, per_page=limit, error_out=False
    )

    return jsonify({
        'files': [f.to_dict() for f in files.items],
        'total': total,
        'page': page,
        'pages': files.pages
    }), 200

@files_bp.route('/<file_id>', methods=['GET'])
@login_required
def get_file(file_id):
    file = File.query.filter_by(
        id=file_id,
        organization_id=current_user.organization_id
    ).first()

    if not file:
        return jsonify({'error': 'File not found'}), 404

    return jsonify(file.to_dict()), 200

@files_bp.route('/<file_id>/download', methods=['GET'])
@login_required
def download_file(file_id):
    file = File.query.filter_by(
        id=file_id,
        organization_id=current_user.organization_id
    ).first()

    if not file:
        return jsonify({'error': 'File not found'}), 404

    if file.status != 'ready':
        return jsonify({'error': f'File is not ready for download. Status: {file.status}'}), 400

    if not os.path.exists(file.file_path):
        return jsonify({'error': 'File not found on disk'}), 404

    # Validate file_path is within upload directory to prevent path traversal
    upload_folder = current_app.config['UPLOAD_FOLDER']
    real_path = os.path.realpath(file.file_path)
    real_upload = os.path.realpath(upload_folder)

    if not real_path.startswith(real_upload):
        return jsonify({'error': 'Invalid file path'}), 400

    return send_file(
        file.file_path,
        as_attachment=True,
        download_name=file.filename,
        mimetype=file.mime_type
    )
