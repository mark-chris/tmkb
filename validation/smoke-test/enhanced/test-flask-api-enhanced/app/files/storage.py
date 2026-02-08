"""File storage utilities with organization-scoped isolation."""

import os
import uuid
from werkzeug.utils import secure_filename
from flask import current_app


class FileStorage:
    """
    Manages file storage with organization-scoped directories.

    Security:
    - Each organization has isolated directory
    - Files stored with UUID-based names to prevent conflicts
    - Path traversal protection via secure_filename()

    Storage structure:
        storage/
        └── organizations/
            ├── 1/
            │   ├── abc123.pdf
            │   └── def456.jpg
            └── 2/
                └── ghi789.docx
    """

    @staticmethod
    def get_organization_dir(organization_id):
        """
        Get absolute path to organization's storage directory.

        Args:
            organization_id: Organization ID

        Returns:
            str: Absolute path to org directory
        """
        base_dir = current_app.config['UPLOAD_FOLDER']
        org_dir = os.path.join(base_dir, 'organizations', str(organization_id))
        return org_dir

    @staticmethod
    def ensure_organization_dir(organization_id):
        """
        Ensure organization directory exists.

        Args:
            organization_id: Organization ID

        Returns:
            str: Absolute path to org directory
        """
        org_dir = FileStorage.get_organization_dir(organization_id)
        os.makedirs(org_dir, exist_ok=True)
        return org_dir

    @staticmethod
    def generate_unique_filename(original_filename):
        """
        Generate unique filename preserving extension.

        Args:
            original_filename: Original filename from user

        Returns:
            str: Unique secure filename
        """
        ext = os.path.splitext(original_filename)[1]
        unique_name = f"{uuid.uuid4().hex}{ext}"
        return secure_filename(unique_name)

    @staticmethod
    def save_file(file, organization_id, original_filename):
        """
        Save uploaded file to organization's directory.

        Args:
            file: Werkzeug FileStorage object
            organization_id: Organization ID
            original_filename: Original filename from user

        Returns:
            tuple: (absolute_path, relative_path)
        """
        org_dir = FileStorage.ensure_organization_dir(organization_id)
        filename = FileStorage.generate_unique_filename(original_filename)

        absolute_path = os.path.join(org_dir, filename)
        file.save(absolute_path)

        # Relative path for database storage
        relative_path = os.path.join('organizations', str(organization_id), filename)

        return absolute_path, relative_path

    @staticmethod
    def get_file_path(relative_path):
        """
        Convert relative path to absolute path.

        Args:
            relative_path: Relative path from database

        Returns:
            str: Absolute path to file
        """
        base_dir = current_app.config['UPLOAD_FOLDER']
        return os.path.join(base_dir, relative_path)

    @staticmethod
    def delete_file(relative_path):
        """
        Delete file from storage.

        Args:
            relative_path: Relative path from database

        Returns:
            bool: True if file was deleted, False if not found
        """
        absolute_path = FileStorage.get_file_path(relative_path)
        if os.path.exists(absolute_path):
            os.remove(absolute_path)
            return True
        return False
