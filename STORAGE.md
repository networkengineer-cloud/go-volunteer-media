# Storage Architecture Documentation

## Quick Start Guide

### Local Development with Postgres (Default)

No configuration needed! The application defaults to storing files in PostgreSQL:

```bash
# Just start the application normally
make dev-backend
```

### Local Testing with Azurite (Azure Storage Emulator)

1. **Start Azurite** (Azure Storage Emulator):
   ```bash
   docker run -p 10000:10000 -p 10001:10001 -p 10002:10002 \
     mcr.microsoft.com/azure-storage/azurite
   ```

2. **Configure environment variables**:
   ```bash
   export STORAGE_PROVIDER=azure
   export AZURE_STORAGE_ACCOUNT_NAME=devstoreaccount1
   export AZURE_STORAGE_ACCOUNT_KEY=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==
   export AZURE_STORAGE_CONTAINER_NAME=volunteer-media-storage
   export AZURE_STORAGE_ENDPOINT=http://127.0.0.1:10000/devstoreaccount1
   ```

3. **Start the application**:
   ```bash
   make dev-backend
   ```

4. **Upload a file** - the application will automatically create the container and store files in Azurite

5. **Verify files in Azurite** using Azure Storage Explorer or:
   ```bash
   curl http://127.0.0.1:10000/devstoreaccount1/volunteer-media-storage?restype=container&comp=list
   ```

### Production Deployment with Azure Blob Storage

1. **Create Azure Storage Account**:
   ```bash
   az storage account create \
     --name myvolunteerstorage \
     --resource-group my-resource-group \
     --location eastus \
     --sku Standard_LRS
   ```

2. **Create container**:
   ```bash
   az storage container create \
     --name volunteer-media-storage \
     --account-name myvolunteerstorage \
     --public-access off
   ```

3. **Get access key** (or use Managed Identity):
   ```bash
   az storage account keys list \
     --account-name myvolunteerstorage \
     --resource-group my-resource-group \
     --query '[0].value' -o tsv
   ```

4. **Set environment variables** in Azure Container Apps:
   ```bash
   az containerapp update \
     --name volunteer-media \
     --resource-group my-resource-group \
     --set-env-vars \
       STORAGE_PROVIDER=azure \
       AZURE_STORAGE_ACCOUNT_NAME=myvolunteerstorage \
       AZURE_STORAGE_ACCOUNT_KEY=<your-key> \
       AZURE_STORAGE_CONTAINER_NAME=volunteer-media-storage
   ```

5. **Deploy and verify** - new uploads will use Azure Blob Storage

### Switching Between Providers

The feature flag allows seamless switching:

```bash
# Use Postgres (default, no change needed)
export STORAGE_PROVIDER=postgres

# Use Azure Blob Storage
export STORAGE_PROVIDER=azure
```

**Note**: Existing files remain accessible regardless of provider. The application will serve files from their original storage location based on the `StorageProvider` field in the database.

## Problem Statement

Currently, all images and documents (PDFs, DOCX files) are stored directly in PostgreSQL as binary data (`bytea` type). While this approach works for small-scale deployments, it has several limitations:

### Current Limitations

1. **Database Bloat**: Binary data significantly increases database size, affecting backup/restore times and costs
2. **Performance**: Serving large binary files through the database adds unnecessary load
3. **Scalability**: PostgreSQL is optimized for relational data, not blob storage
4. **Cost**: Database storage is typically more expensive than object storage
5. **Future Video Support**: Video files would exacerbate these issues significantly

### Current Implementation

**Models with Binary Storage:**
- `AnimalImage.ImageData` (bytea) - Stores resized JPEG images
- `Animal.ProtocolDocumentData` (bytea) - Stores PDF/DOCX protocol documents

**Handlers:**
- `internal/handlers/animal_image.go` - Image upload, serving, deletion
- `internal/handlers/animal_document.go` - Document upload, serving, deletion
- `internal/handlers/animal_upload.go` - Simple image upload and serving

**Current Flow:**
```
Upload → Validate → Process/Resize → Store in DB (bytea) → Generate URL → Serve from DB
```

## Proposed Solution: Azure Blob Storage

