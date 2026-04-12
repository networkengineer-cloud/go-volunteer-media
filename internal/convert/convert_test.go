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
