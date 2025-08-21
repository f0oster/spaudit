package sharepoint

import "time"

// ItemSensitivityLabel represents sensitivity label information discovered from SharePoint file properties
// This is separate from sharing-related sensitivity labels and is discovered during list item processing
type ItemSensitivityLabel struct {
	SiteID   int64
	ItemGUID string

	// Core sensitivity label information
	LabelID          string     // vti_x005f_iplabelid
	DisplayName      string     // MSIP_x005f_Label_x005f_..._x005f_Name or vti_x005f_iplabeldisplayname
	OwnerEmail       string     // vti_x005f_iplabelowneremail
	SetDate          *time.Time // MSIP_x005f_Label_x005f_..._x005f_SetDate
	AssignmentMethod string     // MSIP_x005f_Label_x005f_..._x005f_Method

	// Protection and governance
	HasIRMProtection bool // MSIP_x005f_Label_x005f_..._x005f_Enabled
	ContentBits      int  // MSIP_x005f_Label_x005f_..._x005f_ContentBits
	LabelFlags       int  // vti_x005f_iplabelflags

	// Audit metadata
	DiscoveredAt     time.Time
	PromotionVersion int    // vti_x005f_iplabelpromotionversion
	LabelHash        string // vti_x005f_iplabelhash
}
