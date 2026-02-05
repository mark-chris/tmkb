# API Documentation

Base URL: `http://localhost:5000`

## Authentication

All file endpoints require authentication via session cookie.

### POST /auth/login

Authenticate user and create session.

**Request:**
```json
{
  "username": "admin",
  "password": "password"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "user": {
    "id": 1,
    "username": "admin",
    "organization_id": 1
  }
}
```

**Errors:**
- `400` - Missing username or password
- `401` - Invalid credentials

**Rate Limit:** 5 requests per minute

---

### POST /auth/logout

Clear user session.

**Response (200 OK):**
```json
{
  "success": true
}
```

**Errors:**
- `401` - Not authenticated

---

### GET /auth/me

Get current authenticated user.

**Response (200 OK):**
```json
{
  "user": {
    "id": 1,
    "username": "admin",
    "organization_id": 1
  }
}
```

**Errors:**
- `401` - Not authenticated

---

## File Operations

### POST /files

Upload a file for background processing.

**Request:**
- Content-Type: `multipart/form-data`
- Field: `file` (binary file data)

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "document.pdf",
  "status": "pending",
  "file_size": null,
  "mime_type": null,
  "sha256_hash": null,
  "created_at": "2026-02-03T19:30:00.000000",
  "processed_at": null
}
```

**Errors:**
- `400` - No file provided or invalid file
- `401` - Not authenticated
- `413` - File too large (>50MB)

**Rate Limit:** 10 requests per minute

---

### GET /files

List files for authenticated user's organization.

**Query Parameters:**
- `limit` (optional) - Results per page (default: 20, max: 100)
- `offset` (optional) - Number of results to skip (default: 0)

**Response (200 OK):**
```json
{
  "files": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "filename": "document.pdf",
      "status": "completed",
      "file_size": 1024000,
      "mime_type": "application/pdf",
      "sha256_hash": "abc123...",
      "created_at": "2026-02-03T19:30:00.000000",
      "processed_at": "2026-02-03T19:30:05.000000"
    }
  ],
  "total": 150,
  "limit": 20,
  "offset": 0
}
```

**Errors:**
- `400` - Invalid pagination parameters
- `401` - Not authenticated

**Notes:**
- Results sorted by `created_at` descending (newest first)
- Only shows files belonging to user's organization

---

### GET /files/<file_id>

Get details for a specific file.

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "document.pdf",
  "status": "completed",
  "file_size": 1024000,
  "mime_type": "application/pdf",
  "sha256_hash": "abc123...",
  "created_at": "2026-02-03T19:30:00.000000",
  "processed_at": "2026-02-03T19:30:05.000000",
  "uploaded_by": {
    "id": 1,
    "username": "admin"
  }
}
```

**Errors:**
- `401` - Not authenticated
- `404` - File not found or belongs to different organization

---

## File Status Values

- `pending` - File uploaded, waiting for processing
- `processing` - Background job is processing file
- `completed` - File processed successfully
- `failed` - Processing failed (see `error_message`)

---

## Error Responses

All errors follow this format:

```json
{
  "error": "Human-readable error message"
}
```

Common HTTP status codes:
- `400` - Bad Request (invalid input)
- `401` - Unauthorized (not authenticated)
- `404` - Not Found
- `413` - Payload Too Large (file >50MB)
- `429` - Too Many Requests (rate limited)
- `500` - Internal Server Error
