package storage

import (
	"context"
	"fmt"
	"path"

	"github.com/google/uuid"
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

// Name returns the name of this storage provider
func (p *PostgresProvider) Name() string {
	return "postgres"
}

// UploadImage generates a UUID and URL for image storage
// For Postgres, the actual database insertion is handled by the caller
func (p *PostgresProvider) UploadImage(ctx context.Context, data []byte, mimeType string, metadata map[string]string) (url, identifier, extension string, err error) {
	// Generate UUID for the image
	imageUUID := uuid.New().String()
	imageURL := fmt.Sprintf("/api/images/%s", imageUUID)
	
	// Determine extension from MIME type
	ext := ".jpg"
	switch mimeType {
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}
	
	return imageURL, imageUUID, ext, nil
}

// UploadDocument generates a UUID and URL for document storage
// For Postgres, the actual database insertion is handled by the caller
func (p *PostgresProvider) UploadDocument(ctx context.Context, data []byte, mimeType, filename string) (url, identifier, extension string, err error) {
	// Generate UUID for the document
	documentUUID := uuid.New().String()
	documentURL := fmt.Sprintf("/api/documents/%s", documentUUID)
	
	// Get extension from filename or MIME type
	ext := path.Ext(filename)
	if ext == "" {
		switch mimeType {
		case "application/pdf":
			ext = ".pdf"
		case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
			ext = ".docx"
		default:
			ext = ".bin"
		}
	}
	
	return documentURL, documentUUID, ext, nil
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
