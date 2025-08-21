package spclient

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"spaudit/domain/sharepoint"
)

// ---------- Lite models ----------

type SharingApiResponse struct {
	// identity / basics
	DisplayName          string `json:"displayName"`
	ItemUniqueID         string `json:"itemUniqueId"`
	WebURL               string `json:"webUrl"`
	DirectURL            string `json:"directUrl"`
	FileExtension        string `json:"fileExtension"`
	HasUniquePermissions bool   `json:"hasUniquePermissions"`

	// defaults (leave as ints; SPO values vary over time/tenants)
	DefaultLinkKind            int `json:"defaultLinkKind"`
	DefaultShareLinkPermission int `json:"defaultShareLinkPermission"`
	DefaultShareLinkScope      int `json:"defaultShareLinkScope"`

	// who/what has access (trimmed)
	PermissionsInformation PermissionsInformationApiData `json:"permissionsInformation"`

	// Governance & Tenant Context
	TenantID          string `json:"tenantId"`
	TenantDisplayName string `json:"tenantDisplayName"`
	SiteID            string `json:"siteId"`

	// Security & Policy Configuration
	AnonymousLinkExpirationRestrictionDays int  `json:"anonymousLinkExpirationRestrictionDays"`
	AnyoneLinkTrackUsers                   bool `json:"anyoneLinkTrackUsers"`
	CanAddExternalPrincipal                bool `json:"canAddExternalPrincipal"`
	CanAddInternalPrincipal                bool `json:"canAddInternalPrincipal"`
	BlockPeoplePickerAndSharing            bool `json:"blockPeoplePickerAndSharing"`
	CanRequestAccessForGrantAccess         bool `json:"canRequestAccessForGrantAccess"`

	// Information Barriers & Compliance
	SiteIBMode                string               `json:"siteIBMode"`
	SiteIBSegmentIDs          ODataResults[string] `json:"siteIBSegmentIDs"`
	EnforceIBSegmentFiltering bool                 `json:"enforceIBSegmentFiltering"`

	// Sensitivity Label
	SensitivityLabelInformation *SensitivityLabelInformationApiData `json:"sensitivityLabelInformation"`

	// Sharing Capabilities
	SharingAbilities *SharingAbilitiesApiData `json:"sharingAbilities"`

	// Recipient Limits
	RecipientLimits *RecipientLimitsApiData `json:"recipientLimits"`
}

type PermissionsInformationApiData struct {
	Links      ODataResults[LinkApiData]          `json:"links"`
	Principals ODataResults[PrincipalInfoApiData] `json:"principals"`
	SiteAdmins ODataResults[PrincipalInfoApiData] `json:"siteAdmins"`
}

type ODataResults[T any] struct {
	Results []T `json:"results"`
}

// One entry per sharing link present on the item.
type LinkApiData struct {
	IsInherited           bool                           `json:"isInherited"`
	LinkDetails           LinkDetailsApiData             `json:"linkDetails"`
	LinkMembers           ODataResults[PrincipalApiData] `json:"linkMembers"`
	LinkStatus            *SharingAbilityStatusApiData   `json:"linkStatus"` // can be null
	TotalLinkMembersCount int                            `json:"totalLinkMembersCount"`
}

