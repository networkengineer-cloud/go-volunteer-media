# Document PDF Conversion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert DOCX and XLSX uploads to PDF at upload time using LibreOffice so all group documents render inline in the browser; reject uploads where conversion fails.

**Architecture:** A new `internal/convert` package exposes a `Converter` interface with a `LibreOfficeConverter` implementation that shells out to `libreoffice --headless`. `UploadGroupDocument` accepts `convert.Converter` as a dependency and converts non-PDF files before storage. Conversion failure returns HTTP 422. The Dockerfile switches from `alpine` to `debian:bookworm-slim` and installs LibreOffice.

**Tech Stack:** Go `os/exec`, LibreOffice CLI, `archive/zip` (test helper), Debian Bookworm Slim base image.

---

## File Map

| Action | Path | Purpose |
|--------|------|---------|
| Create | `internal/convert/convert.go` | `Converter` interface + `LibreOfficeConverter` |
| Create | `internal/convert/convert_test.go` | Integration tests (skipped if LibreOffice not on PATH) |
| Modify | `internal/handlers/test_helpers.go` | Add `mockConverter` |
| Modify | `internal/handlers/group_document.go` | Accept `convert.Converter`, insert conversion step |
| Modify | `internal/handlers/group_document_test.go` | Add conversion test cases, wire converter into existing tests |
| Modify | `cmd/api/main.go` | Instantiate `LibreOfficeConverter`, pass to handler, raise `WriteTimeout` |
| Modify | `Dockerfile` | Debian base, install LibreOffice, set `HOME=/tmp` |
| Modify | `frontend/src/pages/GroupPage.tsx` | Update file input label |

---

### Task 1: Create `internal/convert` package

**Files:**
- Create: `internal/convert/convert.go`
- Create: `internal/convert/convert_test.go`

- [ ] **Step 1: Write the failing integration test**

Create `internal/convert/convert_test.go`:

```go
package convert_test

import (
	"archive/zip"
	"bytes"
	"context"
	"os/exec"
	"testing"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/convert"
)

func skipIfNoLibreOffice(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("libreoffice"); err != nil {
		t.Skip("libreoffice not in PATH, skipping integration test")
	}
}

// minimalDOCXBytes builds the smallest valid DOCX (an OOXML ZIP) that LibreOffice accepts.
func minimalDOCXBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body><w:p><w:r><w:t>Test</w:t></w:r></w:p><w:sectPr/></w:body>
</w:document>`,
		"word/_rels/document.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`,
	}
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func TestLibreOfficeConverter_ToPDF_ValidDOCX(t *testing.T) {
	skipIfNoLibreOffice(t)
	c := &convert.LibreOfficeConverter{}
	pdf, err := c.ToPDF(context.Background(), minimalDOCXBytes(t), ".docx")
	if err != nil {
		t.Fatalf("ToPDF returned error: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		n := len(pdf)
		if n > 8 {
			n = 8
		}
		t.Errorf("expected PDF output starting with %%PDF, got: %q", pdf[:n])
	}
}

func TestLibreOfficeConverter_ToPDF_InvalidInput(t *testing.T) {
	skipIfNoLibreOffice(t)
	c := &convert.LibreOfficeConverter{}
	_, err := c.ToPDF(context.Background(), []byte("this is not a valid docx file"), ".docx")
	if err == nil {
		t.Error("expected error for invalid input, got nil")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./internal/convert/... -v -run TestLibreOfficeConverter 2>&1 | head -10
```

Expected: `cannot find package "...internal/convert"` — the package does not exist yet.

- [ ] **Step 3: Create `internal/convert/convert.go`**

```go
package convert

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Converter converts a document file to PDF bytes.
type Converter interface {
	// ToPDF converts data (a DOCX or XLSX file identified by ext, e.g. ".docx") to PDF.
	// Returns an error if conversion fails or times out.
	ToPDF(ctx context.Context, data []byte, ext string) ([]byte, error)
}

// LibreOfficeConverter converts documents to PDF using a locally installed LibreOffice.
// LibreOffice must be on PATH. The container must set HOME=/tmp so LibreOffice can write
// its user profile when running as a non-root user.
type LibreOfficeConverter struct{}

func (c *LibreOfficeConverter) ToPDF(ctx context.Context, data []byte, ext string) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "doc-convert-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input"+ext)
	if err := os.WriteFile(inputPath, data, 0600); err != nil {
		return nil, fmt.Errorf("write input file: %w", err)
	}

	// Cap conversion at 60 seconds to prevent requests hanging indefinitely.
	convCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(convCtx,
		"libreoffice", "--headless",
		"--convert-to", "pdf",
		"--outdir", tmpDir,
		inputPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("libreoffice failed: %w\noutput: %s", err, out)
	}

	outputPath := filepath.Join(tmpDir, "input.pdf")
	pdf, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read converted PDF: %w", err)
	}
	if len(pdf) == 0 {
		return nil, fmt.Errorf("libreoffice produced an empty PDF")
	}
	return pdf, nil
}
```

- [ ] **Step 4: Run the test again**

```bash
go test ./internal/convert/... -v -run TestLibreOfficeConverter 2>&1
```

Expected: tests PASS (or SKIP if LibreOffice is not installed on the dev machine). No compilation errors in either case.

- [ ] **Step 5: Commit**

```bash
git add internal/convert/convert.go internal/convert/convert_test.go
git commit -m "feat: add LibreOffice PDF conversion package"
```

---

### Task 2: Add `mockConverter` and write handler tests for conversion paths

**Files:**
- Modify: `internal/handlers/test_helpers.go`
- Modify: `internal/handlers/group_document_test.go`

- [ ] **Step 1: Add `mockConverter` to `internal/handlers/test_helpers.go`**

Add this block after the `mockStorageProvider` block (after line 156):

```go
// mockConverter is a test double for convert.Converter.
// Set ConvertErr to simulate a conversion failure.
// When ConvertErr is nil, ToPDF returns minimalPDF bytes.
type mockConverter struct {
	ConvertErr error
}