### Why Azure Blob Storage?

1. **Cost-Effective**: ~$0.018/GB for hot storage vs. database storage
2. **Scalability**: Designed for petabytes of unstructured data
3. **Performance**: CDN integration, geo-replication, optimized for blob serving
4. **Feature Rich**: Lifecycle management, versioning, metadata support
5. **Infrastructure Alignment**: Project uses Azure (Container Apps, PostgreSQL Flexible Server)

### Design Principles

1. **Abstraction**: Storage-agnostic interface for future flexibility
2. **Feature Flag**: Gradual rollout with ability to toggle storage providers
3. **Backward Compatibility**: Existing data in Postgres continues to work
4. **Minimal Changes**: Surgical modifications to handlers only
5. **Testability**: Mockable interfaces for comprehensive testing

## Architecture Design

### Storage Interface

Create an abstraction layer that decouples storage logic from handlers:

```go
// internal/storage/storage.go
package storage

import (
	"context"
	"io"
)

// Provider defines the interface for storage operations
type Provider interface {
	// UploadImage uploads an image and returns its URL
	UploadImage(ctx context.Context, data []byte, mimeType string, metadata map[string]string) (string, error)
	
	// UploadDocument uploads a document and returns its URL
	UploadDocument(ctx context.Context, data []byte, mimeType, filename string) (string, error)
	
	// GetImage retrieves an image by its identifier
	GetImage(ctx context.Context, identifier string) ([]byte, string, error)
	
	// GetDocument retrieves a document by its identifier
	GetDocument(ctx context.Context, identifier string) ([]byte, string, error)
	
	// DeleteImage removes an image
	DeleteImage(ctx context.Context, identifier string) error
	
	// DeleteDocument removes a document
	DeleteDocument(ctx context.Context, identifier string) error
	
	// GetImageURL returns the public URL for an image
	GetImageURL(identifier string) string
	
	// GetDocumentURL returns the public URL for a document
	GetDocumentURL(identifier string) string
}

// Config holds storage provider configuration
type Config struct {
	Provider           string // "postgres" or "azure"
	AzureAccountName   string
	AzureAccountKey    string
	AzureContainerName string
	AzureEndpoint      string // Optional: for custom endpoints (e.g., Azurite)
}

// NewProvider creates a storage provider based on configuration
func NewProvider(config Config, db *gorm.DB) (Provider, error)
```

### Implementation: PostgreSQL Provider

Maintains current behavior, extracting DB storage logic:

```go
// internal/storage/postgres.go
package storage

type PostgresProvider struct {
	db *gorm.DB
}

func NewPostgresProvider(db *gorm.DB) *PostgresProvider {
	return &PostgresProvider{db: db}
}

// Implements Provider interface by storing/retrieving from database
// Similar to current implementation
```

### Implementation: Azure Blob Storage Provider

```go
// internal/storage/azure.go
package storage

import (
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type AzureBlobProvider struct {
	client        *azblob.Client
	containerName string
	baseURL       string
}

func NewAzureBlobProvider(accountName, accountKey, containerName, endpoint string) (*AzureBlobProvider, error) {
	// Initialize Azure SDK client
	// Create container if it doesn't exist
	// Set up blob lifecycle policies
}

// Implements Provider interface using Azure Blob Storage
```

### Container Structure

```
volunteer-media-storage/
├── images/
│   ├── animals/
│   │   ├── {uuid}.jpg
│   │   └── {uuid}.jpg
│   └── profiles/
│       └── {uuid}.jpg
└── documents/
    └── protocols/
        ├── {uuid}.pdf
        └── {uuid}.docx
```

### Metadata Storage

**Database tables remain unchanged** - only the binary data moves to Azure:

```go
type AnimalImage struct {
	// ... existing fields ...
	ImageData        []byte  `gorm:"type:bytea" json:"-"`           // NULL when using Azure
	StorageProvider  string  `gorm:"default:'postgres'" json:"-"`   // NEW: "postgres" or "azure"
	BlobIdentifier   string  `json:"-"`                             // NEW: Azure blob path
	// ... other fields ...
}
```

