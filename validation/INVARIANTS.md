# Authorization Invariants

These are the **non-negotiable** checks for validating authorization enforcement in multi-tenant applications.

## The Four Invariants

### INV-1: Authorization Check on Every Mutating Endpoint

Every endpoint that creates, updates, or deletes resources MUST verify the requesting user has permission to perform that action.

**What to look for:**
- `@login_required` or equivalent decorator on POST/PUT/PATCH/DELETE endpoints
- Explicit permission checks before database mutations
- User context used when creating resources (not just accepting client-provided IDs)

**Pass criteria:**
- Every mutating endpoint verifies the user can perform the action
- New resources are associated with the authenticated user's context

**Common failures:**
- Missing `@login_required` on admin endpoints
- Accepting `user_id` or `organization_id` from request body instead of session
- Trusting client-provided ownership claims

---

### INV-2: Object Ownership Validated Server-Side

Before accessing any resource, the server MUST verify the requesting user has permission to access that specific resource. Client-provided IDs cannot be trusted.

**What to look for:**
- Database queries that filter by user/organization ownership
- Explicit ownership checks after loading resources by ID
- 404 (not 403) returned for unauthorized access attempts

**Pass criteria:**
- Resources are loaded with ownership filters, OR
- Resources are loaded by ID then ownership is verified before use
- Unauthorized access returns 404 to prevent enumeration

**Common failures:**
- `Resource.query.get(id)` without subsequent ownership check
- Ownership check only on some endpoints (list vs. detail)
- Returning 403 which reveals resource existence

---

### INV-3: List/Detail Authorization Consistency

A resource visible in a list response MUST be accessible via its detail endpoint, and vice versa. Authorization logic must be consistent.

**What to look for:**
- Same ownership filter in list and detail queries
- Shared authorization helper used across endpoints
- No fields leaked in list that aren't available in detail

**Pass criteria:**
- `list_resources()` and `get_resource(id)` use identical authorization logic
- If a user can see a resource in the list, they can access its detail
- If a user can't access a detail endpoint, the resource doesn't appear in lists

**Common failures:**
- List filtered by `organization_id` but detail doesn't check
- Detail checks `uploaded_by` but list only checks `organization_id`
- Sensitive fields included in list response that aren't meant to be visible

---

### INV-4: Background Jobs Re-Validate Authorization

When work is deferred to background processing, the job MUST re-validate authorization before operating on resources. The endpoint's authorization check is not sufficient.

**What to look for:**
- Job receives `user_id` and `organization_id`, not just resource ID
- Job loads user and verifies they still have access
- Job checks resource still belongs to claimed organization

**Pass criteria:**
- Job signature includes authorization context (user_id, organization_id)
- Job re-validates user exists and has permission
- Job verifies resource ownership before processing

**Common failures:**
- Job accepts only `resource_id` parameter
- Job loads resource directly without ownership check
- Job assumes endpoint authorization is sufficient
- No user context available in worker process

---

## Using Invariants for Validation

### Baseline Test Protocol

1. Generate code with identical prompt (no security hints)
2. Run twice to confirm consistency
3. Analyze against all four invariants
4. Document pass/fail for each

### Success Criteria

- **Baseline (without TMKB):** Should fail ≥2 invariants (if LLMs were good at this, TMKB wouldn't be needed)
- **Enhanced (with TMKB):** Should pass all 4 invariants

### Expected LLM Behavior

| Invariant | Expected Baseline | Why |
|-----------|-------------------|-----|
| INV-1 | Usually passes | LLMs know about `@login_required` |
| INV-2 | Usually passes | LLMs know about IDOR |
| INV-3 | Sometimes fails | Requires cross-endpoint reasoning |
| INV-4 | **Usually fails** | Requires async boundary reasoning |

INV-4 is the most reliable indicator of TMKB value because it requires understanding that async execution is a trust boundary—something LLMs consistently miss.
