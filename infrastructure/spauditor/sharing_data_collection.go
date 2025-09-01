package spauditor

import (
	"context"
	"fmt"

	"spaudit/domain/audit"
	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/infrastructure/spclient"
	"spaudit/logging"
)

// SharingDataCollector collects sharing data.
type SharingDataCollector struct {
	spClient         spclient.SharePointClient
	repo             contracts.SharePointAuditRepository
	sharingService   *sharepoint.SharingService
	logger           *logging.Logger
	progressReporter audit.ProgressReporter
}

// NewSharingDataCollector creates a new sharing data collector
func NewSharingDataCollector(
	spClient spclient.SharePointClient,
	repo contracts.SharePointAuditRepository,
) *SharingDataCollector {
	return &SharingDataCollector{
		spClient:         spClient,
		repo:             repo,
		sharingService:   sharepoint.NewSharingService(),
		logger:           logging.Default().WithComponent("sharing_audit"),
		progressReporter: &audit.NoOpProgressReporter{}, // Default to no-op
	}
}

// SetProgressReporter sets the progress reporter for sharing audit progress
func (s *SharingDataCollector) SetProgressReporter(reporter audit.ProgressReporter) {
	if reporter != nil {
		s.progressReporter = reporter
	}
}

// AuditSiteSharing audits site sharing links.
func (s *SharingDataCollector) AuditSiteSharing(ctx context.Context, auditRunID int64, siteID int64, siteURL string) error {
	// Defensive checks
	if s == nil {
		return fmt.Errorf("SharingDataCollector cannot be nil")
	}
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if siteID <= 0 {
		return fmt.Errorf("site ID must be positive, got: %d", siteID)
	}
	if siteURL == "" {
		return fmt.Errorf("site URL cannot be empty")
	}
	if s.spClient == nil {
		return fmt.Errorf("SharePoint client cannot be nil")
	}
	if s.repo == nil {
		return fmt.Errorf("repository cannot be nil")
	}
	if s.sharingService == nil {
		return fmt.Errorf("sharing service cannot be nil")
	}

	s.logger.Audit("Starting sharing audit", siteURL)

	// Step 1: Find all sharing links in the principals table (not just flexible)
	s.progressReporter.ReportProgress(audit.StandardStages.Sharing, "Discovering sharing links...", 0)
	
	allSharingLinks, err := s.findAllSharingLinks(ctx, siteID)
	if err != nil {
		return fmt.Errorf("find all sharing links: %w", err)
	}

	s.logger.Info("Found sharing links to audit", "count", len(allSharingLinks), "types", "all")
	
	if len(allSharingLinks) == 0 {
		s.progressReporter.ReportProgress(audit.StandardStages.Sharing, "No sharing links found", 0)
		return nil
	}

	s.progressReporter.ReportProgress(audit.StandardStages.Sharing, 
		fmt.Sprintf("Discovered %d sharing links", len(allSharingLinks)), 0)

	// Step 2: For each sharing link, audit the associated item
	for i, link := range allSharingLinks {
		// Report progress per link
		s.progressReporter.ReportProgress(audit.StandardStages.Sharing,
			fmt.Sprintf("Processing sharing link %d/%d", i+1, len(allSharingLinks)), 0)
			
		if err := s.auditSharingLink(ctx, auditRunID, siteID, siteURL, link); err != nil {
			s.logger.Warn("Failed to audit sharing link", "item_guid", link.ItemGUID, "link_type", link.LinkType, "error", err.Error())
			// Continue with other links
			continue
		}
	}

	s.progressReporter.ReportProgress(audit.StandardStages.Sharing,
		fmt.Sprintf("Completed - %d sharing links processed", len(allSharingLinks)), 0)
	s.logger.Audit("Completed sharing audit", siteURL)
	return nil
}

// findAllSharingLinks scans the principals table for all sharing links (all types)
func (s *SharingDataCollector) findAllSharingLinks(ctx context.Context, siteID int64) ([]*sharepoint.DiscoveredSharingLink, error) {
	principals, err := s.repo.GetAllSharingLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all sharing links: %w", err)
	}

	var links []*sharepoint.DiscoveredSharingLink
	for _, p := range principals {
		if p.LoginName == "" {
			continue
		}

		info := s.sharingService.ParseSharingLink(p.LoginName)
		if !info.IsValid {
			s.logger.Warn("Could not parse sharing link from login_name", "login_name", p.LoginName)
			continue
		}
		links = append(links, &sharepoint.DiscoveredSharingLink{
			PrincipalID: p.ID,
			LoginName:   p.LoginName,
			ItemGUID:    info.ItemGUID,
			SharingID:   info.SharingID,
			LinkType:    info.LinkType,
		})
	}

	return links, nil
}