func (m *mockConverter) ToPDF(_ context.Context, _ []byte, _ string) ([]byte, error) {
	if m.ConvertErr != nil {
		return nil, m.ConvertErr
	}
	return minimalPDF, nil
}
```

- [ ] **Step 2: Add a `converterOverride` field to `TestUploadGroupDocument` and new test cases**

In `internal/handlers/group_document_test.go`, update the `TestUploadGroupDocument` test struct to add `converterOverride`:

```go
func TestUploadGroupDocument(t *testing.T) {
	tests := []struct {
		name              string
		setupUser         func(db *gorm.DB, group *models.Group) (userID uint, isAdmin bool)
		fields            map[string]string
		fileFieldName     string
		filename          string
		fileContent       []byte
		converterOverride *mockConverter // nil = use default &mockConverter{}
		expectedStatus    int
	}{
```

Add these three new test cases to the existing slice (after `"title_too_short_(<2_chars)_returns_400"`):

```go
		{
			name: "DOCX is converted to PDF on upload (201)",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "convadmin", Email: "convadmin@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, true)
				return u.ID, false
			},
			fields:        map[string]string{"title": "Converted Doc"},
			fileFieldName: "file",
			filename:      "report.docx",
			// ZIP magic bytes — passes ValidateDocumentUpload for .docx
			fileContent:    append([]byte{0x50, 0x4B, 0x03, 0x04}, make([]byte, 60)...),
			expectedStatus: http.StatusCreated,
		},
		{
			name: "conversion failure returns 422",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "convfailadmin", Email: "convfail@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, true)
				return u.ID, false
			},
			fields:        map[string]string{"title": "Bad Doc"},
			fileFieldName: "file",
			filename:      "broken.docx",
			fileContent:   append([]byte{0x50, 0x4B, 0x03, 0x04}, make([]byte, 60)...),
			converterOverride: &mockConverter{
				ConvertErr: fmt.Errorf("libreoffice conversion failed"),
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "PDF upload skips conversion even when converter would fail",
			setupUser: func(db *gorm.DB, group *models.Group) (uint, bool) {
				u := &models.User{Username: "pdfskipadmin", Email: "pdfskip@test.com", Password: "x"}
				db.Create(u)
				addUserToGroupForDocTest(db, u.ID, group.ID, true)
				return u.ID, false
			},
			fields:        map[string]string{"title": "PDF Doc"},
			fileFieldName: "file",
			filename:      "direct.pdf",
			fileContent:   minimalPDF,
			// ConvertErr is set — if the handler calls the converter for a PDF the upload
			// would return 422. Getting 201 proves the converter was NOT called.
			converterOverride: &mockConverter{ConvertErr: fmt.Errorf("should not be called")},
			expectedStatus:    http.StatusCreated,
		},
```

Update the test loop body to resolve the converter and pass it to the handler. Replace:

```go
			storageMock := &mockStorageProvider{}
			req := buildDocumentMultipartRequest(t, tt.fields, tt.fileFieldName, tt.filename, tt.fileContent)

			c, w := newGroupDocTestContext(userID, isAdmin)
			c.Request = req
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}

			UploadGroupDocument(db, storageMock)(c)
