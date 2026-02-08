"""Database setup script - creates initial organizations and users."""

from app import create_app
from app.extensions import db
from app.models.organization import Organization
from app.models.user import User


def setup_database():
    """Create initial organizations and users for testing."""
    app = create_app()

    with app.app_context():
        print("Creating database tables...")
        db.create_all()

        print("\nCreating organizations...")
        org1 = Organization(name='Acme Corp')
        org2 = Organization(name='TechStart Inc')
        db.session.add_all([org1, org2])
        db.session.commit()
        print(f"✓ Created organization: {org1.name} (ID: {org1.id})")
        print(f"✓ Created organization: {org2.name} (ID: {org2.id})")

        print("\nCreating users...")
        user1 = User(
            email='alice@acmecorp.com',
            organization_id=org1.id,
            is_active=True
        )
        user1.set_password('password123')

        user2 = User(
            email='bob@techstart.com',
            organization_id=org2.id,
            is_active=True
        )
        user2.set_password('password123')

        db.session.add_all([user1, user2])
        db.session.commit()

        print(f"✓ Created user: {user1.email} (Org: {org1.name})")
        print(f"✓ Created user: {user2.email} (Org: {org2.name})")

        print("\n" + "="*60)
        print("Database setup complete!")
        print("="*60)
        print("\nLogin credentials:")
        print(f"  User 1: {user1.email} / password123")
        print(f"  User 2: {user2.email} / password123")
        print("\nYou can now start the application with:")
        print("  python wsgi.py")


if __name__ == '__main__':
    setup_database()
