# Protocol Document Modal E2E Tests

## Prerequisites

These tests require the full application stack to be running:

### 1. Start Backend API Server

```bash
# From project root
go run cmd/api/main.go
```

The backend should be running on `http://localhost:8080`

### 2. Start Frontend Dev Server

```bash
# From frontend directory
cd frontend
npm run dev
```

The frontend should be running on `http://localhost:5173`

### 3. Ensure Database is Seeded

The tests require at least one user (admin) to exist. If you haven't seeded the database:

```bash
# From project root
go run cmd/seed/main.go
```

This creates:
- Admin user: `admin` / `demo1234`
- Test groups and animals

## Running the Tests

Once both servers are running:

```bash
cd frontend
npx playwright test protocol-document-modal.spec.ts
```

### Run in UI Mode (Recommended for Development)

```bash
npx playwright test protocol-document-modal.spec.ts --ui
```

### Run in Headed Mode (Watch Browser)

```bash
npx playwright test protocol-document-modal.spec.ts --headed
```

## What the Tests Verify

When a protocol document is assigned to an animal:

1. **Authorization Header** - Ensures JWT token is sent to `/api/documents/:uuid`
2. **Response Validation** - Confirms 200 status and correct `Content-Type` (PDF/DOCX)
3. **Modal Rendering** - Verifies modal appears with document iframe using blob URL
4. **User Experience**:
   - Modal opens (not new tab)
   - Escape key closes modal
   - Click outside closes modal
   - Close button works
   - ARIA attributes present
   - Button is semantic `<button>` element

## Test Behavior

- If no backend is running: **Tests skip**
- If no animals exist: **Tests skip**  
- If animal has no protocol: **Test uploads minimal PDF fixture, then runs assertions**

## Troubleshooting

### All Tests Skip

**Cause**: Backend not running or login failing

**Solution**: 
1. Start backend: `go run cmd/api/main.go`
2. Verify it's running: `curl http://localhost:8080/api/health`
3. Seed database if needed: `go run cmd/seed/main.go`

### "Failed to upload protocol fixture" Error

**Cause**: API endpoint not accepting multipart upload

**Solution**: Verify backend routes are registered and user has permissions

### Timeout Errors

**Cause**: Slow response or frontend not running

**Solution**:
1. Ensure frontend dev server is running
2. Check network tab in headed mode for failed requests
