package storage

import (
	"context"
	"fmt"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
)

// PostgresProvider implements the Provider interface using PostgreSQL database
// This maintains backward compatibility with the existing storage mechanism
type PostgresProvider struct {
	db *gorm.DB
}

// NewPostgresProvider creates a new PostgreSQL storage provider
func NewPostgresProvider(db *gorm.DB) *PostgresProvider {
	return &PostgresProvider{db: db}
}

// UploadImage stores image data in the database and returns a URL
func (p *PostgresProvider) UploadImage(ctx context.Context, data []byte, mimeType string, metadata map[string]string) (url, identifier string, err error) {
	// For Postgres provider, we don't actually store the data here
	// The caller will create the AnimalImage record with the data
	// We just generate and return a URL based on a UUID
	// The identifier is the UUID extracted from the URL
	
	// Note: This is a transitional implementation
	// The actual database insertion happens in the handler
	// This provider is mainly for serving existing files
	
	return "", "", fmt.Errorf("postgres provider UploadImage should not be called directly; use legacy upload flow")
}

// UploadDocument stores document data in the database and returns a URL
func (p *PostgresProvider) UploadDocument(ctx context.Context, data []byte, mimeType, filename string) (url, identifier string, err error) {
	// Similar to UploadImage, this is a transitional implementation
	return "", "", fmt.Errorf("postgres provider UploadDocument should not be called directly; use legacy upload flow")
}

// GetImage retrieves image data from the database
func (p *PostgresProvider) GetImage(ctx context.Context, identifier string) (data []byte, mimeType string, err error) {
	imageURL := fmt.Sprintf("/api/images/%s", identifier)
	
	var animalImage models.AnimalImage
	if err := p.db.WithContext(ctx).Where("image_url = ?", imageURL).First(&animalImage).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("failed to query image: %w", err)
	}

	if len(animalImage.ImageData) == 0 {
		return nil, "", ErrNotFound
	}

	return animalImage.ImageData, animalImage.MimeType, nil
}

// GetDocument retrieves document data from the database
func (p *PostgresProvider) GetDocument(ctx context.Context, identifier string) (data []byte, mimeType string, err error) {
	documentURL := fmt.Sprintf("/api/documents/%s", identifier)
	
	var animal models.Animal
	if err := p.db.WithContext(ctx).Where("protocol_document_url = ?", documentURL).First(&animal).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("failed to query document: %w", err)
	}

	if len(animal.ProtocolDocumentData) == 0 {
		return nil, "", ErrNotFound
	}

	return animal.ProtocolDocumentData, animal.ProtocolDocumentType, nil
}

// DeleteImage removes image data from the database (soft delete handled by GORM)
func (p *PostgresProvider) DeleteImage(ctx context.Context, identifier string) error {
	// For Postgres provider, deletion is handled by the handler via GORM soft delete
	// This method is a no-op for compatibility
	return nil
}

// DeleteDocument removes document data from the database
func (p *PostgresProvider) DeleteDocument(ctx context.Context, identifier string) error {
	// For Postgres provider, deletion is handled by the handler
	// This method is a no-op for compatibility
	return nil
}

// GetImageURL returns the API URL for an image
func (p *PostgresProvider) GetImageURL(identifier string) string {
	return fmt.Sprintf("/api/images/%s", identifier)
}

// GetDocumentURL returns the API URL for a document
func (p *PostgresProvider) GetDocumentURL(identifier string) string {
	return fmt.Sprintf("/api/documents/%s", identifier)
}
