# TMKB Validation: Visual Architecture Comparison

**Baseline (Without TMKB) vs Enhanced (With TMKB)**

---

## Background Job Authorization Flow

### ❌ Baseline: Authorization Context Lost

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ POST /files + file
       │ Authorization: Bearer <token>
       ▼
┌─────────────────────────────────────────────────┐
│  Upload Endpoint                                │
│                                                 │
│  @login_required ✓                              │
│  organization_id = current_user.organization_id │
│                                                 │
│  file_record = File(                            │
│    organization_id=organization_id,  ✓          │
│    uploaded_by=current_user.id       ✓          │
│  )                                              │
│  db.session.add(file_record)                    │
└──────┬──────────────────────────────────────────┘
       │
       │ Queue background job
       │ process_file.delay(file_id)  ⚠️ ONLY file_id
       │
       ▼
┌─────────────────────────────────────────────────┐
│  Redis Queue                                    │
│  {"task": "process_file", "args": [123]}        │
└──────┬──────────────────────────────────────────┘
       │
       │ ⚠️ Authorization context LOST
       │
       ▼
┌─────────────────────────────────────────────────┐
│  Celery Worker                                  │
│                                                 │
│  def process_file(self, file_id):               │
│      file = File.query.get(file_id)  ❌         │
│                                                 │
│      # NO tenant check                          │
│      # NO user validation                       │
│      # NO permission re-check                   │
│                                                 │
│      # Process any file by ID                   │
│      process(file)                              │
│                                                 │
└─────────────────────────────────────────────────┘

VULNERABILITY: Task can process files from ANY tenant
               if attacker can inject task into queue
```

---

### ✅ Enhanced: Authorization Context Preserved

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ POST /files + file
       │ Authorization: Bearer <token>
       ▼
┌─────────────────────────────────────────────────┐
│  Upload Endpoint                                │
│                                                 │
│  @login_required ✓                              │
│  organization_id = current_user.organization_id │
│                                                 │
│  file_record = File(                            │
│    organization_id=organization_id,  ✓          │
│    uploaded_by=current_user.id       ✓          │
│  )                                              │
│  db.session.add(file_record)                    │
└──────┬──────────────────────────────────────────┘
       │
       │ Queue background job with FULL context
       │ process_file_task.delay(
       │   file_id=file_record.id,
       │   user_id=current_user.id,           ✓
       │   organization_id=organization_id    ✓
       │ )
       ▼
┌─────────────────────────────────────────────────┐
│  Redis Queue                                    │
│  {"task": "process_file_task",                  │
│   "args": [123, 456, 789]}  ✓ Full context      │
└──────┬──────────────────────────────────────────┘
       │
       │ ✅ Authorization context PRESERVED
       │
       ▼
┌─────────────────────────────────────────────────┐
│  Celery Worker                                  │
│                                                 │
│  def process_file_task(self, file_id,           │
│                        user_id,                 │
│                        organization_id):        │
│                                                 │
│    # ✅ CHECK 1: Load with tenant filter        │
│    file = File.get_for_tenant(                  │
│      file_id, tenant_id=organization_id         │
│    )                                            │
│                                                 │
│    # ✅ CHECK 2: Tenant match                   │
│    if file.organization_id != organization_id:  │
│        raise AuthorizationError()               │
│                                                 │
│    # ✅ CHECK 3: User still valid               │
│    user = User.query.get(user_id)               │
│    if user.organization_id != organization_id:  │
│        raise AuthorizationError()               │
│                                                 │
│    # ✅ CHECK 4: File not deleted               │
│    if file.deleted_at:                          │
│        raise AuthorizationError()               │
│                                                 │
│    # ✅ CHECK 5: Uploader match                 │
│    if file.uploaded_by_user_id != user_id:      │
│        raise AuthorizationError()               │
│                                                 │
│    # All checks passed - safe to process        │
│    process(file)                                │
│                                                 │
└─────────────────────────────────────────────────┘

SECURE: Task validates ALL authorization
        before processing
```

