from functools import wraps
from flask import abort
from flask_login import current_user
import os
import re

def require_org_access(model_instance):
    """Verify user has access to resource based on organization"""
    if not hasattr(model_instance, 'organization_id'):
        return True

    if model_instance.organization_id != current_user.organization_id:
        abort(404)  # Don't reveal existence

    return True

def sanitize_filename(filename):
    """Remove path traversal attempts and dangerous characters"""
    # Remove path separators
    filename = os.path.basename(filename)

    # Remove any non-alphanumeric characters except dots, dashes, underscores
    filename = re.sub(r'[^\w\s.-]', '', filename)

    # Limit length
    if len(filename) > 255:
        name, ext = os.path.splitext(filename)
        filename = name[:255-len(ext)] + ext

    return filename

def allowed_file(filename, allowed_extensions=None):
    """Check if file extension is allowed"""
    if allowed_extensions is None:
        # Allow all files by default (MIME validation happens in background)
        return '.' in filename

    return '.' in filename and \
           filename.rsplit('.', 1)[1].lower() in allowed_extensions
