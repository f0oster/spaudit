package sharepoint

// DiscoveredSharingLink represents any type of discovered sharing link from principals table
type DiscoveredSharingLink struct {
	PrincipalID int64
	LoginName   string
	ItemGUID    string
	SharingID   string
	LinkType    string // Flexible, OrganizationView, OrganizationEdit, etc.
}

// FlexibleSharingReference represents a discovered flexible sharing link reference
type FlexibleSharingReference struct {
	PrincipalID int64
	LoginName   string
	ItemGUID    string
	SharingID   string
}
