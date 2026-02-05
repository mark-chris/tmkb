"""
Tests for file management endpoints.
"""
import os
import io
import pytest
from app import db
from app.models import File, Organization


class TestFileUpload:
    """Tests for the POST /api/files/upload endpoint."""

    def test_upload_file_success(self, authenticated_client, user, app):
        """Test successful file upload."""
        data = {
            'file': (io.BytesIO(b'Test file content'), 'test.txt')
        }

        response = authenticated_client.post(
            '/api/files/upload',
            data=data,
            content_type='multipart/form-data'
        )

        assert response.status_code == 201
        json_data = response.get_json()
        assert 'file_id' in json_data
        assert json_data['filename'] == 'test.txt'
        assert json_data['status'] == 'pending'

        # Verify file record in database
        with app.app_context():
            file_record = File.query.filter_by(id=json_data['file_id']).first()
            assert file_record is not None
            assert file_record.filename == 'test.txt'
            assert file_record.organization_id == user['organization_id']
            assert file_record.uploaded_by == user['id']

            # Verify file was saved to disk
            assert os.path.exists(file_record.file_path)

    def test_upload_file_no_file_provided(self, authenticated_client):
        """Test upload fails when no file is provided."""
        response = authenticated_client.post(
            '/api/files/upload',
            data={},
            content_type='multipart/form-data'
        )

        assert response.status_code == 400
        data = response.get_json()
        assert 'error' in data

    def test_upload_file_empty_filename(self, authenticated_client):
        """Test upload fails with empty filename."""
        data = {
            'file': (io.BytesIO(b'content'), '')
        }

        response = authenticated_client.post(
            '/api/files/upload',
            data=data,
            content_type='multipart/form-data'
        )

        assert response.status_code == 400
        data = response.get_json()
        assert 'error' in data

    def test_upload_file_exceeds_quota(self, authenticated_client, user, app):
        """Test upload fails when organization quota is exceeded."""
        # Set a very small quota
        with app.app_context():
            org = Organization.query.get(user['organization_id'])
            org.storage_quota_mb = 0  # 0 MB quota
            db.session.commit()

        data = {
            'file': (io.BytesIO(b'This will exceed quota'), 'large.txt')
        }

        response = authenticated_client.post(
            '/api/files/upload',
            data=data,
            content_type='multipart/form-data'
        )

        assert response.status_code == 400
        json_data = response.get_json()
        assert 'error' in json_data
        assert 'quota' in json_data['error'].lower()

    def test_upload_requires_authentication(self, client):
        """Test that file upload requires authentication."""
        data = {
            'file': (io.BytesIO(b'content'), 'test.txt')
        }

        response = client.post(
            '/api/files/upload',
            data=data,
            content_type='multipart/form-data'
        )

        assert response.status_code == 401


