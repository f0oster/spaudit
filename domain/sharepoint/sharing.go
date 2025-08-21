package sharepoint

import (
	"time"
)

// Relevant reading:
// https://reshmeeauckloo.com/posts/powershell-get-sharing-links-sharepoint/

// SharingInfo represents the complete sharing information for an item
type SharingInfo struct {
	SiteID                     int64 // Reference to parent site
	DisplayName                string
	ItemUniqueID               string
	WebURL                     string
	DirectURL                  string
	FileExtension              string
	HasUniquePermissions       bool
	DefaultLinkKind            int
	DefaultShareLinkPermission int
	DefaultShareLinkScope      int
	Links                      []*SharingLink
	Principals                 []*PrincipalInfo
	SiteAdmins                 []*PrincipalInfo

	// Tenant & Organization Context (Governance)
	TenantID          string
	TenantDisplayName string
	SharePointSiteID  string // SharePoint GUID (different from our database SiteID)

	// Security & Policy Configuration (Governance)
	AnonymousLinkExpirationRestrictionDays int
	AnyoneLinkTrackUsers                   bool
	CanAddExternalPrincipal                bool
	CanAddInternalPrincipal                bool
	BlockPeoplePickerAndSharing            bool
	CanRequestAccessForGrantAccess         bool

	// Information Barriers & Compliance (Governance)
	SiteIBMode                string   // Information Barriers mode ("Open", "Implicit", etc.)
	SiteIBSegmentIDs          []string // Information Barriers segment IDs
	EnforceIBSegmentFiltering bool

	// Sensitivity & Classification (Governance)
	SensitivityLabel *SensitivityLabelInformation

	// Sharing Capabilities Matrix (Governance)
	SharingAbilities *SharingAbilities

	// Recipient Limits (Security Controls)
	RecipientLimits *RecipientLimits
}

// SharingLink represents a sharing link on an item
type SharingLink struct {
	SiteID               int64  // Reference to parent site
	ID                   string // ShareID from SharePoint (natural key)
	ItemGUID             string // ListItem GUID (for database linking)
	FileFolderUniqueID   string // File/Folder UniqueId (used in sharing links)
	ShareID              string
	URL                  string
	LinkKind             int
	Scope                int
	IsActive             bool
	IsDefault            bool
	IsEditLink           bool
	IsReviewLink         bool
	BlocksDownload       bool
	RequiresPassword     bool
	RestrictedMembership bool
	IsInherited          bool
	InheritedFrom        string
	CreatedAt            *time.Time
	CreatedBy            *Principal
	LastModifiedAt       *time.Time
	LastModifiedBy       *Principal
	StatusEnabled        *bool
	StatusDisabledReason *int
	SharingLinkStatus    *int
	TotalMembersCount    int
	ShareToken           string
	Members              []*Principal
	AuditedAt            *time.Time

	// Enhanced Governance Fields
	Expiration                      *time.Time // Link expiration date
	PasswordLastModified            *time.Time
	PasswordLastModifiedBy          *Principal
	HasExternalGuestInvitees        bool // External user involvement
	TrackLinkUsers                  bool // User tracking enabled
	IsEphemeral                     bool // Temporary/ephemeral link
	IsUnhealthy                     bool // Link health status
	IsAddressBarLink                bool // Address bar link
	IsCreateOnlyLink                bool // Create-only permission link
	IsFormsLink                     bool // Forms submission link
	IsMainLink                      bool // Main sharing link
	IsManageListLink                bool // List management link
	AllowsAnonymousAccess           bool // Allows anonymous access
	Embeddable                      bool // Can be embedded
	LimitUseToApplication           bool // Restricted to specific app
	RestrictToExistingRelationships bool // Only existing relationships
}

// IsAnonymousLink returns true if this is an anonymous sharing link
func (s *SharingLink) IsAnonymousLink() bool {
	return s.Scope == ScopeAnonymous
}

// IsInternalLink returns true if this is an organization-only link
func (s *SharingLink) IsInternalLink() bool {
	return s.Scope == ScopeOrganization
}

// IsSpecificPeopleLink returns true if this is a specific people link
func (s *SharingLink) IsSpecificPeopleLink() bool {
	return s.Scope == ScopeSpecificPeople
}

