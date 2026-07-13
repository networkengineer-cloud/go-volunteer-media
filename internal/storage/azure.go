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
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/telemetry"
)

// AzureBlobProvider implements the Provider interface using Azure Blob Storage
type AzureBlobProvider struct {
	client        *azblob.Client
	containerName string
	baseURL       string
}

var tracer = telemetry.Tracer("internal/storage/azure")

// mapBlobError translates an Azure blob-storage error into the generic
// ErrNotFound sentinel only when Azure actually reports the blob as missing.
// Any other failure (auth, throttling, network) is returned as-is instead of
// being collapsed to ErrNotFound — callers translate ErrNotFound into an
// HTTP 404, and a real storage error surfacing as "not found" would hide the
// true cause from both the caller and (now that these call sites are traced)
// from anyone cross-referencing the span's recorded error against the
// response the user actually saw.
func mapBlobError(err error) error {
	if bloberror.HasCode(err, bloberror.BlobNotFound) {
		return ErrNotFound
	}
	return fmt.Errorf("azure blob storage error: %w", err)
}

// failBlob maps err and returns it, collapsing the record-then-map pair
// repeated at every blob-storage error path in this file. A "blob not
// found" result is expected, routine traffic (e.g. a request for an
// already-deleted image) — callers translate it into a plain 404, so it is
// not recorded as a span error, which is reserved for the mapped error
// cases that represent a real storage problem (auth, network, throttling).
func failBlob(span trace.Span, err error, msg string) error {
	mapped := mapBlobError(err)
	if mapped == ErrNotFound {
		return mapped
	}
	telemetry.RecordError(span, mapped, msg)
	return mapped
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
	ctx, span := tracer.Start(ctx, "storage.azure.upload_image", trace.WithAttributes(
		attribute.String("blob.mime_type", mimeType),
		attribute.Int("blob.size_bytes", len(data)),
	))
	defer span.End()

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
	case "video/mp4":
		ext = ".mp4"
	case "video/quicktime":
		ext = ".mov"
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
		return "", "", "", telemetry.Fail(span, fmt.Errorf("failed to upload image to Azure: %w", err), "upload failed")
	}

	// Generate API-proxied URL (not direct blob URL)
	// This avoids 409 errors when public access is disabled on the storage account
	url = fmt.Sprintf("/api/images/%s", imageUUID)

	return url, imageUUID, ext, nil
}

// UploadDocument uploads a document to Azure Blob Storage
func (a *AzureBlobProvider) UploadDocument(ctx context.Context, data []byte, mimeType, filename string) (url, identifier, extension string, err error) {
	ctx, span := tracer.Start(ctx, "storage.azure.upload_document", trace.WithAttributes(
		attribute.String("blob.mime_type", mimeType),
		attribute.Int("blob.size_bytes", len(data)),
	))
	defer span.End()

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
		return "", "", "", telemetry.Fail(span, fmt.Errorf("failed to upload document to Azure: %w", err), "upload failed")
	}

	// Generate API-proxied URL (not direct blob URL)
	// This avoids 409 errors when public access is disabled on the storage account
	url = fmt.Sprintf("/api/documents/%s", docUUID)

	return url, docUUID, ext, nil
}

// GetImage retrieves an image from Azure Blob Storage
// The identifier should include the file extension (e.g., "uuid.jpg")
func (a *AzureBlobProvider) GetImage(ctx context.Context, identifier string) (data []byte, mimeType string, err error) {
	ctx, span := tracer.Start(ctx, "storage.azure.get_image")
	defer span.End()

	// Construct blob path using identifier with extension
	blobPath := path.Join("images", "animals", identifier)
	blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)

	// Download blob
	downloadResponse, err := blockBlobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, "", failBlob(span, err, "download failed")
	}
	defer downloadResponse.Body.Close()

	// Read all data
	data, err = io.ReadAll(downloadResponse.Body)
	if err != nil {
		return nil, "", telemetry.Fail(span, fmt.Errorf("failed to read blob data: %w", err), "read failed")
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
	ctx, span := tracer.Start(ctx, "storage.azure.get_document")
	defer span.End()

	// Construct blob path using identifier with extension
	blobPath := path.Join("documents", "protocols", identifier)
	blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)

	// Download blob
	downloadResponse, err := blockBlobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, "", failBlob(span, err, "download failed")
	}
	defer downloadResponse.Body.Close()

	// Read all data
	data, err = io.ReadAll(downloadResponse.Body)
	if err != nil {
		return nil, "", telemetry.Fail(span, fmt.Errorf("failed to read blob data: %w", err), "read failed")
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
	ctx, span := tracer.Start(ctx, "storage.azure.delete_image")
	defer span.End()

	// Construct blob path using identifier with extension
	blobPath := path.Join("images", "animals", identifier)
	blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)

	// Try to delete
	_, err := blockBlobClient.Delete(ctx, nil)
	if err != nil {
		return failBlob(span, err, "delete failed")
	}

	return nil
}

// DeleteDocument deletes a document from Azure Blob Storage
// The identifier should include the file extension (e.g., "uuid.pdf")
func (a *AzureBlobProvider) DeleteDocument(ctx context.Context, identifier string) error {
	ctx, span := tracer.Start(ctx, "storage.azure.delete_document")
	defer span.End()

	// Construct blob path using identifier with extension
	blobPath := path.Join("documents", "protocols", identifier)
	blockBlobClient := a.client.ServiceClient().NewContainerClient(a.containerName).NewBlockBlobClient(blobPath)

	// Try to delete
	_, err := blockBlobClient.Delete(ctx, nil)
	if err != nil {
		return failBlob(span, err, "delete failed")
	}

	return nil
}

// GetImageURL returns the API URL for an image (proxied through the backend).
// We proxy all images through /api/images/:uuid to avoid 409 errors from Azure
// when public access is disabled on the storage account (which is the secure default).
// The identifier is the UUID (without extension) used to look up the image in the database.
func (a *AzureBlobProvider) GetImageURL(identifier string) string {
	// Extract just the UUID portion (strip any extension) for the API URL
	// The database lookup uses the /api/images/{uuid} URL format
	uuidOnly := identifier
	if ext := path.Ext(identifier); ext != "" {
		uuidOnly = identifier[:len(identifier)-len(ext)]
	}
	return fmt.Sprintf("/api/images/%s", uuidOnly)
}

// GetDocumentURL returns the API URL for a document (proxied through the backend).
// We proxy all documents through /api/documents/:uuid to avoid 409 errors from Azure
// when public access is disabled on the storage account (which is the secure default).
// The identifier is the UUID (without extension) used to look up the document in the database.
func (a *AzureBlobProvider) GetDocumentURL(identifier string) string {
	// Extract just the UUID portion (strip any extension) for the API URL
	uuidOnly := identifier
	if ext := path.Ext(identifier); ext != "" {
		uuidOnly = identifier[:len(identifier)-len(ext)]
	}
	return fmt.Sprintf("/api/documents/%s", uuidOnly)
}
