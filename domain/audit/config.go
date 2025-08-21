package audit

import (
	"fmt"
)

// AuditParameters represents user-configurable audit behavior and preferences.
// This is a domain value object that encapsulates business rules for audit execution.
type AuditParameters struct {
	// Audit scope and behavior
	ScanIndividualItems bool // Whether to perform deep scanning of individual documents/folders within lists
	SkipHidden          bool // Skip hidden lists and items
	IncludeSharing      bool // Whether to include comprehensive sharing audit

	// Performance parameters
	BatchSize  int // User-preferred batch size for API calls
	MaxRetries int // Maximum retry attempts for failed operations
	RetryDelay int // Delay between retries in milliseconds
	Timeout    int // Overall audit timeout in seconds
}

// DefaultParameters returns sensible default audit parameters.
func DefaultParameters() *AuditParameters {
	return &AuditParameters{
		ScanIndividualItems: true,
		SkipHidden:          true,
		IncludeSharing:      true, // Enable comprehensive sharing audit by default
		BatchSize:           100,  // Standard default batch size
		MaxRetries:          3,
		RetryDelay:          1000, // 1 second
		Timeout:             1800, // 30 minutes
	}
}

// SharePointApiConstraints defines the technical limits imposed by SharePoint APIs.
// These are infrastructure concerns, not user preferences.
type SharePointApiConstraints struct {
	MinBatchSize  int // Minimum valid batch size (1)
	MaxBatchSize  int // SharePoint REST API limit (5000)
	MinTimeout    int // Minimum timeout for SharePoint operations (60 seconds)
	MaxTimeout    int // Maximum reasonable timeout (2 hours)
	MaxRetries    int // Maximum retry attempts (10)
	MaxRetryDelay int // Maximum retry delay (60 seconds)
}

// DefaultApiConstraints returns SharePoint API technical limits.
func DefaultApiConstraints() *SharePointApiConstraints {
	return &SharePointApiConstraints{
		MinBatchSize:  1,
		MaxBatchSize:  5000, // SharePoint REST API limit
		MinTimeout:    60,   // 1 minute minimum
		MaxTimeout:    7200, // 2 hours maximum
		MaxRetries:    10,
		MaxRetryDelay: 60000, // 60 seconds
	}
}

// Validate checks the audit parameters against SharePoint API constraints.
func (p *AuditParameters) Validate(constraints *SharePointApiConstraints) error {
	if p == nil {
		return fmt.Errorf("audit parameters cannot be nil")
	}
	if constraints == nil {
		constraints = DefaultApiConstraints()
	}

	// Validate BatchSize using API constraints
	if p.BatchSize < constraints.MinBatchSize {
		return fmt.Errorf("batch_size must be at least %d, got: %d", constraints.MinBatchSize, p.BatchSize)
	}
	if p.BatchSize > constraints.MaxBatchSize {
		return fmt.Errorf("batch_size cannot exceed %d (SharePoint API limit), got: %d", constraints.MaxBatchSize, p.BatchSize)
	}

	// Validate MaxRetries
	if p.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative, got: %d", p.MaxRetries)
	}
	if p.MaxRetries > constraints.MaxRetries {
		return fmt.Errorf("max_retries cannot exceed %d (too many retries), got: %d", constraints.MaxRetries, p.MaxRetries)
	}

	// Validate RetryDelay
	if p.RetryDelay < 0 {
		return fmt.Errorf("retry_delay cannot be negative, got: %d ms", p.RetryDelay)
	}
	if p.RetryDelay > constraints.MaxRetryDelay {
		return fmt.Errorf("retry_delay cannot exceed %d ms, got: %d ms", constraints.MaxRetryDelay, p.RetryDelay)
	}

	// Validate Timeout
	if p.Timeout < constraints.MinTimeout {
		return fmt.Errorf("timeout must be at least %d seconds for SharePoint operations, got: %d seconds", constraints.MinTimeout, p.Timeout)
	}
	if p.Timeout > constraints.MaxTimeout {
		return fmt.Errorf("timeout cannot exceed %d seconds, got: %d seconds", constraints.MaxTimeout, p.Timeout)
	}

	return nil
}

// ValidateAndSetDefaults validates parameters and sets defaults for zero values, then validates against constraints.
func (p *AuditParameters) ValidateAndSetDefaults(constraints *SharePointApiConstraints) error {
	if p == nil {
		return fmt.Errorf("audit parameters cannot be nil")
	}
	if constraints == nil {
		constraints = DefaultApiConstraints()
	}

	// Set defaults for zero values
	if p.BatchSize == 0 {
		p.BatchSize = 100 // Standard default
	}
	if p.MaxRetries == 0 {
		p.MaxRetries = 3
	}
	if p.RetryDelay == 0 {
		p.RetryDelay = 1000
	}
	if p.Timeout == 0 {
		p.Timeout = 1800
	}

	// Now validate with defaults applied
	return p.Validate(constraints)
}

// SetBatchSize sets the batch size with automatic clamping to valid limits.
// This is the preferred way for users to specify batch sizes as it handles validation.
func (p *AuditParameters) SetBatchSize(batchSize int, constraints *SharePointApiConstraints) {
	if constraints == nil {
		constraints = DefaultApiConstraints()
	}

	if batchSize < constraints.MinBatchSize {
		p.BatchSize = constraints.MinBatchSize
	} else if batchSize > constraints.MaxBatchSize {
		p.BatchSize = constraints.MaxBatchSize
	} else {
		p.BatchSize = batchSize
	}
}

// GetEffectiveBatchSize returns the batch size to use, with fallback to default if not set
func (p *AuditParameters) GetEffectiveBatchSize() int {
	if p.BatchSize <= 0 {
		return 100 // Standard default
	}
	return p.BatchSize
}
