package audit

// ProgressReporter defines the interface for reporting audit progress.
type ProgressReporter interface {
	// ReportProgress reports the current stage of the audit.
	ReportProgress(stage, description string, percentage int)

	// ReportItemProgress reports progress with item counts.
	ReportItemProgress(stage, description string, percentage, itemsDone, itemsTotal int)
}

// NoOpProgressReporter is a no-op implementation for when progress reporting is not needed.
type NoOpProgressReporter struct{}

func (n *NoOpProgressReporter) ReportProgress(stage, description string, percentage int) {
	// No operation
}

func (n *NoOpProgressReporter) ReportItemProgress(stage, description string, percentage, itemsDone, itemsTotal int) {
	// No operation
}

// NewNoOpProgressReporter creates a new no-op progress reporter.
func NewNoOpProgressReporter() ProgressReporter {
	return &NoOpProgressReporter{}
}

// ProgressStages defines standard progress stages for consistency.
type ProgressStages struct {
	Authentication string
	WebDiscovery   string
	ListDiscovery  string
	ListProcessing string
	ItemProcessing string
	Permissions    string
	Sharing        string
	Finalization   string
}

// StandardStages provides consistent stage names.
var StandardStages = ProgressStages{
	Authentication: "Authentication",
	WebDiscovery:   "Web Discovery",
	ListDiscovery:  "List Discovery",
	ListProcessing: "List Processing",
	ItemProcessing: "Item Processing",
	Permissions:    "Permissions",
	Sharing:        "Sharing Analysis",
	Finalization:   "Finalization",
}
