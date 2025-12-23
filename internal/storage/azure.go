package storage

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/google/uuid"
)

// AzureBlobProvider implements the Provider interface using Azure Blob Storage
type AzureBlobProvider struct {
	client        *azblob.Client
	containerName string
	baseURL       string
}

// NewAzureBlobProvider creates a new Azure Blob Storage provider
func NewAzureBlobProvider(accountName, accountKey, containerName, endpoint string, useManagedID bool) (*AzureBlobProvider, error) {
	var client *azblob.Client

	// Determine endpoint URL
	var serviceURL string
	if endpoint != "" {
		// Use custom endpoint (e.g., Azurite for local testing)
		serviceURL = endpoint
	} else {
		// Use standard Azure endpoint
		serviceURL = fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	}

	// Create client based on authentication method
	if useManagedID {
		// TODO: Implement Managed Identity authentication
		// This requires Azure Identity SDK
		return nil, fmt.Errorf("managed identity authentication not yet implemented")
	}
	
	// Use shared key credential
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	client, err = azblob.NewClientWithSharedKeyCredential(serviceURL, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure blob client: %w", err)
	}

	provider := &AzureBlobProvider{
		client:        client,
		containerName: containerName,
		baseURL:       fmt.Sprintf("%s%s", serviceURL, containerName),
	}

	// Ensure container exists
	if err := provider.ensureContainer(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure container exists: %w", err)
	}

	return provider, nil
}

// ensureContainer creates the container if it doesn't exist
func (a *AzureBlobProvider) ensureContainer(ctx context.Context) error {
	containerClient := a.client.ServiceClient().NewContainerClient(a.containerName)
	
	// Try to create container (will succeed if it doesn't exist, fail gracefully if it does)
	_, err := containerClient.Create(ctx, &container.CreateOptions{
		Access: to.Ptr(container.PublicAccessTypeBlob), // Blob-level public access
	})
	
	if err != nil {
		// Check if container already exists (not an error)
		// Azure SDK returns specific error codes we can check
		// For now, we'll just log and continue
		// In production, you'd want more sophisticated error handling
	}
	
	return nil
}

// UploadImage uploads an image to Azure Blob Storage
func (a *AzureBlobProvider) UploadImage(ctx context.Context, data []byte, mimeType string, metadata map[string]string) (url, identifier string, err error) {
	// Generate unique identifier
	imageID := uuid.New().String()
	
	// Determine file extension from MIME type
	ext := ".jpg"
	switch mimeType {
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}
	
	// Construct blob path: images/animals/{uuid}{ext}
	blobPath := path.Join("images", "animals", imageID+ext)
	identifier = imageID
	
	// Upload to Azure Blob Storage
	blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)
	
	// Convert metadata from map[string]string to map[string]*string
	azureMetadata := make(map[string]*string)
	for k, v := range metadata {
		value := v
		azureMetadata[k] = &value
	}
	
	// Prepare options with content type and metadata
	uploadOptions := &azblob.UploadBufferOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: &mimeType,
		},
		Metadata: azureMetadata,
	}
	
	// Upload the data
	_, err = blockBlobClient.UploadBuffer(ctx, data, uploadOptions)
	if err != nil {
		return "", "", fmt.Errorf("failed to upload image to Azure: %w", err)
	}
	
	// Generate URL
	url = fmt.Sprintf("%s/%s", a.baseURL, blobPath)
	
	return url, identifier, nil
}

// UploadDocument uploads a document to Azure Blob Storage
func (a *AzureBlobProvider) UploadDocument(ctx context.Context, data []byte, mimeType, filename string) (url, identifier string, err error) {
	// Generate unique identifier
	docID := uuid.New().String()
	
	// Determine file extension from filename
	ext := path.Ext(filename)
	if ext == "" {
		// Fallback based on MIME type
		switch mimeType {
		case "application/pdf":
			ext = ".pdf"
		case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
			ext = ".docx"
		default:
			ext = ".bin"
		}
	}
	
	// Construct blob path: documents/protocols/{uuid}{ext}
	blobPath := path.Join("documents", "protocols", docID+ext)
	identifier = docID
	
	// Upload to Azure Blob Storage
	blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)
	
	// Prepare filename metadata
	filenameValue := filename
	
	// Prepare options with content type
	uploadOptions := &azblob.UploadBufferOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType:        &mimeType,
			BlobContentDisposition: to.Ptr(fmt.Sprintf("inline; filename=\"%s\"", filename)),
		},
		Metadata: map[string]*string{
			"filename": &filenameValue,
		},
	}
	
	// Upload the data
	_, err = blockBlobClient.UploadBuffer(ctx, data, uploadOptions)
	if err != nil {
		return "", "", fmt.Errorf("failed to upload document to Azure: %w", err)
	}
	
	// Generate URL
	url = fmt.Sprintf("%s/%s", a.baseURL, blobPath)
	
	return url, identifier, nil
}

