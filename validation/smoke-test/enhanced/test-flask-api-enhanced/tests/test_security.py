"""Security tests for tenant isolation (TMKB threat validation)."""

import pytest
from flask_login import login_user
from app.models.file import File
from app.models.user import User


class TestTenantIsolation:
    """Test tenant isolation security (TMKB-AUTHZ-004)."""

    def test_user_cannot_list_files_from_other_org(
        self, authenticated_client1, authenticated_client2, user1, user2, db
    ):
        """
        SECURITY TEST: User from Org A cannot list files from Org B.
        Addresses TMKB-AUTHZ-004.
        """
        # Create file for org1
        file1 = File(
            filename='org1_file.txt',
            original_filename='org1_file.txt',
            filepath='organizations/1/org1_file.txt',
            mimetype='text/plain',
            size_bytes=100,
            organization_id=user1.organization_id,
            uploaded_by_user_id=user1.id
        )
        # Create file for org2
        file2 = File(
            filename='org2_file.txt',
            original_filename='org2_file.txt',
            filepath='organizations/2/org2_file.txt',
            mimetype='text/plain',
            size_bytes=200,
            organization_id=user2.organization_id,
            uploaded_by_user_id=user2.id
        )
        db.session.add_all([file1, file2])
        db.session.commit()

        # User1 should only see org1 files
        response = authenticated_client1.get('/files')
        assert response.status_code == 200
        data = response.get_json()
        assert len(data['files']) == 1
        assert data['files'][0]['filename'] == 'org1_file.txt'

        # User2 should only see org2 files
        response = authenticated_client2.get('/files')
        assert response.status_code == 200
        data = response.get_json()
        assert len(data['files']) == 1
        assert data['files'][0]['filename'] == 'org2_file.txt'

    def test_user_cannot_get_file_from_other_org(
        self, authenticated_client1, user1, user2, db
    ):
        """
        SECURITY TEST: User from Org A cannot get file detail from Org B.
        Addresses TMKB-AUTHZ-004.
        """
        # Create file for org2
        file2 = File(
            filename='org2_file.txt',
            original_filename='org2_file.txt',
            filepath='organizations/2/org2_file.txt',
            mimetype='text/plain',
            size_bytes=200,
            organization_id=user2.organization_id,
            uploaded_by_user_id=user2.id
        )
        db.session.add(file2)
        db.session.commit()

        # User1 attempts to access org2 file
        response = authenticated_client1.get(f'/files/{file2.id}')
        assert response.status_code == 404

    def test_user_cannot_download_file_from_other_org(
        self, authenticated_client1, user1, user2, db
    ):
        """
        SECURITY TEST: User from Org A cannot download file from Org B.
        Addresses TMKB-AUTHZ-004.
        """
        # Create file for org2
        file2 = File(
            filename='org2_file.txt',
            original_filename='org2_file.txt',
            filepath='organizations/2/org2_file.txt',
            mimetype='text/plain',
            size_bytes=200,
            organization_id=user2.organization_id,
            uploaded_by_user_id=user2.id
        )
        db.session.add(file2)
        db.session.commit()

        # User1 attempts to download org2 file
        response = authenticated_client1.get(f'/files/{file2.id}/download')
        assert response.status_code == 404

    def test_user_cannot_delete_file_from_other_org(
        self, authenticated_client1, user1, user2, db
    ):
        """
        SECURITY TEST: User from Org A cannot delete file from Org B.
        Addresses TMKB-AUTHZ-004.
        """
        # Create file for org2
        file2 = File(
            filename='org2_file.txt',
            original_filename='org2_file.txt',
            filepath='organizations/2/org2_file.txt',
            mimetype='text/plain',
            size_bytes=200,
            organization_id=user2.organization_id,
            uploaded_by_user_id=user2.id
        )
        db.session.add(file2)
        db.session.commit()

        # User1 attempts to delete org2 file
        response = authenticated_client1.delete(f'/files/{file2.id}')
        assert response.status_code == 404

        # Verify file still exists
        db.session.expire_all()
        file = File.query.get(file2.id)
        assert file is not None
        assert file.deleted_at is None

    def test_tenant_query_without_authentication_raises_error(self, app, db):
        """
        SECURITY TEST: tenant_query() raises error without authentication.
        Addresses TMKB-AUTHZ-004.
        """
        with app.app_context():
            with pytest.raises(RuntimeError, match="without authenticated user"):
                File.tenant_query().all()

    def test_soft_deleted_files_not_visible(
        self, authenticated_client1, user1, db
    ):
        """
        SECURITY TEST: Soft-deleted files are not returned in queries.
        """
        from datetime import datetime

        # Create file and soft delete it
        file = File(
            filename='deleted.txt',
            original_filename='deleted.txt',
            filepath='organizations/1/deleted.txt',
            mimetype='text/plain',
            size_bytes=100,
            organization_id=user1.organization_id,
            uploaded_by_user_id=user1.id,
            deleted_at=datetime.utcnow()
        )
        db.session.add(file)
        db.session.commit()

        # Should not appear in list
        response = authenticated_client1.get('/files')
        assert response.status_code == 200
        data = response.get_json()
        assert len(data['files']) == 0

        # Should not be accessible by ID
        response = authenticated_client1.get(f'/files/{file.id}')
        assert response.status_code == 404