```

With:

```go
			storageMock := &mockStorageProvider{}
			conv := tt.converterOverride
			if conv == nil {
				conv = &mockConverter{}
			}
			req := buildDocumentMultipartRequest(t, tt.fields, tt.fileFieldName, tt.filename, tt.fileContent)

			c, w := newGroupDocTestContext(userID, isAdmin)
			c.Request = req
			c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", group.ID)}}

			UploadGroupDocument(db, storageMock, conv)(c)
```

Also update `TestUploadGroupDocument_PostgresFallback` — replace:

```go
			UploadGroupDocument(db, storageMock)(c)
```

With:

```go
			UploadGroupDocument(db, storageMock, &mockConverter{})(c)
```

Also add `"fmt"` to the test file imports if not already present (it's needed for `fmt.Errorf` in the new test cases).

- [ ] **Step 3: Run the tests to verify they fail with a clear compilation error**

```bash
go test github.com/networkengineer-cloud/go-volunteer-media/internal/handlers -run "TestUploadGroupDocument" -count=1 2>&1 | head -10
```

Expected: compilation error — `UploadGroupDocument` called with 3 arguments but defined with 2.

- [ ] **Step 4: Commit the tests and mock (they'll pass after Task 3)**

```bash
git add internal/handlers/test_helpers.go internal/handlers/group_document_test.go
git commit -m "test: add mockConverter and conversion path tests for UploadGroupDocument"
```

---

### Task 3: Update `UploadGroupDocument` handler and wire up in `main.go`

**Files:**
- Modify: `internal/handlers/group_document.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Update imports in `internal/handlers/group_document.go`**

Replace the current import block:

```go
import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/upload"
	"gorm.io/gorm"
)
```

With:

```go
import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/convert"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/storage"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/upload"
	"gorm.io/gorm"
)
```

- [ ] **Step 2: Update the `UploadGroupDocument` signature**

Change:

```go
func UploadGroupDocument(db *gorm.DB, storageProvider storage.Provider) gin.HandlerFunc {
```

To:

```go
func UploadGroupDocument(db *gorm.DB, storageProvider storage.Provider, converter convert.Converter) gin.HandlerFunc {
```

- [ ] **Step 3: Insert the conversion step**

Find the block that reads:

```go
		fileData := buf.Bytes()

		mimeType := upload.MimeTypeFromFilename(file.Filename)
		uploaderID := userID.(uint)
```

Replace it with:

```go
		fileData := buf.Bytes()

		// Convert DOCX and XLSX to PDF so all documents can be viewed inline in the browser.
		// PDFs pass through unchanged. Conversion failure rejects the upload.
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext != ".pdf" {
			pdfData, convErr := converter.ToPDF(ctx, fileData, ext)
			if convErr != nil {
				logger.WithFields(map[string]interface{}{"error": convErr.Error(), "ext": ext}).
					Warn("Document conversion to PDF failed")
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"error": "File could not be converted to PDF. Please check the file and try again.",
				})
				return
			}
			fileData = pdfData
			file.Filename = strings.TrimSuffix(file.Filename, ext) + ".pdf"
		}

		mimeType := upload.MimeTypeFromFilename(file.Filename)
		uploaderID := userID.(uint)
```

- [ ] **Step 4: Update `cmd/api/main.go`**

Add the `convert` import. In the existing import block, add:

```go
"github.com/networkengineer-cloud/go-volunteer-media/internal/convert"
```

After the storage provider initialisation block (after the `logger.WithFields(...).Info("Storage provider initialized")` line), add:

```go
	// Initialize document converter (LibreOffice must be installed in the container).
	converter := &convert.LibreOfficeConverter{}
```

Update the route registration for document upload (around line 398):

```go
			groupAdminDocuments.POST("", handlers.UploadGroupDocument(db, storageProvider, converter))
```

Increase `WriteTimeout` on the HTTP server to accommodate LibreOffice conversion (up to 60 s) plus network transfer. Find the `http.Server` struct and change:

```go
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
```

To:

```go
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
```

- [ ] **Step 5: Run all handler tests**

```bash
go test github.com/networkengineer-cloud/go-volunteer-media/internal/handlers -count=1 2>&1 | tail -5
```

Expected:

```
ok  	github.com/networkengineer-cloud/go-volunteer-media/internal/handlers
```

- [ ] **Step 6: Run the full Go test suite**

```bash
go test ./... 2>&1 | grep -E "^(ok|FAIL|---)"
```

Expected: all packages pass (the two pre-existing failures in `internal/handlers` `TestCreateGroup` and `internal/models` `TestAnimal_CurrentStatusDuration` are unrelated and pre-existing).

- [ ] **Step 7: Commit**

```bash
git add internal/handlers/group_document.go cmd/api/main.go
git commit -m "feat: convert DOCX/XLSX to PDF at upload time via LibreOffice"
```

---

### Task 4: Update Dockerfile

**Files:**
- Modify: `Dockerfile`

- [ ] **Step 1: Replace the final stage**

In `Dockerfile`, find and replace the final stage section:

```dockerfile
# Final stage
FROM alpine:latest

# Install security updates and runtime dependencies
RUN apk update && apk upgrade && \
    apk add --no-cache ca-certificates tzdata && \
    update-ca-certificates

# Create non-root user
RUN adduser -D -g '' appuser
```

With:

```dockerfile
# Final stage
FROM debian:bookworm-slim

# Install runtime dependencies.
# libreoffice: converts DOCX/XLSX uploads to PDF at upload time.
RUN apt-get update && apt-get install -y --no-install-recommends \
    libreoffice \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user (no home directory needed — HOME is set to /tmp below)
RUN useradd -r -s /bin/false appuser

# LibreOffice writes its user profile to HOME; /tmp is writable by all users.
ENV HOME=/tmp
```

Also update the certificates copy line — Alpine uses `/etc/ssl/certs/ca-certificates.crt` and Debian uses the same path, so it should be unchanged. Verify the line reads:

```dockerfile
COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
```

This is correct for both Alpine and Debian.

- [ ] **Step 2: Build the Docker image locally**

```bash
docker build -t volunteer-media-test . 2>&1 | tail -5
```

Expected: `Successfully built <id>` — the image builds without error.
Note: first build takes several minutes while LibreOffice (~300 MB) downloads. Subsequent builds use the layer cache.

- [ ] **Step 3: Verify LibreOffice is available in the image**

```bash
docker run --rm volunteer-media-test libreoffice --version
```

Expected output similar to: `LibreOffice 7.x.x.x ...`

- [ ] **Step 4: Commit**

```bash
git add Dockerfile
git commit -m "feat: switch to debian base and install LibreOffice for PDF conversion"
```

---

### Task 5: Update frontend label

**Files:**
- Modify: `frontend/src/pages/GroupPage.tsx`

- [ ] **Step 1: Update the file input label**

Find:

```tsx
                  <label htmlFor="doc-file">File * (.pdf, .docx, .xlsx)</label>
```

Replace with:

```tsx
                  <label htmlFor="doc-file">File * (.pdf, .docx, .xlsx — DOCX/XLSX will be converted to PDF)</label>
```

- [ ] **Step 2: Run TypeScript check**

```bash
cd frontend && node_modules/.bin/tsc -p tsconfig.app.json --noEmit
```

Expected: no output (clean).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/GroupPage.tsx
git commit -m "feat: inform users that DOCX/XLSX uploads are converted to PDF"
```