---

## Tenant Isolation Architecture

### ❌ Baseline: Manual Filtering (Fragile)

```
┌────────────────────────────────────────────────────┐
│  File Endpoints                                    │
├────────────────────────────────────────────────────┤
│                                                    │
│  def list_files():                                 │
│      files = File.query.filter_by(                 │
│        organization_id=current_user.organization_id│  ⚠️ Manual filter
│      ).all()                                       │     (easy to forget)
│                                                    │
│  def get_file(file_id):                            │
│      file = File.query.get_or_404(file_id)         │  ⚠️ Load first
│      require_org_access(file)                      │     Check second
│                                                    │     (TOCTOU risk)
│                                                    │
│  def download_file(file_id):                       │
│      file = File.query.get_or_404(file_id)         │  ⚠️ Must remember
│      require_org_access(file)                      │     check on EVERY
│      send_file(file.path)                          │     endpoint
│                                                    │
└────────────────────────────────────────────────────┘
         │
         │ No architectural guarantee
         ▼
┌────────────────────────────────────────────────────┐
│  Database                                          │
│                                                    │
│  SELECT * FROM files WHERE id = ?                  │  ⚠️ No filter
│  SELECT * FROM files WHERE organization_id = ?     │  ✓ Filter (if remembered)
│                                                    │
└────────────────────────────────────────────────────┘

RISK: Future endpoints may forget to add filter
      Two-step pattern allows time-of-check/time-of-use race
```

---

### ✅ Enhanced: Automatic Filtering (Architectural Guarantee)

```
┌────────────────────────────────────────────────────┐
│  TenantScopedMixin (Base Class)                    │
├────────────────────────────────────────────────────┤
│                                                    │
│  @classmethod                                      │
│  def tenant_query(cls):                            │
│      if not current_user.is_authenticated:         │
│          raise RuntimeError("No auth context")     │
│                                                    │
│      query = cls.query.filter_by(                  │
│          organization_id=current_user.org_id       │  ✅ Always filtered
│      )                                             │
│                                                    │
│      if hasattr(cls, 'deleted_at'):                │
│          query = query.filter(                     │  ✅ Auto-exclude
│              cls.deleted_at.is_(None)              │     soft-deleted
│          )                                         │
│      return query                                  │
│                                                    │
│  @classmethod                                      │
│  def get_for_tenant(cls, id, tenant_id=None):      │
│      if tenant_id is None:                         │
│          query = cls.tenant_query()                │  ✅ Request context
│      else:                                         │
│          query = cls.query.filter_by(              │  ✅ Background job
│              organization_id=tenant_id             │     (explicit)
│          )                                         │
│      return query.filter_by(id=id).first_or_404()  │
│                                                    │
└──────┬─────────────────────────────────────────────┘
       │
       │ Inherited by all tenant-scoped models
       ▼
┌────────────────────────────────────────────────────┐
│  File Model                                        │
├────────────────────────────────────────────────────┤
│  class File(db.Model, TenantScopedMixin):          │
│      organization_id = db.Column(...)              │
│      # ... other fields                            │
└──────┬─────────────────────────────────────────────┘
       │
       │ All queries use mixin methods
       ▼
┌────────────────────────────────────────────────────┐
│  File Endpoints                                    │
├────────────────────────────────────────────────────┤
│                                                    │
│  def list_files():                                 │
│      files = File.tenant_query().all()             │  ✅ Automatic filter
│                                                    │     (can't forget)
│  def get_file(file_id):                            │
│      file = File.get_for_tenant(file_id)           │  ✅ Atomic query
│      return jsonify(file.to_dict())                │     (no TOCTOU)
│                                                    │
│  def download_file(file_id):                       │
│      file = File.get_for_tenant(file_id)           │  ✅ Consistent
│      send_file(file.path)                          │     everywhere
│                                                    │
└────────────────────────────────────────────────────┘
         │
         │ Architectural guarantee
         ▼
┌────────────────────────────────────────────────────┐
│  Database                                          │
│                                                    │
│  SELECT * FROM files                               │  ✅ Always includes
│  WHERE organization_id = ?                         │     tenant filter
│  AND deleted_at IS NULL                            │     and soft-delete
│  AND id = ?                                        │     exclusion
│                                                    │
└────────────────────────────────────────────────────┘

SECURE: Impossible to forget tenant filter
        Single filtered query (no TOCTOU)
        Centralized security logic
```