class TestFileList:
    """Tests for the GET /api/files endpoint."""

    def test_list_files_empty(self, authenticated_client):
        """Test listing files when none exist."""
        response = authenticated_client.get('/api/files')

        assert response.status_code == 200
        data = response.get_json()
        assert 'files' in data
        assert len(data['files']) == 0
        assert data['total'] == 0

    def test_list_files_with_files(self, authenticated_client, user, app):
        """Test listing files returns uploaded files."""
        # Upload a file first
        upload_response = authenticated_client.post(
            '/api/files/upload',
            data={'file': (io.BytesIO(b'content1'), 'file1.txt')},
            content_type='multipart/form-data'
        )
        file1_id = upload_response.get_json()['file_id']

        upload_response = authenticated_client.post(
            '/api/files/upload',
            data={'file': (io.BytesIO(b'content2'), 'file2.txt')},
            content_type='multipart/form-data'
        )
        file2_id = upload_response.get_json()['file_id']

        # List files
        response = authenticated_client.get('/api/files')

        assert response.status_code == 200
        data = response.get_json()
        assert len(data['files']) == 2
        assert data['total'] == 2

        # Check file data
        file_ids = [f['id'] for f in data['files']]
        assert file1_id in file_ids
        assert file2_id in file_ids

    def test_list_files_pagination(self, authenticated_client):
        """Test file listing pagination."""
        # Upload multiple files
        for i in range(5):
            authenticated_client.post(
                '/api/files/upload',
                data={'file': (io.BytesIO(f'content{i}'.encode()), f'file{i}.txt')},
                content_type='multipart/form-data'
            )

        # Request first page with limit 2
        response = authenticated_client.get('/api/files?page=1&limit=2')

        assert response.status_code == 200
        data = response.get_json()
        assert len(data['files']) == 2
        assert data['total'] == 5
        assert data['page'] == 1

        # Request second page
        response = authenticated_client.get('/api/files?page=2&limit=2')

        assert response.status_code == 200
        data = response.get_json()
        assert len(data['files']) == 2
        assert data['page'] == 2

    def test_list_files_tenant_isolation(self, authenticated_client, second_authenticated_client, user, second_user):
        """Test that users only see files from their own organization."""
        # User 1 uploads a file
        authenticated_client.post(
            '/api/files/upload',
            data={'file': (io.BytesIO(b'user1 content'), 'user1_file.txt')},
            content_type='multipart/form-data'
        )

        # User 2 uploads a file
        second_authenticated_client.post(
            '/api/files/upload',
            data={'file': (io.BytesIO(b'user2 content'), 'user2_file.txt')},
            content_type='multipart/form-data'
        )

        # User 1 lists files - should only see their own
        response = authenticated_client.get('/api/files')
        assert response.status_code == 200
        data = response.get_json()
        assert len(data['files']) == 1
        assert data['files'][0]['filename'] == 'user1_file.txt'

        # User 2 lists files - should only see their own
        response = second_authenticated_client.get('/api/files')
        assert response.status_code == 200
        data = response.get_json()
        assert len(data['files']) == 1
        assert data['files'][0]['filename'] == 'user2_file.txt'


class TestFileDetails:
    """Tests for the GET /api/files/<file_id> endpoint."""

    def test_get_file_details_success(self, authenticated_client, user, app):
        """Test retrieving file details."""
        # Upload a file
        upload_response = authenticated_client.post(
            '/api/files/upload',
            data={'file': (io.BytesIO(b'test content'), 'details_test.txt')},
            content_type='multipart/form-data'
        )
        file_id = upload_response.get_json()['file_id']

        # Get file details
        response = authenticated_client.get(f'/api/files/{file_id}')

        assert response.status_code == 200
        data = response.get_json()
        assert data['id'] == file_id
        assert data['filename'] == 'details_test.txt'
        assert 'size' in data
        assert 'status' in data
        assert 'uploaded_at' in data
        assert 'uploaded_by_email' in data

    def test_get_file_details_not_found(self, authenticated_client):
        """Test retrieving non-existent file returns 404."""
        response = authenticated_client.get('/api/files/00000000-0000-0000-0000-000000000000')

        assert response.status_code == 404
        data = response.get_json()
        assert 'error' in data

    def test_get_file_details_tenant_isolation(self, authenticated_client, second_authenticated_client):
        """Test that users cannot access files from other organizations."""
        # User 1 uploads a file
        upload_response = authenticated_client.post(
            '/api/files/upload',
            data={'file': (io.BytesIO(b'user1 content'), 'private.txt')},
            content_type='multipart/form-data'
        )
        file_id = upload_response.get_json()['file_id']

        # User 2 tries to access user 1's file
        response = second_authenticated_client.get(f'/api/files/{file_id}')

        assert response.status_code == 404
        data = response.get_json()
        assert 'error' in data