// GetImage retrieves an image from Azure Blob Storage
func (a *AzureBlobProvider) GetImage(ctx context.Context, identifier string) (data []byte, mimeType string, err error) {
	// Try common image extensions
	extensions := []string{".jpg", ".png", ".gif", ".webp"}
	
	for _, ext := range extensions {
		blobPath := path.Join("images", "animals", identifier+ext)
		blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)
		
		// Download blob
		downloadResponse, err := blockBlobClient.DownloadStream(ctx, nil)
		if err == nil {
			// Successfully downloaded
			defer downloadResponse.Body.Close()
			
			// Read all data
			data, err := io.ReadAll(downloadResponse.Body)
			if err != nil {
				return nil, "", fmt.Errorf("failed to read blob data: %w", err)
			}
			
			// Get content type
			contentType := ""
			if downloadResponse.ContentType != nil {
				contentType = *downloadResponse.ContentType
			}
			
			return data, contentType, nil
		}
	}
	
	return nil, "", ErrNotFound
}

// GetDocument retrieves a document from Azure Blob Storage
func (a *AzureBlobProvider) GetDocument(ctx context.Context, identifier string) (data []byte, mimeType string, err error) {
	// Try common document extensions
	extensions := []string{".pdf", ".docx", ".doc", ".bin"}
	
	for _, ext := range extensions {
		blobPath := path.Join("documents", "protocols", identifier+ext)
		blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)
		
		// Download blob
		downloadResponse, err := blockBlobClient.DownloadStream(ctx, nil)
		if err == nil {
			// Successfully downloaded
			defer downloadResponse.Body.Close()
			
			// Read all data
			data, err := io.ReadAll(downloadResponse.Body)
			if err != nil {
				return nil, "", fmt.Errorf("failed to read blob data: %w", err)
			}
			
			// Get content type
			contentType := ""
			if downloadResponse.ContentType != nil {
				contentType = *downloadResponse.ContentType
			}
			
			return data, contentType, nil
		}
	}
	
	return nil, "", ErrNotFound
}

// DeleteImage deletes an image from Azure Blob Storage
func (a *AzureBlobProvider) DeleteImage(ctx context.Context, identifier string) error {
	// Try common image extensions
	extensions := []string{".jpg", ".png", ".gif", ".webp"}
	
	for _, ext := range extensions {
		blobPath := path.Join("images", "animals", identifier+ext)
		blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)
		
		// Try to delete
		_, err := blockBlobClient.Delete(ctx, nil)
		if err == nil {
			return nil // Successfully deleted
		}
	}
	
	// If none found, return not found error
	return ErrNotFound
}

// DeleteDocument deletes a document from Azure Blob Storage
func (a *AzureBlobProvider) DeleteDocument(ctx context.Context, identifier string) error {
	// Try common document extensions
	extensions := []string{".pdf", ".docx", ".doc", ".bin"}
	
	for _, ext := range extensions {
		blobPath := path.Join("documents", "protocols", identifier+ext)
		blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)
		
		// Try to delete
		_, err := blockBlobClient.Delete(ctx, nil)
		if err == nil {
			return nil // Successfully deleted
		}
	}
	
	// If none found, return not found error
	return ErrNotFound
}

// GetImageURL returns the Azure Blob Storage URL for an image
func (a *AzureBlobProvider) GetImageURL(identifier string) string {
	// Default to JPEG extension (most common)
	blobPath := path.Join("images", "animals", identifier+".jpg")
	return fmt.Sprintf("%s/%s", a.baseURL, blobPath)
}

// GetDocumentURL returns the Azure Blob Storage URL for a document
func (a *AzureBlobProvider) GetDocumentURL(identifier string) string {
	// Default to PDF extension (most common)
	blobPath := path.Join("documents", "protocols", identifier+".pdf")
	return fmt.Sprintf("%s/%s", a.baseURL, blobPath)
}