---

## Code Comparison: Side by Side

### Task Signature

```
┌─────────────────────────────────┬─────────────────────────────────┐
│  Baseline                       │  Enhanced                       │
├─────────────────────────────────┼─────────────────────────────────┤
│                                 │                                 │
│  @celery.task(bind=True)        │  @celery.task(bind=True)        │
│  def process_file(              │  def process_file_task(         │
│      self,                      │      self,                      │
│      file_id  # ❌ Only ID      │      file_id,                   │
│  ):                             │      user_id,      # ✅ User    │
│      pass                       │      organization_id # ✅ Org   │
│                                 │  ):                             │
│                                 │      """                        │
│                                 │      Security (TMKB-AUTHZ-001)  │
│                                 │      Re-validates authorization │
│                                 │      """                        │
│                                 │      pass                       │
└─────────────────────────────────┴─────────────────────────────────┘
```

### Task Implementation

```
┌─────────────────────────────────┬─────────────────────────────────┐
│  Baseline                       │  Enhanced                       │
├─────────────────────────────────┼─────────────────────────────────┤
│                                 │                                 │
│  # Load file without checks     │  # CHECK 1: Tenant filter       │
│  file = File.query.get(         │  file = File.get_for_tenant(    │
│      file_id                    │      file_id,                   │
│  )  # ❌ No tenant filter        │      tenant_id=organization_id  │
│                                 │  )  # ✅ Filtered load           │
│                                 │                                 │
│  # Process immediately          │  # CHECK 2: Tenant match        │
│  process(file)                  │  if file.organization_id !=     │
│                                 │     organization_id:            │
│                                 │      raise AuthorizationError() │
│                                 │                                 │
│                                 │  # CHECK 3: User validation     │
│                                 │  user = User.query.get(user_id) │
│                                 │  if user.organization_id !=     │
│                                 │     organization_id:            │
│                                 │      raise AuthorizationError() │
│                                 │                                 │
│                                 │  # CHECK 4: Not deleted         │
│                                 │  if file.deleted_at:            │
│                                 │      raise AuthorizationError() │
│                                 │                                 │
│                                 │  # CHECK 5: Uploader match      │
│                                 │  if file.uploaded_by_user_id != │
│                                 │     user_id:                    │
│                                 │      raise AuthorizationError() │
│                                 │                                 │
│                                 │  # All checks passed            │
│                                 │  process(file)                  │
│                                 │                                 │
└─────────────────────────────────┴─────────────────────────────────┘
```

### Query Pattern

```
┌─────────────────────────────────┬─────────────────────────────────┐
│  Baseline                       │  Enhanced                       │
├─────────────────────────────────┼─────────────────────────────────┤
│                                 │                                 │
│  # List files                   │  # List files                   │
│  files = File.query.filter_by(  │  files = File.tenant_query()    │
│      organization_id=           │      .all()                     │
│          current_user.org_id    │  # ✅ Automatic filter           │
│  ).all()                        │                                 │
│  # ⚠️ Manual filter              │                                 │
│                                 │                                 │
│  # Get single file              │  # Get single file              │
│  file = File.query.get_or_404(  │  file = File.get_for_tenant(    │
│      file_id                    │      file_id                    │
│  )  # ⚠️ Load first              │  )  # ✅ Atomic load + filter    │
│  require_org_access(file)       │                                 │
│  # ⚠️ Check second               │                                 │
│                                 │                                 │
└─────────────────────────────────┴─────────────────────────────────┘
```

