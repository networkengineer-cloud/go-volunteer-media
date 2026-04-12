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
