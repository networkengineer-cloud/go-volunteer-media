package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

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

// Name returns the name of this storage provider
func (a *AzureBlobProvider) Name() string {
	return "azure"
}

// ensureContainer creates the container if it doesn't exist
func (a *AzureBlobProvider) ensureContainer(ctx context.Context) error {
	containerClient := a.client.ServiceClient().NewContainerClient(a.containerName)
	
	// Try to create container with private access (no public access - default when Access is nil)
	_, err := containerClient.Create(ctx, &container.CreateOptions{
		Access: nil, // nil = private container (no public access)
	})
	
	if err != nil {
		// Check if the error message contains "ContainerAlreadyExists"
		// This is the standard Azure error code when container exists
		errMsg := err.Error()
		if strings.Contains(errMsg, "ContainerAlreadyExists") || strings.Contains(errMsg, "The specified container already exists") {
			// Container exists, this is fine
			return nil
		}
		// Real error (permission denied, network failure, etc.)
		return fmt.Errorf("failed to create container: %w", err)
	}
	
	return nil
}

// UploadImage uploads an image to Azure Blob Storage
func (a *AzureBlobProvider) UploadImage(ctx context.Context, data []byte, mimeType string, metadata map[string]string) (url, identifier, extension string, err error) {
	// Generate unique identifier
	imageUUID := uuid.New().String()
	
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
	blobPath := path.Join("images", "animals", imageUUID+ext)
	
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
		return "", "", "", fmt.Errorf("failed to upload image to Azure: %w", err)
	}
	
	// Generate URL
	url = fmt.Sprintf("%s/%s", a.baseURL, blobPath)
	
	return url, imageUUID, ext, nil
}

// UploadDocument uploads a document to Azure Blob Storage
func (a *AzureBlobProvider) UploadDocument(ctx context.Context, data []byte, mimeType, filename string) (url, identifier, extension string, err error) {
	// Generate unique identifier
	docUUID := uuid.New().String()
	
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
	blobPath := path.Join("documents", "protocols", docUUID+ext)
	
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
		return "", "", "", fmt.Errorf("failed to upload document to Azure: %w", err)
	}
	
	// Generate URL
	url = fmt.Sprintf("%s/%s", a.baseURL, blobPath)
	
	return url, docUUID, ext, nil
}

// GetImage retrieves an image from Azure Blob Storage
// The identifier should include the file extension (e.g., "uuid.jpg")
func (a *AzureBlobProvider) GetImage(ctx context.Context, identifier string) (data []byte, mimeType string, err error) {
	// Construct blob path using identifier with extension
	blobPath := path.Join("images", "animals", identifier)
	blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)
	
	// Download blob
	downloadResponse, err := blockBlobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, "", ErrNotFound
	}
	defer downloadResponse.Body.Close()
	
	// Read all data
	data, err = io.ReadAll(downloadResponse.Body)
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

// GetDocument retrieves a document from Azure Blob Storage
// The identifier should include the file extension (e.g., "uuid.pdf")
func (a *AzureBlobProvider) GetDocument(ctx context.Context, identifier string) (data []byte, mimeType string, err error) {
	// Construct blob path using identifier with extension
	blobPath := path.Join("documents", "protocols", identifier)
	blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)
	
	// Download blob
	downloadResponse, err := blockBlobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, "", ErrNotFound
	}
	defer downloadResponse.Body.Close()
	
	// Read all data
	data, err = io.ReadAll(downloadResponse.Body)
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

// DeleteImage deletes an image from Azure Blob Storage
// The identifier should include the file extension (e.g., "uuid.jpg")
func (a *AzureBlobProvider) DeleteImage(ctx context.Context, identifier string) error {
	// Construct blob path using identifier with extension
	blobPath := path.Join("images", "animals", identifier)
	blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)
	
	// Try to delete
	_, err := blockBlobClient.Delete(ctx, nil)
	if err != nil {
		return ErrNotFound
	}
	
	return nil
}

// DeleteDocument deletes a document from Azure Blob Storage
// The identifier should include the file extension (e.g., "uuid.pdf")
func (a *AzureBlobProvider) DeleteDocument(ctx context.Context, identifier string) error {
	// Construct blob path using identifier with extension
	blobPath := path.Join("documents", "protocols", identifier)
	blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)
	
	// Try to delete
	_, err := blockBlobClient.Delete(ctx, nil)
	if err != nil {
		return ErrNotFound
	}
	
	return nil
}

// GetImageURL returns the Azure Blob Storage URL for an image
// The identifier should include the file extension (e.g., "uuid.jpg")
func (a *AzureBlobProvider) GetImageURL(identifier string) string {
	blobPath := path.Join("images", "animals", identifier)
	return fmt.Sprintf("%s/%s", a.baseURL, blobPath)
}

// GetDocumentURL returns the Azure Blob Storage URL for a document
// The identifier should include the file extension (e.g., "uuid.pdf")
func (a *AzureBlobProvider) GetDocumentURL(identifier string) string {
	blobPath := path.Join("documents", "protocols", identifier)
	return fmt.Sprintf("%s/%s", a.baseURL, blobPath)
}
