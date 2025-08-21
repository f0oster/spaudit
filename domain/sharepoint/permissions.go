package sharepoint

// Principal represents a user, group, or security principal
type Principal struct {
	SiteID        int64 // Reference to parent site
	ID            int64
	PrincipalType int64
	Title         string
	LoginName     string
	Email         string
}

// IsUser returns true if this is a user principal
func (p *Principal) IsUser() bool {
	return p.PrincipalType == PrincipalTypeUser
}

// IsGroup returns true if this is a group principal
func (p *Principal) IsGroup() bool {
	return p.PrincipalType == PrincipalTypeSecurity || p.PrincipalType == PrincipalTypeDistribution
}

// IsSharePointGroup returns true if this is a SharePoint group
func (p *Principal) IsSharePointGroup() bool {
	return p.PrincipalType == PrincipalTypeSharePointGroup
}

// GetDisplayName returns the best display name for the principal
func (p *Principal) GetDisplayName() string {
	if p.Title != "" {
		return p.Title
	}
	if p.LoginName != "" {
		return p.LoginName
	}
	return p.Email
}

// RoleDefinition represents a SharePoint permission level
type RoleDefinition struct {
	SiteID      int64 // Reference to parent site
	ID          int64
	Name        string
	Description string
}

// RoleAssignment represents a permission assignment to an object
type RoleAssignment struct {
	SiteID      int64  // Reference to parent site
	ObjectType  string // "web", "list", "item"
	ObjectKey   string // web ID, list ID, or item GUID
	PrincipalID int64
	RoleDefID   int64
	Inherited   bool
}

// Assignment represents a complete assignment with principal and role info
type Assignment struct {
	RoleAssignment *RoleAssignment
	Principal      *Principal
	RoleDefinition *RoleDefinition
}

// IsInherited returns true if this assignment is inherited
func (a *Assignment) IsInherited() bool {
	return a.RoleAssignment.Inherited
}

// IsDirectAssignment returns true if this assignment is direct (not inherited)
func (a *Assignment) IsDirectAssignment() bool {
	return !a.RoleAssignment.Inherited
}

// IsWebAssignment returns true if this assignment is on a web
func (a *Assignment) IsWebAssignment() bool {
	return a.RoleAssignment.ObjectType == ObjectTypeWeb
}

// IsListAssignment returns true if this assignment is on a list
func (a *Assignment) IsListAssignment() bool {
	return a.RoleAssignment.ObjectType == ObjectTypeList
}

// IsItemAssignment returns true if this assignment is on an item
func (a *Assignment) IsItemAssignment() bool {
	return a.RoleAssignment.ObjectType == ObjectTypeItem
}

// ObjectType constants for role assignments
const (
	ObjectTypeWeb  = "web"
	ObjectTypeList = "list"
	ObjectTypeItem = "item"
)

// RootCause represents a single source of permission access
type RootCause struct {
	Type         string // "SHARING_LINK", "SAME_WEB_INHERITANCE", "SYSTEM_GROUP", "UNKNOWN"
	Detail       string // Detailed explanation of this root cause
	SourceObject string // The object that caused this assignment
	SourceRole   string // The role that caused this assignment
}

// ResolvedAssignment represents an assignment with root cause analysis
type ResolvedAssignment struct {
	Assignment *Assignment
	RootCauses []RootCause // All detected permission sources
}

// Root cause type constants
const (
	RootCauseTypeSharingLink = "SHARING_LINK"
	RootCauseTypeInheritance = "SAME_WEB_INHERITANCE"
	RootCauseTypeSystemGroup = "SYSTEM_GROUP"
	RootCauseTypeUnknown     = "UNKNOWN"
)

// Common SharePoint principal types
const (
	PrincipalTypeUser            = 1
	PrincipalTypeDistribution    = 2
	PrincipalTypeSecurity        = 4
	PrincipalTypeSharePointGroup = 8
	PrincipalTypeAll             = 15
)