type LinkDetailsApiData struct {
	// core flags
	IsActive     bool `json:"IsActive"`
	IsDefault    bool `json:"IsDefault"`
	IsEditLink   bool `json:"IsEditLink"`
	IsReviewLink bool `json:"IsReviewLink"`

	// kind/scope (expose raw ints; map to labels in your app if you want)
	LinkKind int `json:"LinkKind"`
	Scope    int `json:"Scope"`

	// timeline + actors (strings are fine; sometimes these are empty)
	Created        string            `json:"Created"`
	LastModified   string            `json:"LastModified"`
	CreatedBy      *PrincipalApiData `json:"CreatedBy"`
	LastModifiedBy *PrincipalApiData `json:"LastModifiedBy"`

	// URL can be null for placeholder rows
	URL *string `json:"Url"`

	// Enhanced governance fields from SharePoint API
	AllowsAnonymousAccess           bool              `json:"AllowsAnonymousAccess"`
	BlocksDownload                  bool              `json:"BlocksDownload"`
	Embeddable                      bool              `json:"Embeddable"`
	Expiration                      string            `json:"Expiration"`
	HasExternalGuestInvitees        bool              `json:"HasExternalGuestInvitees"`
	IsAddressBarLink                bool              `json:"IsAddressBarLink"`
	IsCreateOnlyLink                bool              `json:"IsCreateOnlyLink"`
	IsEphemeral                     bool              `json:"IsEphemeral"`
	IsFormsLink                     bool              `json:"IsFormsLink"`
	IsMainLink                      bool              `json:"IsMainLink"`
	IsManageListLink                bool              `json:"IsManageListLink"`
	IsUnhealthy                     bool              `json:"IsUnhealthy"`
	LimitUseToApplication           bool              `json:"LimitUseToApplication"`
	PasswordLastModified            string            `json:"PasswordLastModified"`
	PasswordLastModifiedBy          *PrincipalApiData `json:"PasswordLastModifiedBy"`
	RequiresPassword                bool              `json:"RequiresPassword"`
	RestrictedShareMembership       bool              `json:"RestrictedShareMembership"`
	RestrictToExistingRelationships bool              `json:"RestrictToExistingRelationships"`
	ShareId                         string            `json:"ShareId"`
	ShareTokenString                *string           `json:"ShareTokenString"`
	SharingLinkStatus               int               `json:"SharingLinkStatus"`
	TrackLinkUsers                  bool              `json:"TrackLinkUsers"`
}

type SharingAbilityStatusApiData struct {
	DisabledReason int  `json:"disabledReason"`
	Enabled        bool `json:"enabled"`
}

type PrincipalApiData struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	LoginName         string  `json:"loginName"`
	Email             *string `json:"email"`
	UserPrincipalName *string `json:"userPrincipalName"`
	PrincipalType     int     `json:"principalType"`
	IsExternal        bool    `json:"isExternal"`
}

type PrincipalInfoApiData struct {
	Principal   PrincipalApiData `json:"principal"`
	Role        int              `json:"role"` // e.g. 1=Read, 2=Edit, 3=Owner (typical)
	IsInherited bool             `json:"isInherited"`
}

// ---------- Decoder that supports both verbose and JSON Light ----------

func DecodeSharingApiResponse(data []byte) (SharingApiResponse, error) {
	// Probe for the "d" wrapper first
	var probe struct {
		D json.RawMessage `json:"d"`
	}
	if err := json.Unmarshal(data, &probe); err == nil && len(probe.D) > 0 {
		var s SharingApiResponse
		if err := json.Unmarshal(probe.D, &s); err != nil {
			return SharingApiResponse{}, err
		}
		return s, nil
	}
	// Otherwise, decode straight into SharingApiResponse
	var s SharingApiResponse
	if err := json.Unmarshal(data, &s); err != nil {
		return SharingApiResponse{}, err
	}
	return s, nil
}

// --- helpers to pretty print members and links ---

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ----- Pretty printers -----