class TestBackgroundJobSecurity:
    """Test background job authorization (TMKB-AUTHZ-001)."""

    def test_background_job_with_wrong_tenant_id_fails(
        self, app, user1, user2, db
    ):
        """
        SECURITY TEST: Background job with wrong tenant_id fails authorization.
        Addresses TMKB-AUTHZ-001.
        """
        from app.tasks.file_processing import process_file_task, AuthorizationError

        # Create file for org1
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

        # Attempt to process with wrong organization_id
        with app.app_context():
            with pytest.raises(AuthorizationError, match="Tenant mismatch"):
                process_file_task(
                    file_id=file.id,
                    user_id=user1.id,
                    organization_id=user2.organization_id  # WRONG ORG
                )

    def test_background_job_with_wrong_user_id_fails(
        self, app, user1, user2, db
    ):
        """
        SECURITY TEST: Background job with wrong user_id fails authorization.
        Addresses TMKB-AUTHZ-001.
        """
        from app.tasks.file_processing import process_file_task, AuthorizationError

        # Create file for user1 in org1
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

        # Attempt to process claiming different user uploaded it
        with app.app_context():
            with pytest.raises(AuthorizationError, match="User mismatch"):
                process_file_task(
                    file_id=file.id,
                    user_id=user2.id,  # WRONG USER
                    organization_id=user1.organization_id
                )

    def test_background_job_for_inactive_user_fails(
        self, app, user1, db
    ):
        """
        SECURITY TEST: Background job for deactivated user fails.
        Addresses TMKB-AUTHZ-001.
        """
        from app.tasks.file_processing import process_file_task, AuthorizationError

        # Create file
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

        # Deactivate user
        user1.is_active = False
        db.session.commit()

        # Attempt to process
        with app.app_context():
            with pytest.raises(AuthorizationError, match="User account disabled"):
                process_file_task(
                    file_id=file.id,
                    user_id=user1.id,
                    organization_id=user1.organization_id
                )

    def test_background_job_for_deleted_file_fails(
        self, app, user1, db
    ):
        """
        SECURITY TEST: Background job for deleted file fails.
        Addresses TMKB-AUTHZ-001.
        """
        from datetime import datetime
        from app.tasks.file_processing import process_file_task, AuthorizationError

        # Create file and soft delete it
        file = File(
            filename='test.txt',
            original_filename='test.txt',
            filepath='organizations/1/test.txt',
            mimetype='text/plain',
            size_bytes=100,
            organization_id=user1.organization_id,
            uploaded_by_user_id=user1.id,
            deleted_at=datetime.utcnow()
        )
        db.session.add(file)
        db.session.commit()

        # Attempt to process
        with app.app_context():
            with pytest.raises(AuthorizationError, match="File has been deleted"):
                process_file_task(
                    file_id=file.id,
                    user_id=user1.id,
                    organization_id=user1.organization_id
                )


class TestInputValidation:
    """Test input validation security."""

    def test_file_upload_respects_allowed_extensions(
        self, authenticated_client1
    ):
        """Test that only allowed file extensions are accepted."""
        import io

        # Attempt to upload executable
        data = {
            'file': (io.BytesIO(b'malicious content'), 'malware.exe')
        }
        response = authenticated_client1.post(
            '/files',
            data=data,
            content_type='multipart/form-data'
        )
        assert response.status_code == 400

        # Attempt to upload shell script
        data = {
            'file': (io.BytesIO(b'#!/bin/bash'), 'script.sh')
        }
        response = authenticated_client1.post(
            '/files',
            data=data,
            content_type='multipart/form-data'
        )
        assert response.status_code == 400

    def test_malicious_filename_sanitized(
        self, authenticated_client1
    ):
        """Test that malicious filenames are sanitized."""
        import io

        # Attempt path traversal in filename
        data = {
            'file': (io.BytesIO(b'content'), '../../../etc/passwd.txt')
        }
        response = authenticated_client1.post(
            '/files',
            data=data,
            content_type='multipart/form-data'
        )

        # Should either succeed with sanitized name or fail
        # but should NOT create file outside org directory
        if response.status_code == 201:
            data = response.get_json()
            # Verify filename doesn't contain path traversal
            assert '../' not in data['file']['filename']
