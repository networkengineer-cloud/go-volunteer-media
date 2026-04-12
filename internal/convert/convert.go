package convert

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// maxConcurrentConversions caps the number of simultaneous LibreOffice processes.
// Each soffice instance uses ~300-500 MB RAM; unbounded concurrency is an OOM risk.
const maxConcurrentConversions = 3

// Converter converts a document file to PDF bytes.
type Converter interface {
	// ToPDF converts data (a DOCX or XLSX file identified by ext, e.g. ".docx") to PDF.
	// Returns an error if conversion fails or times out.
	ToPDF(ctx context.Context, data []byte, ext string) ([]byte, error)
}

// LibreOfficeConverter converts documents to PDF using a locally installed LibreOffice.
// LibreOffice must be on PATH. The container must set HOME=/tmp so LibreOffice can write
// its user profile when running as a non-root user.
// Use NewLibreOfficeConverter to construct; the zero value is not safe to use.
type LibreOfficeConverter struct {
	sem chan struct{} // bounded semaphore limiting concurrent conversions
}

// NewLibreOfficeConverter returns a LibreOfficeConverter ready for use.
func NewLibreOfficeConverter() *LibreOfficeConverter {
	return &LibreOfficeConverter{sem: make(chan struct{}, maxConcurrentConversions)}
}

func (c *LibreOfficeConverter) ToPDF(ctx context.Context, data []byte, ext string) ([]byte, error) {
	// Acquire semaphore slot; respect context cancellation while waiting.
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, fmt.Errorf("conversion cancelled while waiting for slot: %w", ctx.Err())
	}

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
		// Isolate the user profile per conversion so concurrent requests do not
		// collide on the shared LibreOffice lock file under $HOME/.config/libreoffice/.
		"-env:UserInstallation=file://"+tmpDir,
		"--convert-to", "pdf",
		"--outdir", tmpDir,
		inputPath,
	)
	// Put LibreOffice in its own process group so all child processes
	// (soffice.bin, font loaders, etc.) are killed together on timeout/cancel.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// Negative PID kills the entire process group.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		// Truncate output to avoid flooding log aggregation with large soffice traces.
		if len(out) > 500 {
			out = append(out[:500], []byte("... (truncated)")...)
		}
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
