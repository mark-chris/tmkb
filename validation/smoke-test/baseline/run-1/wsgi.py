from app import create_app, db
from app.models import Organization, User, File
import os

app = create_app()

# Make shell context for flask shell
@app.shell_context_processor
def make_shell_context():
    return {
        'db': db,
        'Organization': Organization,
        'User': User,
        'File': File
    }

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000, debug=True)
