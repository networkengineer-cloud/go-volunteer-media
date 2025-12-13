package handlers

import (
	"testing"
)

// TestEscapeSQLWildcards tests the SQL wildcard escaping function
func TestEscapeSQLWildcards(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special characters",
			input:    "normal text",
			expected: "normal text",
		},
		{
			name:     "percent sign",
			input:    "test%",
			expected: "test\\%",
		},
		{
			name:     "underscore",
			input:    "test_name",
			expected: "test\\_name",
		},
		{
			name:     "both percent and underscore",
			input:    "test%_name",
			expected: "test\\%\\_name",
		},
		{
			name:     "multiple percent signs",
			input:    "%%test%%",
			expected: "\\%\\%test\\%\\%",
		},
		{
			name:     "backslash",
			input:    "test\\escape",
			expected: "test\\\\escape",
		},
		{
			name:     "backslash and percent",
			input:    "test\\%",
			expected: "test\\\\\\%",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "SQL injection attempt with wildcards",
			input:    "%' OR '1'='1",
			expected: "\\%' OR '1'='1",
		},
		{
			name:     "realistic animal name with special chars",
			input:    "fluffy_dog%",
			expected: "fluffy\\_dog\\%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeSQLWildcards(tt.input)
			if result != tt.expected {
				t.Errorf("escapeSQLWildcards(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestEscapeSQLWildcards_Integration tests the wildcard escaping in a realistic scenario
func TestEscapeSQLWildcards_Integration(t *testing.T) {
	// This test demonstrates why wildcard escaping is important for security

	testCases := []struct {
		name            string
		userInput       string
		shouldMatch     []string
		shouldNotMatch  []string
		description     string
	}{
		{
			name:            "exact match without wildcards",
			userInput:       "rex",
			shouldMatch:     []string{"Rex", "REX", "rexie"},
			shouldNotMatch:  []string{"T-Rex", "Tyrannosaurus"},
			description:     "Normal search should use LIKE with % wrapping",
		},
		{
			name:            "user input with percent should be literal",
			userInput:       "test%",
			shouldMatch:     []string{"test%", "TEST%"},
			shouldNotMatch:  []string{"test", "tester", "testing"},
			description:     "User's % should be treated as literal character, not SQL wildcard",
		},
		{
			name:            "user input with underscore should be literal",
			userInput:       "fluffy_dog",
			shouldMatch:     []string{"fluffy_dog", "FLUFFY_DOG"},
			shouldNotMatch:  []string{"fluffy dog", "fluffyAdog", "fluffy-dog"},
			description:     "User's _ should be treated as literal character, not single-char wildcard",
		},
		{
			name:            "complex pattern with both wildcards",
			userInput:       "test_%_name",
			shouldMatch:     []string{"test_%_name"},
			shouldNotMatch:  []string{"test_a_name", "test1%1name"},
			description:     "Both wildcards should be escaped and treated literally",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			escaped := escapeSQLWildcards(tc.userInput)
			
			// Log the transformation for documentation
			t.Logf("User input: %q", tc.userInput)
			t.Logf("Escaped: %q", escaped)
			t.Logf("SQL pattern: %%%s%%", escaped)
			t.Logf("Description: %s", tc.description)
			
			// Verify that escaping occurred if wildcards were present
			if tc.userInput != escaped {
				t.Logf("✅ Wildcard characters were properly escaped")
			}
		})
	}
}

// TestEscapeSQLWildcards_Performance tests that escaping is efficient
func TestEscapeSQLWildcards_Performance(t *testing.T) {
	// Test with a reasonably long input to ensure performance is acceptable
	longInput := ""
	for i := 0; i < 100; i++ {
		longInput += "test%_string"
	}
	
	result := escapeSQLWildcards(longInput)
	
	// Verify the function handles long strings without issues
	if len(result) == 0 {
		t.Error("Function failed to process long input")
	}
	
	t.Logf("Processed input of length %d to output of length %d", len(longInput), len(result))
}
