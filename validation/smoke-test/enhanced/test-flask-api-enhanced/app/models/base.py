"""Base model classes with tenant isolation security."""

from flask_login import current_user
from flask import has_request_context
from app.extensions import db


class TenantScopedMixin:
    """
    Mixin providing automatic tenant isolation.

    SECURITY GUARANTEES (addresses TMKB-AUTHZ-004):
    - All queries must go through tenant_query() or get_for_tenant()
    - Background jobs MUST pass explicit tenant_id
    - Automatic filtering prevents cross-tenant data access

    Usage:
        # In request context (endpoints)
        files = File.tenant_query().all()
        file = File.get_for_tenant(file_id)

        # In background jobs
        file = File.get_for_tenant(file_id, tenant_id=organization_id)
    """

    # Every tenant-scoped model has organization_id
    organization_id = db.Column(
        db.Integer,
        db.ForeignKey('organizations.id'),
        nullable=False,
        index=True
    )

    @classmethod
    def tenant_query(cls):
        """
        Returns query automatically filtered to current user's tenant.

        Raises:
            RuntimeError: If no request context or unauthenticated user

        Returns:
            Query filtered by current_user.organization_id
        """
        if not has_request_context():
            raise RuntimeError(
                f"Cannot query {cls.__name__} without request context. "
                "Background jobs must use get_for_tenant(id, tenant_id=...)"
            )

        if not current_user or not current_user.is_authenticated:
            raise RuntimeError(
                f"Cannot query {cls.__name__} without authenticated user"
            )

        query = cls.query.filter_by(
            organization_id=current_user.organization_id
        )

        # Exclude soft-deleted records if model supports it
        if hasattr(cls, 'deleted_at'):
            query = query.filter(cls.deleted_at.is_(None))

        return query

    @classmethod
    def get_for_tenant(cls, id, tenant_id=None):
        """
        Get record by ID with automatic tenant verification.

        Args:
            id: Record ID
            tenant_id: Explicit tenant ID (required for background jobs)

        Returns:
            Record or raises 404

        Raises:
            RuntimeError: If tenant_id not provided and no request context
        """
        if tenant_id is None:
            # Request context - use current user's tenant
            query = cls.tenant_query()
        else:
            # Background job context - use explicit tenant_id
            query = cls.query.filter_by(organization_id=tenant_id)
            if hasattr(cls, 'deleted_at'):
                query = query.filter(cls.deleted_at.is_(None))

        record = query.filter_by(id=id).first()
        if not record:
            from werkzeug.exceptions import NotFound
            raise NotFound(f"{cls.__name__} not found")

        return record

    @classmethod
    def all_for_tenant(cls, tenant_id=None):
        """
        Get all records for a tenant.

        Args:
            tenant_id: Explicit tenant ID (optional, uses current_user if None)

        Returns:
            List of records
        """
        if tenant_id is None:
            return cls.tenant_query().all()
        else:
            query = cls.query.filter_by(organization_id=tenant_id)
            if hasattr(cls, 'deleted_at'):
                query = query.filter(cls.deleted_at.is_(None))
            return query.all()