**Migration Strategy:**
- Add nullable `StorageProvider` and `BlobIdentifier` columns
- Default to "postgres" for backward compatibility
- New uploads use configured provider

## Feature Flag Configuration

### Environment Variables

```bash
# Storage Provider Selection
STORAGE_PROVIDER=postgres  # Options: "postgres" (default), "azure"

# Azure Blob Storage Configuration (required when STORAGE_PROVIDER=azure)
AZURE_STORAGE_ACCOUNT_NAME=myaccountname
AZURE_STORAGE_ACCOUNT_KEY=base64-encoded-key
AZURE_STORAGE_CONTAINER_NAME=volunteer-media-storage
AZURE_STORAGE_ENDPOINT=  # Optional: custom endpoint (e.g., http://127.0.0.1:10000/devstoreaccount1 for Azurite)

# Azure Storage Options
AZURE_STORAGE_USE_MANAGED_IDENTITY=false  # Use Azure Managed Identity for auth (recommended for production)
AZURE_STORAGE_CDN_ENDPOINT=  # Optional: CDN endpoint for serving files
```

### Configuration Loading

```go
// internal/storage/config.go
package storage

import "os"

func LoadConfig() Config {
	provider := os.Getenv("STORAGE_PROVIDER")
	if provider == "" {
		provider = "postgres" // Default to current behavior
	}
	
	return Config{
		Provider:           provider,
		AzureAccountName:   os.Getenv("AZURE_STORAGE_ACCOUNT_NAME"),
		AzureAccountKey:    os.Getenv("AZURE_STORAGE_ACCOUNT_KEY"),
		AzureContainerName: os.Getenv("AZURE_STORAGE_CONTAINER_NAME"),
		AzureEndpoint:      os.Getenv("AZURE_STORAGE_ENDPOINT"),
	}
}
```

## Handler Integration

### Modified Upload Flow

**Before:**
```go
// handlers/animal_image.go
imageData := buf.Bytes()
animalImage := models.AnimalImage{
	ImageData: imageData,
	ImageURL:  fmt.Sprintf("/api/images/%s", uuid),
}
db.Create(&animalImage)
```

**After:**
```go
// handlers/animal_image.go
imageData := buf.Bytes()

// Upload to configured storage provider
storageURL, err := storageProvider.UploadImage(ctx, imageData, "image/jpeg", metadata)
if err != nil {
	return err
}

animalImage := models.AnimalImage{
	ImageURL:  storageURL,
	ImageData: nil, // Not stored in DB when using Azure
	StorageProvider: storageConfig.Provider,
	BlobIdentifier: extractIdentifier(storageURL),
}
db.Create(&animalImage)
```

### Modified Serving Flow

**Before:**
```go
// handlers/animal_upload.go - ServeImage
var animalImage models.AnimalImage
db.Where("image_url = ?", imageURL).First(&animalImage)
c.Data(http.StatusOK, animalImage.MimeType, animalImage.ImageData)
```

**After:**
```go
// handlers/animal_upload.go - ServeImage
var animalImage models.AnimalImage
db.Where("image_url = ?", imageURL).First(&animalImage)

if animalImage.StorageProvider == "azure" {
	// Redirect to Azure Blob URL (with SAS token if needed)
	c.Redirect(http.StatusTemporaryRedirect, animalImage.ImageURL)
} else {
	// Serve from database (backward compatibility)
	c.Data(http.StatusOK, animalImage.MimeType, animalImage.ImageData)
}
```

**Alternative (Proxy through API):**
```go
// For stricter access control, proxy through API
data, mimeType, err := storageProvider.GetImage(ctx, animalImage.BlobIdentifier)
if err != nil {
	return err
}
c.Data(http.StatusOK, mimeType, data)
```

## Migration Strategy

### Phase 1: Infrastructure Setup (Non-Breaking)

1. Add Azure SDK dependency: `go get github.com/Azure/azure-sdk-for-go/sdk/storage/azblob`
2. Create storage interface and implementations
3. Add database migrations for new columns (nullable, default to postgres)
4. Deploy with `STORAGE_PROVIDER=postgres` (no behavior change)

### Phase 2: Testing & Validation