// Friendly summary for one link entry from permissionsInformation.links.results[i]
// status *LinkStatusLite,
func SummarizeLink(d LinkDetailsApiData, members []PrincipalApiData, total int) string {
	var by string
	if d.CreatedBy != nil && d.CreatedBy.Name != "" {
		by = d.CreatedBy.Name
	}
	parts := []string{
		fmt.Sprintf("Kind: %s", sharepoint.LinkKindName(d.LinkKind)),
		fmt.Sprintf("Scope: %s", sharepoint.ScopeName(d.Scope)),
		fmt.Sprintf("Active: %t", d.IsActive),
		fmt.Sprintf("Default: %t", d.IsDefault),
	}
	if d.IsReviewLink {
		parts = append(parts, "Review link: yes")
	}
	// if d.BlocksDownload {
	// 	parts = append(parts, "Blocks download: yes")
	// }
	// if d.RequiresPassword {
	// 	parts = append(parts, "Password protected: yes")
	// }
	// if d.Url != "" {
	// 	parts = append(parts, "Url: "+d.Url)
	// }
	if by != "" {
		parts = append(parts, "CreatedBy: "+by)
	}
	// if status != nil {
	// 	parts = append(parts,
	// 		fmt.Sprintf("Enabled: %t", status.Enabled),
	// 		"DisabledReason: "+disabledReasonName(status.DisabledReason),
	// 	)
	// }

	// Members
	var mm []string
	for _, m := range members {
		label := m.Name
		// Safely handle potentially nil Email pointer
		if m.Email != nil && *m.Email != "" && !strings.EqualFold(*m.Email, label) {
			label += " <" + *m.Email + ">"
		}
		types := strings.Join(sharepoint.PrincipalTypeNames(m.PrincipalType), ", ")
		mm = append(mm, fmt.Sprintf("%s [%s]", label, types))
	}
	memberLine := "Members: none"
	if len(mm) > 0 {
		memberLine = "Members: " + strings.Join(mm, "; ")
	}
	parts = append(parts, memberLine, fmt.Sprintf("TotalMembers: %d", total))

	return strings.Join(parts, " | ")
}

// New governance API data structures

// SensitivityLabelInformationApiData represents sensitivity label information from SharePoint API
type SensitivityLabelInformationApiData struct {
	ID                             string `json:"id"`
	DisplayName                    string `json:"displayName"`
	Color                          string `json:"color"`
	Tooltip                        string `json:"tooltip"`
	HasIRMProtection               bool   `json:"hasIRMProtection"`
	SensitivityLabelProtectionType string `json:"sensitivityLabelProtectionType"`
}

// SharingAbilitiesApiData represents the sharing abilities matrix from SharePoint API
type SharingAbilitiesApiData struct {
	CanStopSharing             bool                           `json:"canStopSharing"`
	AnonymousLinkAbilities     *SharingLinkAbilitiesApiData   `json:"anonymousLinkAbilities"`
	AnyoneLinkAbilities        *SharingLinkAbilitiesApiData   `json:"anyoneLinkAbilities"`
	OrganizationLinkAbilities  *SharingLinkAbilitiesApiData   `json:"organizationLinkAbilities"`
	PeopleSharingLinkAbilities *SharingLinkAbilitiesApiData   `json:"peopleSharingLinkAbilities"`
	DirectSharingAbilities     *DirectSharingAbilitiesApiData `json:"directSharingAbilities"`
}