// IsViewOnlyLink returns true if this is a view-only link
func (s *SharingLink) IsViewOnlyLink() bool {
	return !s.IsEditLink && !s.IsReviewLink
}

// IsSecure returns true if this link has security restrictions
func (s *SharingLink) IsSecure() bool {
	return s.RequiresPassword || s.RestrictedMembership || s.BlocksDownload
}

// HasMembers returns true if this link has members assigned
func (s *SharingLink) HasMembers() bool {
	return s.TotalMembersCount > 0
}

// GetLinkKindName returns a human-readable name for the link kind
func (s *SharingLink) GetLinkKindName() string {
	switch s.LinkKind {
	case LinkKindUninitialized:
		return "Uninitialized"
	case LinkKindDirect:
		return "Direct"
	case LinkKindOrganizationView:
		return "Organization View"
	case LinkKindOrganizationEdit:
		return "Organization Edit"
	case LinkKindAnonymousView:
		return "Anonymous View"
	case LinkKindAnonymousEdit:
		return "Anonymous Edit"
	case LinkKindFlexible:
		return "Flexible"
	default:
		return "Unknown"
	}
}

// GetScopeName returns a human-readable name for the scope
func (s *SharingLink) GetScopeName() string {
	switch s.Scope {
	case ScopeNotApplicable:
		return "Not Applicable"
	case ScopeAnonymous:
		return "Anonymous"
	case ScopeOrganization:
		return "Organization"
	case ScopeSpecificPeople:
		return "Specific People"
	case ScopeExistingAccess:
		return "Existing Access"
	default:
		return "Unknown"
	}
}

// PrincipalInfo represents a principal with role and inheritance info
type PrincipalInfo struct {
	Principal   *Principal
	Role        int
	IsInherited bool
}

// Common sharing link kinds (SP.SharingLinkKind)
// https://learn.microsoft.com/en-us/dotnet/api/microsoft.sharepoint.client.sharinglinkkind?view=sharepoint-csom
const (
	LinkKindUninitialized    = 0
	LinkKindDirect           = 1
	LinkKindOrganizationView = 2
	LinkKindOrganizationEdit = 3
	LinkKindAnonymousView    = 4
	LinkKindAnonymousEdit    = 5
	LinkKindFlexible         = 6
)

// Common sharing scopes
// https://learn.microsoft.com/en-us/graph/api/resources/sharinglink?view=graph-rest-1.0
// https://learn.microsoft.com/en-us/sharepoint/change-default-sharing-link
// https://learn.microsoft.com/en-us/sharepoint/shareable-links-anyone-specific-people-organization
// Based on SharePoint API behavior observed in practice:
// anonymous = 0, organization = 1, specificPeople = 2, existingAccess = 3 (hypothetical)
const (
	ScopeNotApplicable  = -1 // For inactive/disabled links
	ScopeAnonymous      = 0  // Anyone with the link (may include external users)
	ScopeOrganization   = 1  // Anyone in your organization
	ScopeSpecificPeople = 2  // Specific list of people (active sharing links)
	ScopeExistingAccess = 3  // Only people who already have access (hypothetical)
)

// Common roles
const (
	RoleNone           = 0
	RoleView           = 1
	RoleEdit           = 2
	RoleOwner          = 3
	RoleLimitedView    = 4
	RoleLimitedEdit    = 5
	RoleReview         = 6
	RoleRestrictedView = 7
	RoleSubmit         = 8
	RoleManageList     = 9
)

// SharingLinkWithItemData represents a sharing link enriched with item information for UI display
type SharingLinkWithItemData struct {
	*SharingLink
	ItemName     string
	ItemIsFile   bool
	ItemIsFolder bool
}

// SensitivityLabelInformation represents sensitivity labeling information for governance
type SensitivityLabelInformation struct {
	ID                             string
	DisplayName                    string
	Color                          string
	Tooltip                        string
	HasIRMProtection               bool
	SensitivityLabelProtectionType string
}