// findFlexibleSharingLinks scans the principals table for flexible sharing links
func (s *SharingDataCollector) findFlexibleSharingLinks(ctx context.Context, siteID int64) ([]*sharepoint.FlexibleSharingReference, error) {
	principals, err := s.repo.GetFlexibleSharingLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("get flexible sharing links: %w", err)
	}

	var links []*sharepoint.FlexibleSharingReference
	for _, p := range principals {
		if p.LoginName == "" {
			continue
		}

		info := s.sharingService.ParseFlexibleSharingLink(p.LoginName)
		if !info.IsValid {
			s.logger.Warn("Could not parse flexible sharing link from login_name", "login_name", p.LoginName)
			continue
		}
		links = append(links, &sharepoint.FlexibleSharingReference{
			PrincipalID: p.ID,
			LoginName:   p.LoginName,
			ItemGUID:    info.ItemGUID,
			SharingID:   info.SharingID,
		})
	}

	return links, nil
}

// auditSharingLink audits any type of sharing link (generic method)
func (s *SharingDataCollector) auditSharingLink(ctx context.Context, auditRunID int64, siteID int64, siteURL string, link *sharepoint.DiscoveredSharingLink) error {
	s.logger.Debug("Auditing sharing link for item", "link_type", link.LinkType, "item_guid", link.ItemGUID)

	// Use the same logic as flexible sharing links - the process is identical
	// regardless of link type (Flexible, OrganizationView, OrganizationEdit, etc.)
	flexibleLink := &sharepoint.FlexibleSharingReference{
		PrincipalID: link.PrincipalID,
		LoginName:   link.LoginName,
		ItemGUID:    link.ItemGUID,
		SharingID:   link.SharingID,
	}

	return s.auditFlexibleSharingLink(ctx, auditRunID, siteID, siteURL, flexibleLink)
}

// auditFlexibleSharingLink audits a specific flexible sharing link using a transaction
func (s *SharingDataCollector) auditFlexibleSharingLink(ctx context.Context, auditRunID int64, siteID int64, siteURL string, link *sharepoint.FlexibleSharingReference) error {
	s.logger.Debug("Auditing flexible sharing link for item", "item_guid", link.ItemGUID)

	// Step 1: Determine if the GUID represents a file or folder
	item, err := s.identifyAndFetchItem(ctx, siteID, siteURL, link.ItemGUID)
	if err != nil {
		return fmt.Errorf("identify and fetch item %s (site_id=%d, sharing_id=%s): %w",
			link.ItemGUID, siteID, link.SharingID, err)
	}

	// Step 2: Check if item already exists using repository pattern
	if err := s.ensureItemExists(ctx, auditRunID, siteID, item); err != nil {
		return fmt.Errorf("ensure item exists: %w", err)
	}

	// Step 3: Get sharing information for the item
	sharingInfo, err := s.spClient.GetItemSharingInfo(ctx, link.ItemGUID)
	if err != nil {
		return fmt.Errorf("get sharing info for item %s (site_id=%d): %w", link.ItemGUID, siteID, err)
	}

	// Step 4: Populate ItemGUID in sharing links and save sharing information
	for _, sharingLink := range sharingInfo.Links {
		// Set the ListItem GUID for database linking
		sharingLink.ItemGUID = item.ListItemGUID
		// FileFolderUniqueID should already be set from mapApiResponseToDomain
	}

	// Save sharing links using repository pattern
	if err := s.repo.SaveSharingLinks(ctx, sharingInfo.Links); err != nil {
		return fmt.Errorf("save sharing links for item %s (site_id=%d, link_count=%d): %w",
			link.ItemGUID, siteID, len(sharingInfo.Links), err)
	}

	// Save governance data (site-level data that comes with each sharing info response)
	if err := s.saveGovernanceData(ctx, siteID, link.ItemGUID, sharingInfo); err != nil {
		s.logger.Warn("Failed to save governance data", "error", err, "item_guid", link.ItemGUID)
		// Don't fail the entire operation for governance data issues
	}

	s.logger.Debug("Successfully audited flexible sharing link for item", "item_guid", link.ItemGUID)
	return nil
}