// SharingLinkAbilitiesApiData represents capabilities for a specific type of sharing link
type SharingLinkAbilitiesApiData struct {
	CanAddNewExternalPrincipals             SharingAbilityStatusApiData                `json:"canAddNewExternalPrincipals"`
	CanDeleteEditLink                       SharingAbilityStatusApiData                `json:"canDeleteEditLink"`
	CanDeleteManageListLink                 SharingAbilityStatusApiData                `json:"canDeleteManageListLink"`
	CanDeleteReadLink                       SharingAbilityStatusApiData                `json:"canDeleteReadLink"`
	CanDeleteRestrictedViewLink             SharingAbilityStatusApiData                `json:"canDeleteRestrictedViewLink"`
	CanDeleteReviewLink                     SharingAbilityStatusApiData                `json:"canDeleteReviewLink"`
	CanDeleteSubmitOnlyLink                 SharingAbilityStatusApiData                `json:"canDeleteSubmitOnlyLink"`
	CanGetEditLink                          SharingAbilityStatusApiData                `json:"canGetEditLink"`
	CanGetManageListLink                    SharingAbilityStatusApiData                `json:"canGetManageListLink"`
	CanGetReadLink                          SharingAbilityStatusApiData                `json:"canGetReadLink"`
	CanGetRestrictedViewLink                SharingAbilityStatusApiData                `json:"canGetRestrictedViewLink"`
	CanGetReviewLink                        SharingAbilityStatusApiData                `json:"canGetReviewLink"`
	CanGetSubmitOnlyLink                    SharingAbilityStatusApiData                `json:"canGetSubmitOnlyLink"`
	CanHaveExternalUsers                    SharingAbilityStatusApiData                `json:"canHaveExternalUsers"`
	CanManageEditLink                       SharingAbilityStatusApiData                `json:"canManageEditLink"`
	CanManageManageListLink                 SharingAbilityStatusApiData                `json:"canManageManageListLink"`
	CanManageReadLink                       SharingAbilityStatusApiData                `json:"canManageReadLink"`
	CanManageRestrictedViewLink             SharingAbilityStatusApiData                `json:"canManageRestrictedViewLink"`
	CanManageReviewLink                     SharingAbilityStatusApiData                `json:"canManageReviewLink"`
	CanManageSubmitOnlyLink                 SharingAbilityStatusApiData                `json:"canManageSubmitOnlyLink"`
	LinkExpiration                          *SharingLinkExpirationAbilityStatusApiData `json:"linkExpiration"`
	PasswordProtected                       *SharingLinkPasswordAbilityStatusApiData   `json:"passwordProtected"`
	SubmitOnlyLinkExpiration                *SharingLinkExpirationAbilityStatusApiData `json:"submitOnlylinkExpiration"`
	SupportsRestrictedView                  SharingAbilityStatusApiData                `json:"supportsRestrictedView"`
	SupportsRestrictToExistingRelationships SharingAbilityStatusApiData                `json:"supportsRestrictToExistingRelationships"`
}

// DirectSharingAbilitiesApiData represents capabilities for direct sharing (non-link based)
type DirectSharingAbilitiesApiData struct {
	CanAddExternalPrincipal                           SharingAbilityStatusApiData `json:"canAddExternalPrincipal"`
	CanAddInternalPrincipal                           SharingAbilityStatusApiData `json:"canAddInternalPrincipal"`
	CanAddNewExternalPrincipal                        SharingAbilityStatusApiData `json:"canAddNewExternalPrincipal"`
	CanRequestGrantAccess                             SharingAbilityStatusApiData `json:"canRequestGrantAccess"`
	CanRequestGrantAccessForExistingExternalPrincipal SharingAbilityStatusApiData `json:"canRequestGrantAccessForExistingExternalPrincipal"`
	CanRequestGrantAccessForInternalPrincipal         SharingAbilityStatusApiData `json:"canRequestGrantAccessForInternalPrincipal"`
	CanRequestGrantAccessForNewExternalPrincipal      SharingAbilityStatusApiData `json:"canRequestGrantAccessForNewExternalPrincipal"`
	SupportsEditPermission                            SharingAbilityStatusApiData `json:"supportsEditPermission"`
	SupportsManageListPermission                      SharingAbilityStatusApiData `json:"supportsManageListPermission"`
	SupportsReadPermission                            SharingAbilityStatusApiData `json:"supportsReadPermission"`
	SupportsRestrictedViewPermission                  SharingAbilityStatusApiData `json:"supportsRestrictedViewPermission"`
	SupportsReviewPermission                          SharingAbilityStatusApiData `json:"supportsReviewPermission"`
}

// SharingLinkExpirationAbilityStatusApiData represents link expiration capabilities
type SharingLinkExpirationAbilityStatusApiData struct {
	Enabled                 bool `json:"enabled"`
	DisabledReason          int  `json:"disabledReason"`
	DefaultExpirationInDays int  `json:"defaultExpirationInDays"`
}

// SharingLinkPasswordAbilityStatusApiData represents password protection capabilities
type SharingLinkPasswordAbilityStatusApiData struct {
	Enabled        bool `json:"enabled"`
	DisabledReason int  `json:"disabledReason"`
}