// SharingAbilities represents the complete sharing capabilities matrix for governance
type SharingAbilities struct {
	CanStopSharing             bool
	AnonymousLinkAbilities     *SharingLinkAbilities
	AnyoneLinkAbilities        *SharingLinkAbilities
	OrganizationLinkAbilities  *SharingLinkAbilities
	PeopleSharingLinkAbilities *SharingLinkAbilities
	DirectSharingAbilities     *DirectSharingAbilities
}

// SharingLinkAbilities represents capabilities for a specific type of sharing link
type SharingLinkAbilities struct {
	CanAddNewExternalPrincipals             SharingAbilityStatus
	CanDeleteEditLink                       SharingAbilityStatus
	CanDeleteManageListLink                 SharingAbilityStatus
	CanDeleteReadLink                       SharingAbilityStatus
	CanDeleteRestrictedViewLink             SharingAbilityStatus
	CanDeleteReviewLink                     SharingAbilityStatus
	CanDeleteSubmitOnlyLink                 SharingAbilityStatus
	CanGetEditLink                          SharingAbilityStatus
	CanGetManageListLink                    SharingAbilityStatus
	CanGetReadLink                          SharingAbilityStatus
	CanGetRestrictedViewLink                SharingAbilityStatus
	CanGetReviewLink                        SharingAbilityStatus
	CanGetSubmitOnlyLink                    SharingAbilityStatus
	CanHaveExternalUsers                    SharingAbilityStatus
	CanManageEditLink                       SharingAbilityStatus
	CanManageManageListLink                 SharingAbilityStatus
	CanManageReadLink                       SharingAbilityStatus
	CanManageRestrictedViewLink             SharingAbilityStatus
	CanManageReviewLink                     SharingAbilityStatus
	CanManageSubmitOnlyLink                 SharingAbilityStatus
	LinkExpiration                          *SharingLinkExpirationAbilityStatus
	PasswordProtected                       *SharingLinkPasswordAbilityStatus
	SubmitOnlyLinkExpiration                *SharingLinkExpirationAbilityStatus
	SupportsRestrictedView                  SharingAbilityStatus
	SupportsRestrictToExistingRelationships SharingAbilityStatus
}

// DirectSharingAbilities represents capabilities for direct sharing (non-link based)
type DirectSharingAbilities struct {
	CanAddExternalPrincipal                           SharingAbilityStatus
	CanAddInternalPrincipal                           SharingAbilityStatus
	CanAddNewExternalPrincipal                        SharingAbilityStatus
	CanRequestGrantAccess                             SharingAbilityStatus
	CanRequestGrantAccessForExistingExternalPrincipal SharingAbilityStatus
	CanRequestGrantAccessForInternalPrincipal         SharingAbilityStatus
	CanRequestGrantAccessForNewExternalPrincipal      SharingAbilityStatus
	SupportsEditPermission                            SharingAbilityStatus
	SupportsManageListPermission                      SharingAbilityStatus
	SupportsReadPermission                            SharingAbilityStatus
	SupportsRestrictedViewPermission                  SharingAbilityStatus
	SupportsReviewPermission                          SharingAbilityStatus
}

// SharingAbilityStatus represents the status of a sharing ability with reason if disabled
type SharingAbilityStatus struct {
	Enabled        bool
	DisabledReason int
}

// SharingLinkExpirationAbilityStatus represents link expiration capabilities
type SharingLinkExpirationAbilityStatus struct {
	Enabled                 bool
	DisabledReason          int
	DefaultExpirationInDays int
}

// SharingLinkPasswordAbilityStatus represents password protection capabilities
type SharingLinkPasswordAbilityStatus struct {
	Enabled        bool
	DisabledReason int
}

// RecipientLimits represents sharing recipient limits for security controls
type RecipientLimits struct {
	CheckPermissions         *RecipientLimitsInfo
	GrantDirectAccess        *RecipientLimitsInfo
	ShareLink                *RecipientLimitsInfo
	ShareLinkWithDeferRedeem *RecipientLimitsInfo
}

// RecipientLimitsInfo represents recipient limits for a specific sharing type
type RecipientLimitsInfo struct {
	AliasOnly       int
	EmailOnly       int
	MixedRecipients int
	ObjectIdOnly    int
}
