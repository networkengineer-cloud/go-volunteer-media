package storage

import (
	"context"
	"errors"
	"os"

	"gorm.io/gorm"
)

var (
	// ErrNotFound is returned when a file is not found in storage
	ErrNotFound = errors.New("file not found in storage")
	// ErrInvalidProvider is returned when an unsupported storage provider is specified
	ErrInvalidProvider = errors.New("invalid storage provider")
)

// Provider defines the interface for storage operations
// This abstraction allows switching between different storage backends (Postgres, Azure, S3, etc.)
type Provider interface {
	// UploadImage uploads an image and returns its URL and blob identifier
	// data: image binary data (already processed/resized)
	// mimeType: MIME type of the image (e.g., "image/jpeg")
	// metadata: additional metadata (width, height, caption, etc.)
	// Returns: public URL and internal identifier (used for retrieval)
	UploadImage(ctx context.Context, data []byte, mimeType string, metadata map[string]string) (url, identifier string, err error)

	// UploadDocument uploads a document and returns its URL and blob identifier
	// data: document binary data
	// mimeType: MIME type of the document (e.g., "application/pdf")
	// filename: original filename for proper Content-Disposition header
	// Returns: public URL and internal identifier (used for retrieval)
	UploadDocument(ctx context.Context, data []byte, mimeType, filename string) (url, identifier string, err error)

	// GetImage retrieves an image by its identifier
	// Returns: binary data and MIME type
	GetImage(ctx context.Context, identifier string) (data []byte, mimeType string, err error)

	// GetDocument retrieves a document by its identifier
	// Returns: binary data and MIME type
	GetDocument(ctx context.Context, identifier string) (data []byte, mimeType string, err error)

	// DeleteImage removes an image from storage
	DeleteImage(ctx context.Context, identifier string) error

	// DeleteDocument removes a document from storage
	DeleteDocument(ctx context.Context, identifier string) error

	// GetImageURL returns the public URL for an image given its identifier
	// This is used when the URL needs to be generated without fetching the file
	GetImageURL(identifier string) string

	// GetDocumentURL returns the public URL for a document given its identifier
	GetDocumentURL(identifier string) string
}

// Config holds storage provider configuration
type Config struct {
	// Provider specifies which storage backend to use ("postgres" or "azure")
	Provider string

	// Azure Blob Storage configuration
	AzureAccountName   string
	AzureAccountKey    string
	AzureContainerName string
	AzureEndpoint      string // Optional: for custom endpoints (e.g., Azurite local emulator)
	AzureUseManagedID  bool   // Use Azure Managed Identity instead of account key
}

// LoadConfig loads storage configuration from environment variables
func LoadConfig() Config {
	provider := os.Getenv("STORAGE_PROVIDER")
	if provider == "" {
		provider = "postgres" // Default to current behavior for backward compatibility
	}

	useManagedID := false
	if os.Getenv("AZURE_STORAGE_USE_MANAGED_IDENTITY") == "true" {
		useManagedID = true
	}

	return Config{
		Provider:           provider,
		AzureAccountName:   os.Getenv("AZURE_STORAGE_ACCOUNT_NAME"),
		AzureAccountKey:    os.Getenv("AZURE_STORAGE_ACCOUNT_KEY"),
		AzureContainerName: os.Getenv("AZURE_STORAGE_CONTAINER_NAME"),
		AzureEndpoint:      os.Getenv("AZURE_STORAGE_ENDPOINT"),
		AzureUseManagedID:  useManagedID,
	}
}

// NewProvider creates a storage provider based on configuration
func NewProvider(config Config, db *gorm.DB) (Provider, error) {
	switch config.Provider {
	case "postgres":
		return NewPostgresProvider(db), nil
	case "azure":
		if config.AzureAccountName == "" || config.AzureContainerName == "" {
			return nil, errors.New("Azure storage configuration incomplete: account name and container name required")
		}
		if !config.AzureUseManagedID && config.AzureAccountKey == "" {
			return nil, errors.New("Azure storage configuration incomplete: account key required (or enable managed identity)")
		}
		return NewAzureBlobProvider(
			config.AzureAccountName,
			config.AzureAccountKey,
			config.AzureContainerName,
			config.AzureEndpoint,
			config.AzureUseManagedID,
		)
	default:
		return nil, ErrInvalidProvider
	}
}
