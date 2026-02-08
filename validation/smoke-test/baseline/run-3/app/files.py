import os

from flask import Blueprint, current_app, jsonify, request
from flask_login import current_user, login_required
from werkzeug.utils import secure_filename

from app import db
from app.models import File
from app.tasks import process_file

files_bp = Blueprint("files", __name__)


@files_bp.route("/upload", methods=["POST"])
@login_required
def upload():
    if "file" not in request.files:
        return jsonify({"error": "No file part in the request"}), 400

    f = request.files["file"]
    if f.filename == "":
        return jsonify({"error": "No file selected"}), 400

    filename = secure_filename(f.filename)
    org_dir = os.path.join(
        current_app.config["UPLOAD_FOLDER"], str(current_user.organization_id)
    )
    os.makedirs(org_dir, exist_ok=True)

    stored_path = os.path.join(org_dir, filename)

    # Avoid overwriting: append a counter if file exists
    base, ext = os.path.splitext(filename)
    counter = 1
    while os.path.exists(stored_path):
        stored_path = os.path.join(org_dir, f"{base}_{counter}{ext}")
        counter += 1

    f.save(stored_path)

    file_record = File(
        filename=filename,
        stored_path=stored_path,
        mimetype=f.content_type,
        status="pending",
        uploaded_by=current_user.id,
        organization_id=current_user.organization_id,
    )
    db.session.add(file_record)
    db.session.commit()

    try:
        process_file.delay(file_record.id)
    except Exception:
        current_app.logger.warning("Could not dispatch processing task for file %s", file_record.id)

    return jsonify({"message": "File uploaded", "file": file_record.to_dict()}), 202


@files_bp.route("/", methods=["GET"])
@login_required
def list_files():
    page = request.args.get("page", 1, type=int)
    per_page = request.args.get("per_page", 20, type=int)
    per_page = min(per_page, 100)

    query = File.query.filter_by(organization_id=current_user.organization_id).order_by(
        File.created_at.desc()
    )
    pagination = query.paginate(page=page, per_page=per_page, error_out=False)

    return jsonify(
        {
            "files": [f.to_dict() for f in pagination.items],
            "total": pagination.total,
            "page": pagination.page,
            "pages": pagination.pages,
        }
    )


@files_bp.route("/<int:file_id>", methods=["GET"])
@login_required
def get_file(file_id):
    file_record = db.session.get(File, file_id)
    if not file_record or file_record.organization_id != current_user.organization_id:
        return jsonify({"error": "File not found"}), 404

    return jsonify({"file": file_record.to_dict()})
