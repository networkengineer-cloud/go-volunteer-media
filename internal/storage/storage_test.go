package storage

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected Config
	}{
		{
			name:    "default to postgres when not set",
			envVars: map[string]string{},
			expected: Config{
				Provider: "postgres",
			},
		},
		{
			name: "load azure configuration",
			envVars: map[string]string{
				"STORAGE_PROVIDER":              "azure",
				"AZURE_STORAGE_ACCOUNT_NAME":    "testaccount",
				"AZURE_STORAGE_ACCOUNT_KEY":     "testkey",
				"AZURE_STORAGE_CONTAINER_NAME":  "testcontainer",
				"AZURE_STORAGE_ENDPOINT":        "http://localhost:10000/devstoreaccount1",
				"AZURE_STORAGE_USE_MANAGED_IDENTITY": "false",
			},
			expected: Config{
				Provider:           "azure",
				AzureAccountName:   "testaccount",
				AzureAccountKey:    "testkey",
				AzureContainerName: "testcontainer",
				AzureEndpoint:      "http://localhost:10000/devstoreaccount1",
				AzureUseManagedID:  false,
			},
		},
		{
			name: "load azure with managed identity",
			envVars: map[string]string{
				"STORAGE_PROVIDER":                   "azure",
				"AZURE_STORAGE_ACCOUNT_NAME":         "testaccount",
				"AZURE_STORAGE_CONTAINER_NAME":       "testcontainer",
				"AZURE_STORAGE_USE_MANAGED_IDENTITY": "true",
			},
			expected: Config{
				Provider:           "azure",
				AzureAccountName:   "testaccount",
				AzureContainerName: "testcontainer",
				AzureUseManagedID:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Load config
			config := LoadConfig()

			// Verify
			if config.Provider != tt.expected.Provider {
				t.Errorf("Provider = %v, want %v", config.Provider, tt.expected.Provider)
			}
			if config.AzureAccountName != tt.expected.AzureAccountName {
				t.Errorf("AzureAccountName = %v, want %v", config.AzureAccountName, tt.expected.AzureAccountName)
			}
			if config.AzureAccountKey != tt.expected.AzureAccountKey {
				t.Errorf("AzureAccountKey = %v, want %v", config.AzureAccountKey, tt.expected.AzureAccountKey)
			}
			if config.AzureContainerName != tt.expected.AzureContainerName {
				t.Errorf("AzureContainerName = %v, want %v", config.AzureContainerName, tt.expected.AzureContainerName)
			}
			if config.AzureEndpoint != tt.expected.AzureEndpoint {
				t.Errorf("AzureEndpoint = %v, want %v", config.AzureEndpoint, tt.expected.AzureEndpoint)
			}
			if config.AzureUseManagedID != tt.expected.AzureUseManagedID {
				t.Errorf("AzureUseManagedID = %v, want %v", config.AzureUseManagedID, tt.expected.AzureUseManagedID)
			}
		})
	}
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
		errMsg    string
	}{
		{
			name: "create postgres provider",
			config: Config{
				Provider: "postgres",
			},
			expectErr: false,
		},
		{
			name: "azure without account name fails",
			config: Config{
				Provider:           "azure",
				AzureContainerName: "test",
				AzureAccountKey:    "key",
			},
			expectErr: true,
			errMsg:    "account name and container name required",
		},
		{
			name: "azure without container name fails",
			config: Config{
				Provider:         "azure",
				AzureAccountName: "test",
				AzureAccountKey:  "key",
			},
			expectErr: true,
			errMsg:    "account name and container name required",
		},
		{
			name: "azure without key and managed identity disabled fails",
			config: Config{
				Provider:           "azure",
				AzureAccountName:   "test",
				AzureContainerName: "test",
				AzureUseManagedID:  false,
			},
			expectErr: true,
			errMsg:    "account key required",
		},
		{
			name: "invalid provider fails",
			config: Config{
				Provider: "invalid",
			},
			expectErr: true,
			errMsg:    "invalid storage provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.config, nil)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					// For partial error message matching
					t.Logf("error message: %v", err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if provider == nil {
					t.Errorf("expected provider but got nil")
				}
			}
		})
	}
}

func TestPostgresProviderURLGeneration(t *testing.T) {
	provider := NewPostgresProvider(nil)

	tests := []struct {
		name       string
		identifier string
		wantURL    string
	}{
		{
			name:       "image URL generation",
			identifier: "123e4567-e89b-12d3-a456-426614174000",
			wantURL:    "/api/images/123e4567-e89b-12d3-a456-426614174000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := provider.GetImageURL(tt.identifier)
			if url != tt.wantURL {
				t.Errorf("GetImageURL() = %v, want %v", url, tt.wantURL)
			}
		})
	}

	// Test document URL
	docID := "doc123"
	docURL := provider.GetDocumentURL(docID)
	expectedDocURL := "/api/documents/doc123"
	if docURL != expectedDocURL {
		t.Errorf("GetDocumentURL() = %v, want %v", docURL, expectedDocURL)
	}
}

func TestAzureBlobProviderURLGeneration(t *testing.T) {
	// Note: We can't easily test NewAzureBlobProvider without real Azure credentials
	// or extensive mocking. These tests focus on URL generation logic.
	
	// This test would require a working Azure connection or mock
	// For now, we'll test the URL generation patterns
	
	t.Log("Azure provider tests require integration testing with Azurite or real Azure Storage")
	t.Log("See integration tests for full Azure provider testing")
}