// saveGovernanceData persists site-level governance data from sharing information
func (s *SharingDataCollector) saveGovernanceData(ctx context.Context, siteID int64, itemGUID string, sharingInfo *sharepoint.SharingInfo) error {
	if sharingInfo == nil {
		return nil
	}

	// Save site-level governance data (one record per site)
	if err := s.repo.SaveSharingGovernance(ctx, sharingInfo); err != nil {
		return fmt.Errorf("save sharing governance: %w", err)
	}

	// Save sharing abilities matrix
	if sharingInfo.SharingAbilities != nil {
		if err := s.repo.SaveSharingAbilities(ctx, sharingInfo.SharingAbilities); err != nil {
			return fmt.Errorf("save sharing abilities: %w", err)
		}
	}

	// Save recipient limits
	if sharingInfo.RecipientLimits != nil {
		if err := s.repo.SaveRecipientLimits(ctx, sharingInfo.RecipientLimits); err != nil {
			return fmt.Errorf("save recipient limits: %w", err)
		}
	}

	// Save sensitivity label (sharing-related sensitivity label information)
	if sharingInfo.SensitivityLabel != nil {
		if err := s.repo.SaveSensitivityLabel(ctx, itemGUID, sharingInfo.SensitivityLabel); err != nil {
			return fmt.Errorf("save sensitivity label: %w", err)
		}
	}

	return nil
}

// ensureItemExists checks if item exists using repository pattern and saves if needed
func (s *SharingDataCollector) ensureItemExists(ctx context.Context, auditRunID int64, siteID int64, item *sharepoint.Item) error {
	// Use the provided audit run ID
	// Check by File/Folder UniqueId (sharing link GUID)
	existingByGUID, err := s.repo.GetItemByGUID(ctx, item.GUID)
	if err != nil {
		return fmt.Errorf("check item existence by GUID %s (site_id=%d): %w", item.GUID, siteID, err)
	}

	// Check by ListItem GUID if we have it
	var existingByListItemGUID *sharepoint.Item
	if item.ListItemGUID != "" {
		existingByListItemGUID, err = s.repo.GetItemByListItemGUID(ctx, item.ListItemGUID)
		if err != nil {
			return fmt.Errorf("check item existence by ListItemGUID %s (site_id=%d): %w", item.ListItemGUID, siteID, err)
		}
	}

	// Check by list_id+item_id as fallback
	existingByListID, err := s.repo.GetItemByListAndID(ctx, item.ListID, int64(item.ID))
	if err != nil {
		return fmt.Errorf("check item existence by list_id+item_id %s/%d (site_id=%d): %w", item.ListID, item.ID, siteID, err)
	}

	// Determine if item already exists
	itemExists := existingByGUID != nil || existingByListItemGUID != nil || existingByListID != nil

	if itemExists {
		if existingByGUID != nil {
			s.logger.Info("Item already exists in database by File/Folder UniqueId", "guid", existingByGUID.GUID)
		} else if existingByListItemGUID != nil {
			s.logger.Info("Item already exists in database by ListItem GUID", "guid", existingByListItemGUID.GUID)
		} else {
			s.logger.Info("Item already exists in database by list_id+item_id", "list_id", item.ListID, "item_id", item.ID, "guid", existingByListID.GUID)
		}
	} else {
		// Item doesn't exist by any method, save it using repository
		if err := s.repo.SaveItem(ctx, item); err != nil {
			return fmt.Errorf("save new item %s (site_id=%d, list_id=%s, name=%s): %w",
				item.GUID, siteID, item.ListID, item.Name, err)
		}
		s.logger.Info("Saved new item discovered via sharing link", "file_folder_unique_id", item.GUID, "list_item_guid", item.ListItemGUID)
	}

	return nil
}

// identifyAndFetchItem determines if GUID is file or folder and fetches item details
func (s *SharingDataCollector) identifyAndFetchItem(ctx context.Context, siteID int64, siteURL, itemGUID string) (*sharepoint.Item, error) {
	// Try as file first
	item, err := s.fetchItemAsFile(ctx, siteID, siteURL, itemGUID)
	if err == nil {
		return item, nil
	}

	s.logger.Debug("Failed to fetch as file, trying as folder", "item_guid", itemGUID, "error", err.Error())

	// Try as folder
	item, err = s.fetchItemAsFolder(ctx, siteID, siteURL, itemGUID)
	if err == nil {
		return item, nil
	}

	return nil, fmt.Errorf("item %s is neither a file nor a folder: file_err=%v, folder_err=%v", itemGUID, err, err)
}

// fetchItemAsFile attempts to fetch item details treating it as a file
func (s *SharingDataCollector) fetchItemAsFile(ctx context.Context, siteID int64, siteURL, itemGUID string) (*sharepoint.Item, error) {
	item, err := s.spClient.ResolveFileByGUID(ctx, itemGUID)
	if err != nil {
		return nil, err
	}
	item.SiteID = siteID
	return item, nil
}

// fetchItemAsFolder attempts to fetch item details treating it as a folder
func (s *SharingDataCollector) fetchItemAsFolder(ctx context.Context, siteID int64, siteURL, itemGUID string) (*sharepoint.Item, error) {
	item, err := s.spClient.ResolveFolderByGUID(ctx, itemGUID)
	if err != nil {
		return nil, err
	}
	item.SiteID = siteID
	return item, nil
}
