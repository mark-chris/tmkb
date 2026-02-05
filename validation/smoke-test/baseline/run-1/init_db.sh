#!/bin/bash
set -e

echo "Waiting for services to be ready..."
sleep 5

echo "Initializing database..."
flask db init || echo "Database already initialized"

echo "Creating migration..."
flask db migrate -m "Initial schema with multi-tenant support"

echo "Applying migrations..."
flask db upgrade

echo "Seeding initial data..."
flask seed-data

echo "Database setup complete!"