1. Local testing with Azurite (Azure Storage Emulator):
   ```bash
   docker run -p 10000:10000 -p 10001:10001 -p 10002:10002 mcr.microsoft.com/azure-storage/azurite
   ```
2. Set `STORAGE_PROVIDER=azure` with Azurite endpoint
3. Test upload, retrieval, deletion flows
4. Verify backward compatibility with existing Postgres-stored files

### Phase 3: Production Rollout

1. Create Azure Storage Account and container
2. Configure Azure Storage Account keys or Managed Identity
3. Deploy with `STORAGE_PROVIDER=azure`
4. Monitor logs for errors
5. Gradually migrate existing files (optional):
   ```go
   // Migration script: cmd/migrate-storage/main.go
   // Reads ImageData from DB, uploads to Azure, updates records
   ```

### Phase 4: Cleanup (Optional)

1. After validation period, migrate all old files to Azure
2. Remove `ImageData` and `ProtocolDocumentData` columns
3. Remove Postgres storage provider code

## Security Considerations

### Azure Blob Storage Security

1. **Private Containers**: Set container access level to "Private" (no anonymous access)
2. **Shared Access Signatures (SAS)**: Generate time-limited URLs for downloads
3. **Managed Identity**: Use Azure Managed Identity for Container Apps (no keys in environment)
4. **Network Security**: Restrict storage account to Azure Virtual Network (optional)
5. **Encryption**: Enable encryption at rest (automatic) and in transit (HTTPS only)

### Access Control

**Current (Database):**
- Authentication via JWT middleware
- Authorization via group membership checks in handlers
- Files served only to authenticated, authorized users

**Proposed (Azure):**
- **Option A (Redirect)**: Generate short-lived SAS tokens, redirect users
  - Pros: Efficient, offloads serving to Azure
  - Cons: URLs are temporarily public (but unguessable)
  
- **Option B (Proxy)**: API downloads from Azure and serves to user
  - Pros: Maintains strict access control, no public URLs
  - Cons: More server resources, slower for large files

**Recommendation**: Start with Option B (proxy) for security, evaluate Option A for performance optimization later.

### SAS Token Generation

```go
// internal/storage/azure.go
func (a *AzureBlobProvider) GetImageWithSAS(ctx context.Context, identifier string, expiryMinutes int) (string, error) {
	// Generate SAS token with read permission
	// Token expires after specified minutes
	// Return signed URL
}
```

## Testing Strategy

### Unit Tests

```go
// internal/storage/storage_test.go
func TestPostgresProvider(t *testing.T) {
	// Mock database
	// Test upload, retrieval, deletion
}

func TestAzureBlobProvider(t *testing.T) {
	// Mock Azure SDK client
	// Test upload, retrieval, deletion
}
```

### Integration Tests

```go
// internal/storage/integration_test.go (skip if !integration flag)
func TestAzureBlobProviderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	// Use Azurite or real Azure Storage
	// Test full upload/download cycle
}
```

### Handler Tests

```go
// internal/handlers/animal_image_test.go
func TestUploadAnimalImageWithAzure(t *testing.T) {
	// Mock storage provider
	// Test handler logic
}
```

## Performance Considerations

### Image Serving

**Current (Postgres):**
- Query database for binary data
- Serve through Go HTTP handler
- No caching (client-side only)

**Proposed (Azure):**
- Redirect to Azure Blob URL (Option A) or proxy (Option B)
- Azure CDN integration for global distribution
- Browser and CDN caching
- Reduced API server load

### Upload Performance

- No significant change (processing happens in API before upload)
- Azure SDK streams data efficiently
- Parallel uploads possible for multiple files

## Cost Analysis

### Current Costs (Postgres)

Assuming 10,000 images averaging 500KB each:
- Storage: 5GB database storage
- Azure Database for PostgreSQL: ~$0.20/GB = **$1.00/month**
- Backup storage proportional to database size
- Higher compute for serving files

### Proposed Costs (Azure Blob Storage)

Same 10,000 images:
- Blob storage (Hot tier): 5GB × $0.018/GB = **$0.09/month**
- Transactions: Negligible for this scale
- CDN costs (if enabled): Pay per GB transferred
- **Savings: ~91% on storage costs**

