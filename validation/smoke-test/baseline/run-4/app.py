import os
import uuid
from datetime import datetime
from pathlib import Path

from celery import Celery
from flask import Flask, jsonify, request, send_from_directory
from flask_login import LoginManager, UserMixin, current_user, login_required, login_user
from flask_sqlalchemy import SQLAlchemy
from werkzeug.security import check_password_hash, generate_password_hash
from werkzeug.utils import secure_filename

BASE_DIR = Path(__file__).resolve().parent
UPLOAD_DIR = BASE_DIR / "uploads"


db = SQLAlchemy()
login_manager = LoginManager()


def make_celery(app: Flask) -> Celery:
    celery = Celery(
        app.import_name,
        broker=app.config["CELERY_BROKER_URL"],
        backend=app.config["CELERY_RESULT_BACKEND"],
    )
    celery.conf.update(app.config)

    class ContextTask(celery.Task):
        def __call__(self, *args, **kwargs):
            with app.app_context():
                return self.run(*args, **kwargs)

    celery.Task = ContextTask
    return celery


class Organization(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(120), nullable=False, unique=True)
    created_at = db.Column(db.DateTime, default=datetime.utcnow)


class User(UserMixin, db.Model):
    id = db.Column(db.Integer, primary_key=True)
    email = db.Column(db.String(255), unique=True, nullable=False)
    password_hash = db.Column(db.String(255), nullable=False)
    organization_id = db.Column(db.Integer, db.ForeignKey("organization.id"), nullable=False)
    organization = db.relationship("Organization", backref="users")
    created_at = db.Column(db.DateTime, default=datetime.utcnow)

    def set_password(self, password: str) -> None:
        self.password_hash = generate_password_hash(password)

    def check_password(self, password: str) -> bool:
        return check_password_hash(self.password_hash, password)


class FileUpload(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    organization_id = db.Column(db.Integer, db.ForeignKey("organization.id"), nullable=False)
    organization = db.relationship("Organization", backref="files")
    filename = db.Column(db.String(255), nullable=False)
    storage_key = db.Column(db.String(255), nullable=False, unique=True)
    status = db.Column(db.String(50), nullable=False, default="queued")
    uploaded_at = db.Column(db.DateTime, default=datetime.utcnow)
    processed_at = db.Column(db.DateTime, nullable=True)


app = Flask(__name__)
app.config.update(
    SECRET_KEY=os.environ.get("SECRET_KEY", "dev-secret"),
    SQLALCHEMY_DATABASE_URI=os.environ.get(
        "DATABASE_URL", f"sqlite:///{BASE_DIR / 'app.db'}"
    ),
    SQLALCHEMY_TRACK_MODIFICATIONS=False,
    CELERY_BROKER_URL=os.environ.get("CELERY_BROKER_URL", "redis://localhost:6379/0"),
    CELERY_RESULT_BACKEND=os.environ.get("CELERY_RESULT_BACKEND", "redis://localhost:6379/0"),
    UPLOAD_FOLDER=str(UPLOAD_DIR),
)


db.init_app(app)
login_manager.init_app(app)
login_manager.login_view = "login"
celery = make_celery(app)


@login_manager.user_loader
def load_user(user_id: str):
    return User.query.get(int(user_id))


@celery.task(name="process_uploaded_file")
def process_uploaded_file(file_id: int) -> None:
    file_record = FileUpload.query.get(file_id)
    if not file_record:
        return
    file_record.status = "processing"
    db.session.commit()

    # Simulate processing work.
    # Replace with real parsing, virus scanning, or ETL work.
    file_record.status = "processed"
    file_record.processed_at = datetime.utcnow()
    db.session.commit()


@app.before_first_request
def setup_storage() -> None:
    UPLOAD_DIR.mkdir(parents=True, exist_ok=True)
    db.create_all()


@app.post("/auth/register")
def register():
    payload = request.get_json(force=True)
    org_name = payload.get("organization")
    email = payload.get("email")
    password = payload.get("password")
    if not org_name or not email or not password:
        return jsonify({"error": "organization, email, and password are required"}), 400

    org = Organization.query.filter_by(name=org_name).first()
    if not org:
        org = Organization(name=org_name)
        db.session.add(org)
        db.session.commit()

    if User.query.filter_by(email=email).first():
        return jsonify({"error": "email already registered"}), 400

    user = User(email=email, organization_id=org.id)
    user.set_password(password)
    db.session.add(user)
    db.session.commit()
    return jsonify({"message": "registered", "user_id": user.id}), 201


@app.post("/auth/login")
def login():
    payload = request.get_json(force=True)
    email = payload.get("email")
    password = payload.get("password")
    if not email or not password:
        return jsonify({"error": "email and password required"}), 400

    user = User.query.filter_by(email=email).first()
    if not user or not user.check_password(password):
        return jsonify({"error": "invalid credentials"}), 401

    login_user(user)
    return jsonify({"message": "logged in"})


@app.post("/files")
@login_required
def upload_file():
    if "file" not in request.files:
        return jsonify({"error": "file is required"}), 400

    file = request.files["file"]
    if file.filename == "":
        return jsonify({"error": "file is required"}), 400

    safe_name = secure_filename(file.filename)
    storage_key = f"{uuid.uuid4().hex}_{safe_name}"
    save_path = UPLOAD_DIR / storage_key
    file.save(save_path)

    record = FileUpload(
        organization_id=current_user.organization_id,
        filename=safe_name,
        storage_key=storage_key,
        status="queued",
    )
    db.session.add(record)
    db.session.commit()

    process_uploaded_file.delay(record.id)

    return (
        jsonify(
            {
                "id": record.id,
                "filename": record.filename,
                "status": record.status,
            }
        ),
        201,
    )


@app.get("/files")
@login_required
def list_files():
    files = (
        FileUpload.query.filter_by(organization_id=current_user.organization_id)
        .order_by(FileUpload.uploaded_at.desc())
        .all()
    )
    return jsonify(
        [
            {
                "id": f.id,
                "filename": f.filename,
                "status": f.status,
                "uploaded_at": f.uploaded_at.isoformat(),
                "processed_at": f.processed_at.isoformat() if f.processed_at else None,
            }
            for f in files
        ]
    )


@app.get("/files/<int:file_id>")
@login_required
def get_file(file_id: int):
    file_record = FileUpload.query.filter_by(
        id=file_id, organization_id=current_user.organization_id
    ).first()
    if not file_record:
        return jsonify({"error": "not found"}), 404

    return jsonify(
        {
            "id": file_record.id,
            "filename": file_record.filename,
            "status": file_record.status,
            "uploaded_at": file_record.uploaded_at.isoformat(),
            "processed_at": file_record.processed_at.isoformat()
            if file_record.processed_at
            else None,
        }
    )


@app.get("/files/<int:file_id>/download")
@login_required
def download_file(file_id: int):
    file_record = FileUpload.query.filter_by(
        id=file_id, organization_id=current_user.organization_id
    ).first()
    if not file_record:
        return jsonify({"error": "not found"}), 404

    return send_from_directory(app.config["UPLOAD_FOLDER"], file_record.storage_key)


if __name__ == "__main__":
    app.run(debug=True)