---

## Security Metrics Comparison

```
┌──────────────────────────────────────────────────────────────────┐
│  Metric                          Baseline    Enhanced    Delta   │
├──────────────────────────────────────────────────────────────────┤
│  Authorization checks in task        0           5        +5     │
│  Tenant isolation method         Manual   Automatic              │
│  Query pattern                Two-step      Atomic                │
│  TMKB references in code             0           6        +6     │
│  Security test files                 0           1        +1     │
│  Soft-delete protection             No         Yes               │
│  Defense-in-depth layers             1           5        +4     │
└──────────────────────────────────────────────────────────────────┘
```

---

## Attack Scenarios

### Scenario 1: Queue Injection Attack

**Baseline:**
```
Attacker injects task into Redis:
  {"task": "process_file", "args": [victim_file_id]}

Result: ❌ VULNERABLE
  Worker loads file without tenant check
  Processes victim's file
```

**Enhanced:**
```
Attacker injects task into Redis:
  {"task": "process_file_task",
   "args": [victim_file_id, attacker_user_id, attacker_org_id]}

Result: ✅ PROTECTED
  Worker loads with tenant filter
  File not found for attacker's org
  Task fails safely
```

---

### Scenario 2: User Moves Organizations

**Baseline:**
```
1. User uploads file to Org A
2. Admin moves user to Org B
3. Background job runs

Result: ❌ VULNERABLE
  Worker processes file from old org
  User no longer has permission
```

**Enhanced:**
```
1. User uploads file to Org A
2. Admin moves user to Org B
3. Background job runs

Result: ✅ PROTECTED
  CHECK 3 fails: user.organization_id != file.organization_id
  AuthorizationError raised
  Task fails safely
```

---

### Scenario 3: Soft-Delete Resurrection

**Baseline:**
```
1. User deletes file
2. Background job still in queue
3. Job processes deleted file

Result: ⚪ N/A (no soft-delete)
```

**Enhanced:**
```
1. User deletes file (soft-delete)
2. Background job still in queue
3. Job processes deleted file

Result: ✅ PROTECTED
  CHECK 4 fails: file.deleted_at is not None
  AuthorizationError raised
  Task fails safely
```

---

## Architecture Evolution

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  Baseline Architecture                                          │
│  ────────────────────                                           │
│                                                                 │
│   Endpoints                                                     │
│      ↓                                                          │
│   Manual Filters (per endpoint)                                 │
│      ↓                                                          │
│   Database                                                      │
│                                                                 │
│  Weakness: Each endpoint must remember to filter                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

                              ↓ TMKB Guidance

┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  Enhanced Architecture                                          │
│  ─────────────────────                                          │
│                                                                 │
│   Endpoints                                                     │
│      ↓                                                          │
│   TenantScopedMixin (automatic filtering)                       │
│      ↓                                                          │
│   Database                                                      │
│                                                                 │
│  Strength: Architectural guarantee prevents future mistakes     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Summary: The TMKB Difference

| Aspect | Baseline | Enhanced | Improvement |
|--------|----------|----------|-------------|
| **Background job auth** | 0 checks | 5 checks | Critical security fix |
| **Tenant isolation** | Manual (fragile) | Automatic (robust) | Architectural |
| **Query pattern** | Load then check | Filter then load | Structural |
| **Security documentation** | Minimal | TMKB-referenced | Traceable |
| **Test coverage** | Functional only | Security-focused | Comprehensive |
| **Defense layers** | Single | Multiple | Defense-in-depth |

**The Bottom Line:**

Without TMKB: Code works but has authorization gaps at trust boundaries

With TMKB: Code is secure by design with architectural guarantees