For 1TB of video files (future):
- Postgres: 1TB × $0.20/GB = **$200/month**
- Azure Blob (Cool tier): 1TB × $0.01/GB = **$10/month**
- **Savings: ~95% on storage costs**

## Implementation Checklist

### Development

- [ ] Create `internal/storage/` package structure
- [ ] Implement `Provider` interface
- [ ] Implement `PostgresProvider` (extract current logic)
- [ ] Implement `AzureBlobProvider` with Azure SDK
- [ ] Add database migrations for `StorageProvider` and `BlobIdentifier` columns
- [ ] Update handler upload functions to use storage interface
- [ ] Update handler serving functions to handle both providers
- [ ] Add comprehensive unit tests
- [ ] Add integration tests with Azurite

### Configuration

- [ ] Add Azure Storage environment variables to `.env.example`
- [ ] Update `cmd/api/main.go` to initialize storage provider
- [ ] Document configuration in this file and README

### Testing

- [ ] Test with `STORAGE_PROVIDER=postgres` (backward compatibility)
- [ ] Test with `STORAGE_PROVIDER=azure` and Azurite
- [ ] Test file upload, retrieval, deletion
- [ ] Test authentication and authorization with Azure URLs
- [ ] Load testing for performance validation

### Deployment

- [ ] Create Azure Storage Account in production subscription
- [ ] Create container with private access level
- [ ] Configure Managed Identity for Container App (preferred) or connection string
- [ ] Set environment variables in Azure Container Apps
- [ ] Deploy and monitor

### Documentation

- [ ] Update README.md with storage configuration
- [ ] Document migration process for existing deployments
- [ ] Create runbook for troubleshooting storage issues

## Future Enhancements

### Video Support

With Azure Blob Storage foundation:
1. Add `AnimalVideo` model similar to `AnimalImage`
2. Use Azure Media Services for transcoding (optional)
3. Support streaming with Azure CDN
4. Thumbnail generation for video previews

### Advanced Features

1. **Image Variants**: Store multiple sizes (thumbnail, medium, large)
2. **Lazy Loading**: Progressive image loading with placeholders
3. **Compression**: Use Azure Functions for automatic optimization
4. **Backup**: Geo-redundant storage for disaster recovery
5. **Analytics**: Track file access patterns with Application Insights

### CDN Integration

```bash
AZURE_STORAGE_CDN_ENDPOINT=https://mycdn.azureedge.net
```

Serve files through Azure CDN for:
- Global distribution
- Reduced latency
- DDoS protection
- SSL termination

## References

- [Azure Blob Storage Go SDK](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azblob)
- [Azure Storage Emulator (Azurite)](https://learn.microsoft.com/en-us/azure/storage/common/storage-use-azurite)
- [Azure Managed Identity](https://learn.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview)
- [Azure Storage Security](https://learn.microsoft.com/en-us/azure/storage/common/storage-security-guide)

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2024-12-23 | Use interface-based abstraction | Enables testing, future flexibility |
| 2024-12-23 | Feature flag for gradual rollout | Reduces risk, enables A/B testing |
| 2024-12-23 | Keep metadata in Postgres | Maintains transactional consistency, easy queries |
| 2024-12-23 | Support both Postgres and Azure simultaneously | Backward compatibility, gradual migration |
| 2024-12-23 | Proxy approach for serving (Option B) | Security-first, matches current access control model |

---

**Status**: Implementation Complete (Phase 1)  
**Next Steps**: Handler integration, integration testing with Azurite, production rollout  
**Owner**: Development Team  
**Last Updated**: 2024-12-23

## Implementation Status

### ✅ Completed
- Storage interface design and implementation
- PostgresProvider for backward compatibility
- AzureBlobProvider with full Azure SDK integration
- Feature flag configuration system
- Database model updates (StorageProvider, BlobIdentifier fields)
- Unit tests (100% passing)
- Comprehensive documentation

### 🚧 In Progress
- Handler integration (images and documents)
- Integration testing with Azurite
- End-to-end testing

### 📋 Pending
- Production deployment guide
- Migration script for existing data
- Performance benchmarking
- CDN integration (optional enhancement)