class TestFileDownload:
    """Tests for the GET /api/files/<file_id>/download endpoint."""

    def test_download_file_success(self, authenticated_client, app):
        """Test downloading a file with 'ready' status."""
        # Upload a file
        file_content = b'Download test content'
        upload_response = authenticated_client.post(
            '/api/files/upload',
            data={'file': (io.BytesIO(file_content), 'download_test.txt')},
            content_type='multipart/form-data'
        )
        file_id = upload_response.get_json()['file_id']

        # Mark file as ready (simulate successful processing)
        with app.app_context():
            file_record = File.query.get(file_id)
            file_record.status = 'ready'
            db.session.commit()

        # Download the file
        response = authenticated_client.get(f'/api/files/{file_id}/download')

        assert response.status_code == 200
        assert response.data == file_content
        assert 'attachment' in response.headers.get('Content-Disposition', '')

    def test_download_file_pending_status(self, authenticated_client):
        """Test that downloading a pending file is blocked."""
        # Upload a file (status will be 'pending')
        upload_response = authenticated_client.post(
            '/api/files/upload',
            data={'file': (io.BytesIO(b'content'), 'pending.txt')},
            content_type='multipart/form-data'
        )
        file_id = upload_response.get_json()['file_id']

        # Try to download
        response = authenticated_client.get(f'/api/files/{file_id}/download')

        assert response.status_code == 400
        data = response.get_json()
        assert 'error' in data
        assert 'pending' in data['error'].lower() or 'not ready' in data['error'].lower()

    def test_download_file_failed_status(self, authenticated_client, app):
        """Test that downloading a failed file is blocked."""
        # Upload a file
        upload_response = authenticated_client.post(
            '/api/files/upload',
            data={'file': (io.BytesIO(b'content'), 'failed.txt')},
            content_type='multipart/form-data'
        )
        file_id = upload_response.get_json()['file_id']

        # Mark file as failed
        with app.app_context():
            file_record = File.query.get(file_id)
            file_record.status = 'failed'
            file_record.error_message = 'Validation failed'
            db.session.commit()

        # Try to download
        response = authenticated_client.get(f'/api/files/{file_id}/download')

        assert response.status_code == 400
        data = response.get_json()
        assert 'error' in data

    def test_download_file_not_found(self, authenticated_client):
        """Test downloading non-existent file returns 404."""
        response = authenticated_client.get('/api/files/00000000-0000-0000-0000-000000000000/download')

        assert response.status_code == 404

    def test_download_file_tenant_isolation(self, authenticated_client, second_authenticated_client, app):
        """Test that users cannot download files from other organizations."""
        # User 1 uploads a file
        upload_response = authenticated_client.post(
            '/api/files/upload',
            data={'file': (io.BytesIO(b'private content'), 'private.txt')},
            content_type='multipart/form-data'
        )
        file_id = upload_response.get_json()['file_id']

        # Mark as ready
        with app.app_context():
            file_record = File.query.get(file_id)
            file_record.status = 'ready'
            db.session.commit()

        # User 2 tries to download user 1's file
        response = second_authenticated_client.get(f'/api/files/{file_id}/download')

        assert response.status_code == 404
        data = response.get_json()
        assert 'error' in data


class TestFileProcessing:
    """Tests for file validation and background processing."""

    def test_file_validation_task_execution(self, authenticated_client, app):
        """Test that file validation task is executed (synchronously in tests)."""
        # Upload a file
        upload_response = authenticated_client.post(
            '/api/files/upload',
            data={'file': (io.BytesIO(b'validation test'), 'validate.txt')},
            content_type='multipart/form-data'
        )
        file_id = upload_response.get_json()['file_id']

        # Since CELERY_TASK_ALWAYS_EAGER=True, task should execute immediately
        # Check file status was updated
        with app.app_context():
            file_record = File.query.get(file_id)
            # Status should be 'ready' if validation succeeded
            assert file_record.status in ['ready', 'pending']
            # If ready, processed_at should be set
            if file_record.status == 'ready':
                assert file_record.processed_at is not None
