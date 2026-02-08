# TMKB Badges

## TMKB-Enhanced Badge

Projects that use TMKB to guide AI-generated code can display this badge to indicate their code was developed with architectural threat modeling context.

### Markdown

```markdown
[![TMKB-Enhanced](https://img.shields.io/badge/TMKB-Enhanced-blue?style=flat-square&logo=data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTYiIGhlaWdodD0iMTYiIHZpZXdCb3g9IjAgMCAxNiAxNiIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cGF0aCBkPSJNOCAxLjVMMi41IDQuNVYxMS41TDggMTQuNUwxMy41IDExLjVWNC41TDggMS41WiIgc3Ryb2tlPSJ3aGl0ZSIgc3Ryb2tlLXdpZHRoPSIxLjUiIGZpbGw9Im5vbmUiLz48cGF0aCBkPSJNOCA2VjEwTTYgOEgxMCIgc3Ryb2tlPSJ3aGl0ZSIgc3Ryb2tlLXdpZHRoPSIxLjUiLz48L3N2Zz4=)](https://github.com/mark-chris/tmkb)
```

### HTML

```html
<a href="https://github.com/mark-chris/tmkb">
  <img src="https://img.shields.io/badge/TMKB-Enhanced-blue?style=flat-square&logo=data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTYiIGhlaWdodD0iMTYiIHZpZXdCb3g9IjAgMCAxNiAxNiIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cGF0aCBkPSJNOCAxLjVMMi41IDQuNVYxMS41TDggMTQuNUwxMy41IDExLjVWNC41TDggMS41WiIgc3Ryb2tlPSJ3aGl0ZSIgc3Ryb2tlLXdpZHRoPSIxLjUiIGZpbGw9Im5vbmUiLz48cGF0aCBkPSJNOCA2VjEwTTYgOEgxMCIgc3Ryb2tlPSJ3aGl0ZSIgc3Ryb2tlLXdpZHRoPSIxLjUiLz48L3N2Zz4=" alt="TMKB-Enhanced" />
</a>
```

### Preview

[![TMKB-Enhanced](https://img.shields.io/badge/TMKB-Enhanced-blue?style=flat-square&logo=data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTYiIGhlaWdodD0iMTYiIHZpZXdCb3g9IjAgMCAxNiAxNiIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cGF0aCBkPSJNOCAxLjVMMi41IDQuNVYxMS41TDggMTQuNUwxMy41IDExLjVWNC41TDggMS41WiIgc3Ryb2tlPSJ3aGl0ZSIgc3Ryb2tlLXdpZHRoPSIxLjUiIGZpbGw9Im5vbmUiLz48cGF0aCBkPSJNOCA2VjEwTTYgOEgxMCIgc3Ryb2tlPSJ3aGl0ZSIgc3Ryb2tlLXdpZHRoPSIxLjUiLz48L3N2Zz4=)](https://github.com/mark-chris/tmkb)

---

## What Does "TMKB-Enhanced" Mean?

The TMKB-Enhanced badge indicates that:

1. **AI-generated code** was created with TMKB (Threat Model Knowledge Base) context
2. **Architectural security patterns** were considered during code generation
3. **Authorization boundaries** across async operations, multi-tenant contexts, and trust boundaries were explicitly addressed
4. **Code references TMKB patterns** in comments for traceability

### Example TMKB-Enhanced Code Characteristics

```python
@celery.task(bind=True)
def process_file(self, file_id, user_id, organization_id):
    """
    Process uploaded file with authorization re-validation.

    Security (TMKB-AUTHZ-001):
    - Re-validates ALL authorization checks from endpoint
    - Verifies tenant_id matches at every step
    - Does NOT trust authorization from original request
    """
    # ... code with 5 authorization checks
```

```python
class File(db.Model, TenantScopedMixin):
    """
    File model with tenant isolation.

    Security (TMKB-AUTHZ-004):
    - Automatic tenant filtering via TenantScopedMixin
    - Cannot query without authorization context
    """
```

---

## How to Earn This Badge

To use the TMKB-Enhanced badge:

1. **Configure TMKB MCP** in your AI coding assistant (Claude Code, etc.)
2. **Generate code** with TMKB context available
3. **Verify TMKB patterns** were applied (check for TMKB-AUTHZ-* references in code)
4. **Validate security** against relevant invariants
5. **Add the badge** to your README

### Validation Checklist

- [ ] Code includes TMKB pattern IDs in comments
- [ ] Background jobs re-validate authorization (TMKB-AUTHZ-001)
- [ ] Tenant isolation is enforced (TMKB-AUTHZ-004)
- [ ] Authorization checks are consistent across boundaries

---

## Alternative Styles

### Flat

```markdown
[![TMKB-Enhanced](https://img.shields.io/badge/TMKB-Enhanced-blue?style=flat&logo=shield)](https://github.com/mark-chris/tmkb)
```

[![TMKB-Enhanced](https://img.shields.io/badge/TMKB-Enhanced-blue?style=flat&logo=shield)](https://github.com/mark-chris/tmkb)

### For the Badge

```markdown
[![TMKB-Enhanced](https://img.shields.io/badge/TMKB-Enhanced-blue?style=for-the-badge&logo=shield)](https://github.com/mark-chris/tmkb)
```

[![TMKB-Enhanced](https://img.shields.io/badge/TMKB-Enhanced-blue?style=for-the-badge&logo=shield)](https://github.com/mark-chris/tmkb)

### Plastic

```markdown
[![TMKB-Enhanced](https://img.shields.io/badge/TMKB-Enhanced-blue?style=plastic&logo=shield)](https://github.com/mark-chris/tmkb)
```

[![TMKB-Enhanced](https://img.shields.io/badge/TMKB-Enhanced-blue?style=plastic&logo=shield)](https://github.com/mark-chris/tmkb)

---

## Badge Colors by Security Level

### High Confidence (All invariants pass)

```markdown
[![TMKB-Enhanced](https://img.shields.io/badge/TMKB-Enhanced-brightgreen?style=flat-square&logo=shield)](https://github.com/mark-chris/tmkb)
```

[![TMKB-Enhanced](https://img.shields.io/badge/TMKB-Enhanced-brightgreen?style=flat-square&logo=shield)](https://github.com/mark-chris/tmkb)

### Medium Confidence (Partial implementation)

```markdown
[![TMKB-Enhanced](https://img.shields.io/badge/TMKB-Partial-yellow?style=flat-square&logo=shield)](https://github.com/mark-chris/tmkb)
```

[![TMKB-Enhanced](https://img.shields.io/badge/TMKB-Partial-yellow?style=flat-square&logo=shield)](https://github.com/mark-chris/tmkb)

### In Progress

```markdown
[![TMKB](https://img.shields.io/badge/TMKB-In_Progress-orange?style=flat-square&logo=shield)](https://github.com/mark-chris/tmkb)
```

[![TMKB](https://img.shields.io/badge/TMKB-In_Progress-orange?style=flat-square&logo=shield)](https://github.com/mark-chris/tmkb)
