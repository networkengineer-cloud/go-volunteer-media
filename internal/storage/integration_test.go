package storage

import (
	"context"
	"os"
	"testing"
)

// TestAzureBlobProviderIntegration tests the Azure Blob Storage provider with Azurite
// This test is skipped by default and requires Azurite to be running
//
// To run integration tests with Azurite:
// 1. Start Azurite: docker run -p 10000:10000 mcr.microsoft.com/azure-storage/azurite
// 2. Run: go test -tags=integration ./internal/storage/...
func TestAzureBlobProviderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if Azurite is configured
	if os.Getenv("AZURE_STORAGE_ENDPOINT") == "" {
		t.Skip("Skipping integration test: AZURE_STORAGE_ENDPOINT not set")
	}

	// Azurite default credentials
	config := Config{
		Provider:           "azure",
		AzureAccountName:   "devstoreaccount1",
		AzureAccountKey:    "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==",
		AzureContainerName: "test-container",
		AzureEndpoint:      "http://127.0.0.1:10000/devstoreaccount1",
		AzureUseManagedID:  false,
	}

	provider, err := NewAzureBlobProvider(
		config.AzureAccountName,
		config.AzureAccountKey,
		config.AzureContainerName,
		config.AzureEndpoint,
		config.AzureUseManagedID,
	)
	if err != nil {
		t.Fatalf("Failed to create Azure provider: %v", err)
	}

	ctx := context.Background()

	t.Run("upload and retrieve image", func(t *testing.T) {
		// Sample image data (1x1 red pixel JPEG)
		imageData := []byte{
			0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01,
			0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xff, 0xdb, 0x00, 0x43,
		}
		metadata := map[string]string{
			"width":   "100",
			"height":  "100",
			"caption": "test image",
		}

		// Upload image
		url, identifier, err := provider.UploadImage(ctx, imageData, "image/jpeg", metadata)
		if err != nil {
			t.Fatalf("Failed to upload image: %v", err)
		}
		if url == "" || identifier == "" {
			t.Error("Expected non-empty URL and identifier")
		}
		t.Logf("Uploaded image: URL=%s, ID=%s", url, identifier)

		// Retrieve image
		retrievedData, mimeType, err := provider.GetImage(ctx, identifier)
		if err != nil {
			t.Fatalf("Failed to retrieve image: %v", err)
		}
		if mimeType != "image/jpeg" {
			t.Errorf("Expected MIME type image/jpeg, got %s", mimeType)
		}
		if len(retrievedData) == 0 {
			t.Error("Expected non-empty image data")
		}
		t.Logf("Retrieved image: %d bytes, MIME type: %s", len(retrievedData), mimeType)

		// Delete image
		err = provider.DeleteImage(ctx, identifier)
		if err != nil {
			t.Errorf("Failed to delete image: %v", err)
		}

		// Verify deletion
		_, _, err = provider.GetImage(ctx, identifier)
		if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound after deletion, got: %v", err)
		}
	})

	t.Run("upload and retrieve document", func(t *testing.T) {
		// Sample PDF data (minimal valid PDF)
		documentData := []byte("%PDF-1.4\n%\n1 0 obj\n<<\n/Type /Catalog\n/Pages 2 0 R\n>>\nendobj\n")
		filename := "test-protocol.pdf"

		// Upload document
		url, identifier, err := provider.UploadDocument(ctx, documentData, "application/pdf", filename)
		if err != nil {
			t.Fatalf("Failed to upload document: %v", err)
		}
		if url == "" || identifier == "" {
			t.Error("Expected non-empty URL and identifier")
		}
		t.Logf("Uploaded document: URL=%s, ID=%s", url, identifier)

		// Retrieve document
		retrievedData, mimeType, err := provider.GetDocument(ctx, identifier)
		if err != nil {
			t.Fatalf("Failed to retrieve document: %v", err)
		}
		if mimeType != "application/pdf" {
			t.Errorf("Expected MIME type application/pdf, got %s", mimeType)
		}
		if len(retrievedData) == 0 {
			t.Error("Expected non-empty document data")
		}
		t.Logf("Retrieved document: %d bytes, MIME type: %s", len(retrievedData), mimeType)

		// Delete document
		err = provider.DeleteDocument(ctx, identifier)
		if err != nil {
			t.Errorf("Failed to delete document: %v", err)
		}

		// Verify deletion
		_, _, err = provider.GetDocument(ctx, identifier)
		if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound after deletion, got: %v", err)
		}
	})
}