// RecipientLimitsApiData represents sharing recipient limits for security controls
type RecipientLimitsApiData struct {
	CheckPermissions         *RecipientLimitsInfoApiData `json:"checkPermissions"`
	GrantDirectAccess        *RecipientLimitsInfoApiData `json:"grantDirectAccess"`
	ShareLink                *RecipientLimitsInfoApiData `json:"shareLink"`
	ShareLinkWithDeferRedeem *RecipientLimitsInfoApiData `json:"shareLinkWithDeferRedeem"`
}

// RecipientLimitsInfoApiData represents recipient limits for a specific sharing type
type RecipientLimitsInfoApiData struct {
	AliasOnly       int `json:"AliasOnly"`
	EmailOnly       int `json:"EmailOnly"`
	MixedRecipients int `json:"MixedRecipients"`
	ObjectIdOnly    int `json:"ObjectIdOnly"`
}

// ---------- File and Properties structures for list item queries ----------

// ListItemApiResponse represents a SharePoint list item from the Items API
type ListItemApiResponse struct {
	ID                   int            `json:"Id"`
	GUID                 string         `json:"GUID"`
	Title                string         `json:"Title"`
	FileRef              string         `json:"FileRef"`
	FileSystemObjectType int            `json:"FileSystemObjectType"`
	FileLeafRef          string         `json:"FileLeafRef"`
	File                 *FileApiData   `json:"File"`
	Folder               *FolderApiData `json:"Folder"`
}

// FileApiData represents the File object from SharePoint list items
type FileApiData struct {
	ServerRelativeUrl string                 `json:"ServerRelativeUrl"`
	Properties        *FilePropertiesApiData `json:"Properties"`
}

// FolderApiData represents the Folder object from SharePoint list items
type FolderApiData struct {
	ServerRelativeUrl string `json:"ServerRelativeUrl"`
}

// FilePropertiesApiData represents File.Properties containing sensitivity label and other metadata
type FilePropertiesApiData struct {
	// Sensitivity Label Properties (Information Rights Management)
	IPLabelOwnerEmail       string `json:"vti_x005f_iplabelowneremail"`
	IPLabelAssignmentMethod string `json:"vti_x005f_iplabelassignmentmethod"`
	IPLabelDisplayName      string `json:"vti_x005f_iplabeldisplayname"`
	IPLabelChangedDate      string `json:"vti_x005f_iplabelchangeddate"`
	IPLabelSharingProps     string `json:"vti_x005f_iplabelsharingprops"`

	// Microsoft Information Protection (MSIP) Label - GUID-based property names
	MSIPLabelSetDate string `json:"MSIP_x005f_Label_x005f_8c3d088b_x002d_6243_x002d_4963_x002d_a2e2_x002d_8b321ab7f8fc_x005f_SetDate"`

	// Add other common properties as map for flexibility
	OtherProperties map[string]interface{} `json:"-"`
}

// UnmarshalJSON custom unmarshaler to capture all properties including unknown ones
func (f *FilePropertiesApiData) UnmarshalJSON(data []byte) error {
	// First unmarshal into a generic map
	var allProps map[string]interface{}
	if err := json.Unmarshal(data, &allProps); err != nil {
		return err
	}

	// Extract known properties
	if v, ok := allProps["vti_x005f_iplabelowneremail"].(string); ok {
		f.IPLabelOwnerEmail = v
	}
	if v, ok := allProps["vti_x005f_iplabelassignmentmethod"].(string); ok {
		f.IPLabelAssignmentMethod = v
	}
	if v, ok := allProps["vti_x005f_iplabeldisplayname"].(string); ok {
		f.IPLabelDisplayName = v
	}
	if v, ok := allProps["vti_x005f_iplabelchangeddate"].(string); ok {
		f.IPLabelChangedDate = v
	}
	if v, ok := allProps["vti_x005f_iplabelsharingprops"].(string); ok {
		f.IPLabelSharingProps = v
	}
	if v, ok := allProps["MSIP_x005f_Label_x005f_8c3d088b_x002d_6243_x002d_4963_x002d_a2e2_x002d_8b321ab7f8fc_x005f_SetDate"].(string); ok {
		f.MSIPLabelSetDate = v
	}

	// Store remaining properties in OtherProperties
	f.OtherProperties = make(map[string]interface{})
	knownKeys := map[string]bool{
		"vti_x005f_iplabelowneremail":       true,
		"vti_x005f_iplabelassignmentmethod": true,
		"vti_x005f_iplabeldisplayname":      true,
		"vti_x005f_iplabelchangeddate":      true,
		"vti_x005f_iplabelsharingprops":     true,
		"MSIP_x005f_Label_x005f_8c3d088b_x002d_6243_x002d_4963_x002d_a2e2_x002d_8b321ab7f8fc_x005f_SetDate": true,
		"__metadata": true,
	}
	for key, value := range allProps {
		if !knownKeys[key] {
			f.OtherProperties[key] = value
		}
	}

	return nil
}

