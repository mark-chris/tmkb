from flask import Blueprint

files_bp = Blueprint('files', __name__)

from app.files import routes
