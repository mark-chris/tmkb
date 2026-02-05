import click
from flask.cli import with_appcontext
from app import db
from app.models import Organization, User

@click.command('seed-data')
@with_appcontext
def seed_data():
    """Seed database with initial test data"""
    # Create organization
    org = Organization.query.filter_by(name='Acme Corp').first()
    if not org:
        org = Organization(name='Acme Corp')
        db.session.add(org)
        db.session.flush()

    # Create user
    user = User.query.filter_by(username='admin').first()
    if not user:
        user = User(username='admin', organization_id=org.id)
        user.set_password('password')
        db.session.add(user)

    db.session.commit()

    click.echo(f'Created organization: {org.name} (ID: {org.id})')
    click.echo(f'Created user: {user.username} (password: password)')
    click.echo('Seed data created successfully!')

def init_app(app):
    """Register CLI commands"""
    app.cli.add_command(seed_data)
