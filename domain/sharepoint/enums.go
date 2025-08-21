package sharepoint

import "fmt"

// ----- Link kind (SP.SharingLinkKind) -----
func LinkKindName(v int) string {
	switch v {
	case 0:
		return "Uninitialized"
	case 1:
		return "Direct"
	case 2:
		return "Organization View"
	case 3:
		return "Organization Edit"
	case 4:
		return "Anonymous View"
	case 5:
		return "Anonymous Edit"
	case 6:
		return "Flexible"
	default:
		return fmt.Sprintf("Unknown (%d)", v)
	}
}

// ----- Link scope (sharing link scope) -----
// Based on observed SharePoint API behavior: anonymous=0, organization=1, specificPeople=2
// -1 shows up for "placeholder" link entries in GetSharingInformation.
func ScopeName(v int) string {
	switch v {
	case -1:
		return "Not Applicable"
	case 0:
		return "Anonymous"
	case 1:
		return "Organization"
	case 2:
		return "Specific People"
	case 3:
		return "Existing Access"
	default:
		return fmt.Sprintf("Unknown (%d)", v)
	}
}

// ----- Role / permission on the item (Microsoft.SharePoint.Client.Sharing.Role) -----
func RoleName(v int) string {
	switch v {
	case 0:
		return "None"
	case 1:
		return "View"
	case 2:
		return "Edit"
	case 3:
		return "Owner"
	case 4:
		return "LimitedView"
	case 5:
		return "LimitedEdit"
	case 6:
		return "Review"
	case 7:
		return "RestrictedView"
	case 8:
		return "Submit"
	case 9:
		return "ManageList"
	default:
		return fmt.Sprintf("Unknown (%d)", v)
	}
}

// ----- PrincipalType flags (Microsoft.SharePoint.Client.Utilities.PrincipalType) -----
// Flags: User=1, DistributionList=2, SecurityGroup=4, SharePointGroup=8 (All=15)
func PrincipalTypeNames(v int) []string {
	if v == 15 {
		return []string{"All"}
	}
	type bitName struct {
		bit  int
		name string
	}
	bits := []bitName{
		{1, "User"},
		{2, "DistributionList"},
		{4, "SecurityGroup"},
		{8, "SharePointGroup"},
	}
	var out []string
	for _, bn := range bits {
		if v&bn.bit != 0 {
			out = append(out, bn.name)
		}
	}
	if len(out) == 0 {
		out = []string{fmt.Sprintf("Unknown (%d)", v)}
	}
	return out
}

func PrincipalTypeName(t int) string {
	// common SharePoint principal types (trimmed)
	switch t {
	case 1:
		return "User"
	case 4:
		return "SecurityGroup"
	case 8:
		return "SharePointGroup"
	default:
		return fmt.Sprintf("%d", t)
	}
}