// MapToSensitivityLabel converts file properties to domain sensitivity label model
func (f *FilePropertiesApiData) MapToSensitivityLabel(siteID int64, itemGUID string) *sharepoint.ItemSensitivityLabel {
	if f == nil {
		return nil
	}

	// Only create if we have a label ID
	labelID := f.IPLabelDisplayName != "" || f.MSIPLabelSetDate != ""
	if vti_iplabelid, exists := f.OtherProperties["vti_x005f_iplabelid"].(string); exists && vti_iplabelid != "" {
		labelID = true
	}

	if !labelID {
		return nil
	}

	label := &sharepoint.ItemSensitivityLabel{
		SiteID:           siteID,
		ItemGUID:         itemGUID,
		DisplayName:      f.IPLabelDisplayName,
		OwnerEmail:       f.IPLabelOwnerEmail,
		AssignmentMethod: f.IPLabelAssignmentMethod,
		DiscoveredAt:     time.Now(),
	}

	// Extract label ID from other properties
	if vti_iplabelid, ok := f.OtherProperties["vti_x005f_iplabelid"].(string); ok {
		label.LabelID = vti_iplabelid
	}

	// Parse set date
	if f.MSIPLabelSetDate != "" {
		if setTime, err := time.Parse(time.RFC3339, f.MSIPLabelSetDate); err == nil {
			label.SetDate = &setTime
		}
	}

	// Extract MSIP method if not already set
	if label.AssignmentMethod == "" {
		if method, ok := f.OtherProperties["MSIP_x005f_Label_x005f_8c3d088b_x002d_6243_x002d_4963_x002d_a2e2_x002d_8b321ab7f8fc_x005f_Method"].(string); ok {
			label.AssignmentMethod = method
		}
	}

	// Extract display name from MSIP if not already set
	if label.DisplayName == "" {
		if name, ok := f.OtherProperties["MSIP_x005f_Label_x005f_8c3d088b_x002d_6243_x002d_4963_x002d_a2e2_x002d_8b321ab7f8fc_x005f_Name"].(string); ok {
			label.DisplayName = name
		}
	}

	// Extract protection status
	if enabled, ok := f.OtherProperties["MSIP_x005f_Label_x005f_8c3d088b_x002d_6243_x002d_4963_x002d_a2e2_x002d_8b321ab7f8fc_x005f_Enabled"].(bool); ok {
		label.HasIRMProtection = enabled
	}

	// Extract content bits
	if contentBits, ok := f.OtherProperties["MSIP_x005f_Label_x005f_8c3d088b_x002d_6243_x002d_4963_x002d_a2e2_x002d_8b321ab7f8fc_x005f_ContentBits"].(float64); ok {
		label.ContentBits = int(contentBits)
	}

	// Extract label flags
	if flags, ok := f.OtherProperties["vti_x005f_iplabelflags"].(float64); ok {
		label.LabelFlags = int(flags)
	}

	// Extract promotion version
	if version, ok := f.OtherProperties["vti_x005f_iplabelpromotionversion"].(float64); ok {
		label.PromotionVersion = int(version)
	}

	// Extract label hash
	if hash, ok := f.OtherProperties["vti_x005f_iplabelhash"].(string); ok {
		label.LabelHash = hash
	}

	return label
}
