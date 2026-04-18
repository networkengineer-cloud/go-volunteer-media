# Document PDF Conversion Design

**Date:** 2026-04-11  
**Status:** Approved

## Problem

The group documents feature accepts DOCX and XLSX uploads, but those formats cannot be displayed natively in a browser. This forces all documents to be download-only, which is a poor experience for volunteers who want to quickly review protocols and reference material.

## Goal

Convert DOCX and XLSX files to PDF at upload time so every document can be viewed inline in the browser. If conversion fails the upload is rejected — the viewer experience is the priority, not permissive storage.

## Approach

LibreOffice installed in the production container (Option A). No additional services or infrastructure required. LibreOffice is invoked as a subprocess at upload time.

## Design

### 1. Dockerfile

Switch the final stage from `alpine:latest` to `debian:bookworm-slim` and install LibreOffice.

- Alpine's LibreOffice package is fragile and poorly maintained; Debian is the standard target.
- `HOME=/tmp` is set as an environment variable so LibreOffice can write its user profile when running as the non-root `appuser`.
- `--no-install-recommends` keeps the image lean.
- Everything else is unchanged: non-root user, binary copy, port 8080, entrypoint.

### 2. Conversion package — `internal/convert/convert.go`

Single exported function:

```go
func ToPDF(ctx context.Context, data []byte) ([]byte, error)
```

Behaviour:
1. Creates a temp directory via `os.MkdirTemp("", "doc-convert-*")`.
2. Writes input bytes to a file in that directory.
3. Calls `libreoffice --headless --convert-to pdf --outdir <tmpDir> <inputFile>` via `exec.CommandContext` with a 60-second timeout.
4. Reads the resulting `.pdf` file.
5. Removes the temp directory on return (deferred).
6. Returns an error if the command exits non-zero or the output file is missing.

LibreOffice detects the input format from the file content, so the filename extension written to disk must match the original (e.g. `input.docx`, `input.xlsx`) to ensure correct format detection.

### 3. Upload handler — `internal/handlers/group_document.go`

After `ValidateDocumentUpload` succeeds, insert a conversion step for non-PDF files:

```go
ext := strings.ToLower(filepath.Ext(file.Filename))
if ext != ".pdf" {
    pdfData, err := convert.ToPDF(ctx, fileData)
    if err != nil {
        logger.WithFields(map[string]interface{}{"error": err.Error()}).
            Warn("Document conversion to PDF failed")
        c.JSON(http.StatusUnprocessableEntity, gin.H{
            "error": "File could not be converted to PDF. Please check the file and try again.",
        })
        return
    }
    fileData = pdfData
    mimeType = "application/pdf"
    file.Filename = strings.TrimSuffix(file.Filename, ext) + ".pdf"
}
```

The remainder of the handler is unchanged — it receives PDF bytes regardless of the original format.

### 4. Frontend — `GroupPage.tsx`

Update the file input label to set user expectations:

```
File * (.pdf, .docx, .xlsx — DOCX/XLSX will be converted to PDF)
```

No other frontend changes are required. The accepted file types remain the same.

## Error Handling

| Scenario | Response |
|---|---|
| Conversion succeeds | 201 Created with PDF stored |
| LibreOffice exits non-zero | 422 Unprocessable Entity — user-facing message |
| Conversion times out (>60s) | 422 Unprocessable Entity — same message |
| PDF uploaded directly | Skips conversion, stored as-is |

## Testing

- Unit tests for `ToPDF` require LibreOffice on the test machine. Tests will be integration-style, guarded by a `libreoffice` binary check: skip if not present.
- The upload handler conversion path is tested by injecting a mock converter interface — same pattern as the storage provider mock.
- Existing upload handler tests continue to pass unchanged (they use `.pdf` files, which skip conversion).

## Out of Scope

- Converting PDFs to other formats.
- Preserving the original DOCX/XLSX alongside the converted PDF.
- On-the-fly conversion at serve time.
- Image or other file type conversion.
