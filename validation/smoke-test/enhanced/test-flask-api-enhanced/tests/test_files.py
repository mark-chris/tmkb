"""File operations tests."""

import io
import pytest
from app.models.file import File


def test_upload_file(authenticated_client1, user1, db):
    """Test file upload."""
    data = {
        'file': (io.BytesIO(b'test file content'), 'test.txt')
    }

    response = authenticated_client1.post(
        '/files',
        data=data,
        content_type='multipart/form-data'
    )

    assert response.status_code == 201
    response_data = response.get_json()
    assert response_data['message'] == 'File uploaded successfully'
    assert response_data['file']['filename'] == 'test.txt'
    assert response_data['file']['status'] == 'pending'

    # Verify file in database
    file_record = File.query.filter_by(
        original_filename='test.txt',
        organization_id=user1.organization_id
    ).first()
    assert file_record is not None
    assert file_record.uploaded_by_user_id == user1.id


def test_upload_file_invalid_type(authenticated_client1):
    """Test file upload with invalid file type."""
    data = {
        'file': (io.BytesIO(b'executable content'), 'malware.exe')
    }

    response = authenticated_client1.post(
        '/files',
        data=data,
        content_type='multipart/form-data'
    )

    assert response.status_code == 400
    response_data = response.get_json()
    assert 'not allowed' in response_data['error'].lower()


def test_upload_file_no_file(authenticated_client1):
    """Test file upload without file."""
    response = authenticated_client1.post(
        '/files',
        data={},
        content_type='multipart/form-data'
    )

    assert response.status_code == 400


def test_list_files(authenticated_client1, user1, db):
    """Test listing files."""
    # Create test files
    file1 = File(
        filename='file1.txt',
        original_filename='file1.txt',
        filepath='organizations/1/file1.txt',
        mimetype='text/plain',
        size_bytes=100,
        organization_id=user1.organization_id,
        uploaded_by_user_id=user1.id
    )
    file2 = File(
        filename='file2.txt',
        original_filename='file2.txt',
        filepath='organizations/1/file2.txt',
        mimetype='text/plain',
        size_bytes=200,
        organization_id=user1.organization_id,
        uploaded_by_user_id=user1.id
    )
    db.session.add_all([file1, file2])
    db.session.commit()

    response = authenticated_client1.get('/files')

    assert response.status_code == 200
    data = response.get_json()
    assert len(data['files']) == 2


def test_get_file(authenticated_client1, user1, db):
    """Test getting file metadata."""
    file = File(
        filename='test.txt',
        original_filename='test.txt',
        filepath='organizations/1/test.txt',
        mimetype='text/plain',
        size_bytes=100,
        organization_id=user1.organization_id,
        uploaded_by_user_id=user1.id
    )
    db.session.add(file)
    db.session.commit()

    response = authenticated_client1.get(f'/files/{file.id}')

    assert response.status_code == 200
    data = response.get_json()
    assert data['filename'] == 'test.txt'
    assert data['uploaded_by'] == user1.id


def test_delete_file(authenticated_client1, user1, db):
    """Test file deletion."""
    file = File(
        filename='test.txt',
        original_filename='test.txt',
        filepath='organizations/1/test.txt',
        mimetype='text/plain',
        size_bytes=100,
        organization_id=user1.organization_id,
        uploaded_by_user_id=user1.id
    )
    db.session.add(file)
    db.session.commit()
    file_id = file.id

    response = authenticated_client1.delete(f'/files/{file_id}')

    assert response.status_code == 200

    # Verify soft delete
    db.session.expire_all()
    file = File.query.get(file_id)
    assert file.deleted_at is not None
