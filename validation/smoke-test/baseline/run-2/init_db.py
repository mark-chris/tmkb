"""
Database initialization script.

Creates all database tables defined in models.py.
Optionally creates sample data for development/testing.

Usage:
    python init_db.py              # Create tables only
    python init_db.py --sample     # Create tables and sample data
"""

import sys
import os
from app import create_app, db
from app.models import Organization, User, File

def init_database(create_sample_data=False):
    """Initialize database with tables and optional sample data."""
    app = create_app()

    with app.app_context():
        # Drop all tables (careful in production!)
        print("Dropping all existing tables...")
        db.drop_all()

        # Create all tables
        print("Creating database tables...")
        db.create_all()
        print("Database tables created successfully!")

        if create_sample_data:
            print("\nCreating sample data...")

            # Create sample organization
            org = Organization(
                name="Acme Corporation",
                storage_quota_mb=1000
            )
            db.session.add(org)
            db.session.commit()
            print(f"Created organization: {org.name} (ID: {org.id})")

            # Create sample user
            user = User(
                email="admin@acme.com",
                organization_id=org.id,
                is_active=True
            )
            user.set_password("password123")
            db.session.add(user)
            db.session.commit()
            print(f"Created user: {user.email} (ID: {user.id})")
            print("  Password: password123")

            print("\nSample data created successfully!")
            print("\nYou can now login with:")
            print(f"  Email: {user.email}")
            print("  Password: password123")

        print("\nDatabase initialization complete!")

if __name__ == '__main__':
    # Check for --sample flag
    create_sample = '--sample' in sys.argv

    if create_sample:
        print("Initializing database with sample data...\n")
    else:
        print("Initializing database (no sample data)...\n")

    init_database(create_sample_data=create_sample)
