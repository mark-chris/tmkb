import os
from datetime import datetime, timezone

from app import celery, db
from app.models import File


@celery.task
def process_file(file_id):
    file_record = db.session.get(File, file_id)
    if not file_record:
        return {"error": f"File {file_id} not found"}

    try:
        stat = os.stat(file_record.stored_path)
        file_record.size_bytes = stat.st_size
        file_record.status = "processed"
        file_record.processed_at = datetime.now(timezone.utc)
        db.session.commit()
        return {"status": "processed", "file_id": file_id}
    except Exception as exc:
        file_record.status = "failed"
        db.session.commit()
        return {"status": "failed", "file_id": file_id, "error": str(exc)}
