package application

import (
	"net/url"
	"os"
	"testing"

	"spaudit/domain/audit"

	"github.com/stretchr/testify/assert"
)

func TestAuditServiceImpl_BuildAuditParametersFromFormData(t *testing.T) {
	// Clear environment variables to ensure predictable defaults
	originalEnvVars := map[string]string{
		"SP_AUDIT_TIMEOUT":    os.Getenv("SP_AUDIT_TIMEOUT"),
		"SP_AUDIT_BATCH_SIZE": os.Getenv("SP_AUDIT_BATCH_SIZE"),
	}
	defer func() {
		// Restore original environment variables
		for key, value := range originalEnvVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Clear environment variables for consistent testing
	os.Unsetenv("SP_AUDIT_TIMEOUT")
	os.Unsetenv("SP_AUDIT_BATCH_SIZE")

	tests := []struct {
		name     string
		formData map[string][]string
		expected func(*audit.AuditParameters) // function to verify expected values
	}{
		{
			name: "timeout and batch_size from form data",
			formData: map[string][]string{
				"timeout":    {"300"},
				"batch_size": {"50"},
			},
			expected: func(parameters *audit.AuditParameters) {
				assert.Equal(t, 300, parameters.Timeout)
				assert.Equal(t, 50, parameters.BatchSize)
				// Check that defaults are used for other values
				assert.True(t, parameters.ScanIndividualItems) // default
				assert.True(t, parameters.SkipHidden)          // default
			},
		},
		{
			name: "boolean checkboxes and numeric values",
			formData: map[string][]string{
				"scan_individual_items": {"on"},
				"skip_hidden":           {"on"},
				"include_sharing":       {"on"},
				"timeout":               {"600"},
				"batch_size":            {"200"},
			},
			expected: func(parameters *audit.AuditParameters) {
				assert.True(t, parameters.ScanIndividualItems)
				assert.True(t, parameters.SkipHidden)
				assert.True(t, parameters.IncludeSharing)
				assert.Equal(t, 600, parameters.Timeout)
				assert.Equal(t, 200, parameters.BatchSize)
			},
		},
		{
			name: "unchecked checkboxes explicitly present",
			formData: map[string][]string{
				"scan_individual_items": {""},
				"skip_hidden":           {""},
				"include_sharing":       {""},
				"timeout":               {"120"},
			},
			expected: func(parameters *audit.AuditParameters) {
				assert.False(t, parameters.ScanIndividualItems)
				assert.False(t, parameters.SkipHidden)
				assert.False(t, parameters.IncludeSharing)
				assert.Equal(t, 120, parameters.Timeout)
			},
		},
		{
			name: "empty or invalid numeric values use defaults",
			formData: map[string][]string{
				"timeout":    {""},
				"batch_size": {"invalid"},
			},
			expected: func(parameters *audit.AuditParameters) {
				// Should use default parameters (from DefaultParameters)
				assert.Equal(t, 1800, parameters.Timeout)  // default timeout
				assert.Equal(t, 100, parameters.BatchSize) // default batch size
			},
		},
		{
			name: "zero values are ignored",
			formData: map[string][]string{
				"timeout":    {"0"},
				"batch_size": {"0"},
			},
			expected: func(parameters *audit.AuditParameters) {
				// Should use default parameters since 0 values are ignored
				assert.Equal(t, 1800, parameters.Timeout)
				assert.Equal(t, 100, parameters.BatchSize)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create service instance
			service := &AuditServiceImpl{}

			parameters := service.BuildAuditParametersFromFormData(tt.formData)

			assert.NotNil(t, parameters)
			tt.expected(parameters)
		})
	}
}

func TestAuditServiceImpl_BuildAuditParametersFromFormData_URLValues(t *testing.T) {
	// Test with url.Values (which is what http.Request.Form provides)
	formValues := url.Values{}
	formValues.Set("timeout", "300")
	formValues.Set("batch_size", "75")
	formValues.Set("scan_individual_items", "on")

	service := &AuditServiceImpl{}

	// Convert url.Values to map[string][]string for testing
	parameters := service.BuildAuditParametersFromFormData(map[string][]string(formValues))

	assert.Equal(t, 300, parameters.Timeout)
	assert.Equal(t, 75, parameters.BatchSize)
	assert.True(t, parameters.ScanIndividualItems)
}
