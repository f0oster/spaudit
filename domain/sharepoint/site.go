package sharepoint

import (
	"time"
)

// Site represents a SharePoint site collection
type Site struct {
	ID        int64 // Auto-generated site ID for database
	URL       string
	Title     string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

// Web represents a SharePoint web/subsite
type Web struct {
	SiteID     int64 // Reference to parent site
	ID         string
	URL        string
	Title      string
	Template   string
	HasUnique  bool
	AuditRunID *int64
}

// List represents a SharePoint list or document library
type List struct {
	SiteID       int64 // Reference to parent site
	ID           string
	WebID        string
	Title        string
	URL          string
	BaseTemplate int
	ItemCount    int
	HasUnique    bool
	AuditRunID   *int64
}

// IsEmpty returns true if the list has no items
func (l *List) IsEmpty() bool {
	return l.ItemCount == 0
}

// IsDocumentLibrary returns true if this is a document library (BaseTemplate 101)
func (l *List) IsDocumentLibrary() bool {
	return l.BaseTemplate == 101
}

// IsCustomList returns true if this is a custom list (BaseTemplate 100)
func (l *List) IsCustomList() bool {
	return l.BaseTemplate == 100
}

// Item represents a SharePoint list item, file, or folder
type Item struct {
	SiteID       int64  // Reference to parent site
	GUID         string // File/Folder UniqueId (used in sharing links)
	ListItemGUID string // List Item GUID (ListItemAllFields.GUID)
	ListID       string
	ID           int
	URL          string
	Name         string
	IsFile       bool
	IsFolder     bool
	HasUnique    bool
	AuditRunID   *int64
}

// IsDocument returns true if this is a file
func (i *Item) IsDocument() bool {
	return i.IsFile
}

// IsDirectory returns true if this is a folder
func (i *Item) IsDirectory() bool {
	return i.IsFolder
}

// IsListItem returns true if this is neither a file nor folder (regular list item)
func (i *Item) IsListItem() bool {
	return !i.IsFile && !i.IsFolder
}

// GetDisplayName returns a user-friendly name for the item
func (i *Item) GetDisplayName() string {
	if i.Name != "" {
		return i.Name
	}
	return i.GUID // Fallback to GUID if no name
}

// FileSystemObjectType constants
const (
	FileSystemObjectTypeFile   = 0
	FileSystemObjectTypeFolder = 1
	FileSystemObjectTypeItem   = 2
)
