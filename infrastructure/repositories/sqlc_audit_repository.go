package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"spaudit/database"
	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/gen/db"
)

// SqlcAuditRepository implements contracts.AuditRepository using sqlc-generated queries with read/write separation
type SqlcAuditRepository struct {
	*BaseRepository
}

// NewSqlcAuditRepository creates a new sqlc-based audit repository with read/write database separation
func NewSqlcAuditRepository(database *database.Database) contracts.AuditRepository {
	return &SqlcAuditRepository{
		BaseRepository: NewBaseRepository(database),
	}
}

// SaveSite persists a site to the database and updates the site ID
func (r *SqlcAuditRepository) SaveSite(ctx context.Context, site *sharepoint.Site) error {
	siteID, err := r.WriteQueries().UpsertSite(ctx, db.UpsertSiteParams{
		SiteUrl: site.URL,
		Title:   r.ToNullString(site.Title),
	})
	if err != nil {
		return err
	}
	site.ID = siteID // Update the domain object with the returned site ID
	return nil
}

// GetSiteByURL retrieves a site from the database by URL
func (r *SqlcAuditRepository) GetSiteByURL(ctx context.Context, siteURL string) (*sharepoint.Site, error) {
	row, err := r.ReadQueries().GetSiteByURL(ctx, siteURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Site not found
		}
		return nil, fmt.Errorf("query site by URL: %w", err)
	}

	return &sharepoint.Site{
		ID:    row.SiteID,
		URL:   row.SiteUrl,
		Title: r.FromNullString(row.Title),
	}, nil
}

// SaveWeb persists a web to the database
func (r *SqlcAuditRepository) SaveWeb(ctx context.Context, auditRunID int64, web *sharepoint.Web) error {
	return r.WriteQueries().InsertWeb(ctx, db.InsertWebParams{
		SiteID:     web.SiteID,
		WebID:      web.ID,
		Url:        r.ToNullString(web.URL),
		Title:      r.ToNullString(web.Title),
		Template:   r.ToNullString(web.Template),
		HasUnique:  r.ToNullBool(web.HasUnique),
		AuditRunID: sql.NullInt64{Int64: auditRunID, Valid: true},
	})
}

// SaveList persists a list to the database
func (r *SqlcAuditRepository) SaveList(ctx context.Context, auditRunID int64, list *sharepoint.List) error {
	// Transform domain List to SQLC params
	return r.WriteQueries().InsertList(ctx, db.InsertListParams{
		SiteID:       list.SiteID,
		ListID:       list.ID,
		WebID:        list.WebID,
		Title:        list.Title,
		Url:          r.ToNullString(list.URL),
		BaseTemplate: r.ToNullInt64(int64(list.BaseTemplate)),
		ItemCount:    r.ToNullInt64(int64(list.ItemCount)),
		HasUnique:    r.ToNullBool(list.HasUnique),
		AuditRunID:   sql.NullInt64{Int64: auditRunID, Valid: true},
	})
}

// SaveItem persists an item to the database
func (r *SqlcAuditRepository) SaveItem(ctx context.Context, auditRunID int64, item *sharepoint.Item) error {
	return r.WriteQueries().InsertItem(ctx, db.InsertItemParams{
		SiteID:       item.SiteID,
		ItemGuid:     item.GUID,
		ListItemGuid: r.ToNullString(item.ListItemGUID),
		ListID:       item.ListID,
		ItemID:       int64(item.ID),
		Url:          r.ToNullString(item.URL),
		IsFile:       r.ToNullBool(item.IsFile),
		IsFolder:     r.ToNullBool(item.IsFolder),
		HasUnique:    r.ToNullBool(item.HasUnique),
		Name:         r.ToNullString(item.Name),
		AuditRunID:   sql.NullInt64{Int64: auditRunID, Valid: true},
	})
}

// SaveRoleDefinitions persists role definitions to the database
func (r *SqlcAuditRepository) SaveRoleDefinitions(ctx context.Context, auditRunID int64, siteID int64, roleDefs []*sharepoint.RoleDefinition) error {
	for _, rd := range roleDefs {
		if err := r.WriteQueries().InsertRoleDefinition(ctx, db.InsertRoleDefinitionParams{
			SiteID:      siteID,
			RoleDefID:   rd.ID,
			Name:        rd.Name,
			Description: r.ToNullString(rd.Description),
			AuditRunID:  sql.NullInt64{Int64: auditRunID, Valid: true},
		}); err != nil {
			return err
		}
	}
	return nil
}

// SavePrincipal persists a principal to the database, handling duplicates within audit run
func (r *SqlcAuditRepository) SavePrincipal(ctx context.Context, auditRunID int64, principal *sharepoint.Principal) error {
	err := r.WriteQueries().InsertPrincipal(ctx, db.InsertPrincipalParams{
		SiteID:        principal.SiteID,
		PrincipalID:   principal.ID,
		PrincipalType: principal.PrincipalType,
		Title:         r.ToNullString(strings.TrimSpace(principal.Title)),
		LoginName:     r.ToNullString(principal.LoginName),
		Email:         r.ToNullString(principal.Email),
		AuditRunID:    sql.NullInt64{Int64: auditRunID, Valid: true},
	})
	
	// Ignore duplicate principal within same audit run (UNIQUE constraint on site_id, principal_id, audit_run_id)
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: principals.site_id, principals.principal_id, principals.audit_run_id") {
		return nil // Principal already exists in this audit run - this is expected
	}
	
	return err
}

// SaveRoleAssignments persists role assignments to the database
func (r *SqlcAuditRepository) SaveRoleAssignments(ctx context.Context, auditRunID int64, siteID int64, assignments []*sharepoint.RoleAssignment) error {
	for _, assignment := range assignments {
		if err := r.WriteQueries().InsertRoleAssignment(ctx, db.InsertRoleAssignmentParams{
			SiteID:      siteID,
			ObjectType:  assignment.ObjectType,
			ObjectKey:   assignment.ObjectKey,
			PrincipalID: assignment.PrincipalID,
			RoleDefID:   assignment.RoleDefID,
			Inherited:   r.ToNullBool(assignment.Inherited),
			AuditRunID:  sql.NullInt64{Int64: auditRunID, Valid: true},
		}); err != nil {
			return err
		}
	}
	return nil
}

// ClearRoleAssignments removes existing role assignments for an object
func (r *SqlcAuditRepository) ClearRoleAssignments(ctx context.Context, siteID int64, objectType, objectKey string) error {
	return r.WriteQueries().DeleteRoleAssignmentsForObject(ctx, db.DeleteRoleAssignmentsForObjectParams{
		SiteID:     siteID,
		ObjectType: objectType,
		ObjectKey:  objectKey,
	})
}

// SaveSharingLinks persists sharing links to the database
func (r *SqlcAuditRepository) SaveSharingLinks(ctx context.Context, auditRunID int64, siteID int64, links []*sharepoint.SharingLink) error {
	for _, link := range links {

		// Skip links without URLs as they are likely stale, inactive, or incomplete
		// TODO: Investigate SharePoint API specifics in spclient, this is a hacky fix
		if link.URL == "" {
			continue
		}

		// Save CreatedBy and LastModifiedBy principals first
		var createdByID, lastModifiedByID sql.NullInt64
		if link.CreatedBy != nil {
			link.CreatedBy.SiteID = siteID
			if err := r.SavePrincipal(ctx, auditRunID, link.CreatedBy); err != nil {
				return fmt.Errorf("save CreatedBy principal %d: %w", link.CreatedBy.ID, err)
			}
			createdByID = sql.NullInt64{Int64: link.CreatedBy.ID, Valid: true}
		}
		if link.LastModifiedBy != nil {
			link.LastModifiedBy.SiteID = siteID
			if err := r.SavePrincipal(ctx, auditRunID, link.LastModifiedBy); err != nil {
				return fmt.Errorf("save LastModifiedBy principal %d: %w", link.LastModifiedBy.ID, err)
			}
			lastModifiedByID = sql.NullInt64{Int64: link.LastModifiedBy.ID, Valid: true}
		}

		// Convert time fields
		var createdAt, lastModifiedAt sql.NullTime
		if link.CreatedAt != nil {
			createdAt = sql.NullTime{Time: *link.CreatedAt, Valid: true}
		}
		if link.LastModifiedAt != nil {
			lastModifiedAt = sql.NullTime{Time: *link.LastModifiedAt, Valid: true}
		}

		// Insert the sharing link
		linkID, err := r.WriteQueries().InsertSharingLink(ctx, db.InsertSharingLinkParams{
			SiteID:                    siteID,
			LinkID:                    link.ID,
			ItemGuid:                  r.ToNullString(link.ItemGUID),
			FileFolderUniqueID:        r.ToNullString(link.FileFolderUniqueID),
			Url:                       r.ToNullString(link.URL),
			LinkKind:                  r.ToNullInt64(int64(link.LinkKind)),
			Scope:                     r.ToNullInt64(int64(link.Scope)),
			IsActive:                  r.ToNullBool(link.IsActive),
			IsDefault:                 r.ToNullBool(link.IsDefault),
			IsEditLink:                r.ToNullBool(link.IsEditLink),
			IsReviewLink:              r.ToNullBool(link.IsReviewLink),
			IsInherited:               r.ToNullBool(link.IsInherited),
			CreatedAt:                 createdAt,
			CreatedByPrincipalID:      createdByID,
			LastModifiedAt:            lastModifiedAt,
			LastModifiedByPrincipalID: lastModifiedByID,
			TotalMembersCount:         r.ToNullInt64(int64(len(link.Members))),

			// Enhanced governance fields
			Expiration:                        r.ToNullTime(link.Expiration),
			PasswordLastModified:              r.ToNullTime(link.PasswordLastModified),
			PasswordLastModifiedByPrincipalID: r.principalToNullInt64(link.PasswordLastModifiedBy),
			HasExternalGuestInvitees:          r.ToNullBool(link.HasExternalGuestInvitees),
			TrackLinkUsers:                    r.ToNullBool(link.TrackLinkUsers),
			IsEphemeral:                       r.ToNullBool(link.IsEphemeral),
			IsUnhealthy:                       r.ToNullBool(link.IsUnhealthy),
			IsAddressBarLink:                  r.ToNullBool(link.IsAddressBarLink),
			IsCreateOnlyLink:                  r.ToNullBool(link.IsCreateOnlyLink),
			IsFormsLink:                       r.ToNullBool(link.IsFormsLink),
			IsMainLink:                        r.ToNullBool(link.IsMainLink),
			IsManageListLink:                  r.ToNullBool(link.IsManageListLink),
			AllowsAnonymousAccess:             r.ToNullBool(link.AllowsAnonymousAccess),
			Embeddable:                        r.ToNullBool(link.Embeddable),
			LimitUseToApplication:             r.ToNullBool(link.LimitUseToApplication),
			RestrictToExistingRelationships:   r.ToNullBool(link.RestrictToExistingRelationships),
			BlocksDownload:                    r.ToNullBool(link.BlocksDownload),
			RequiresPassword:                  r.ToNullBool(link.RequiresPassword),
			RestrictedMembership:              r.ToNullBool(link.RestrictedMembership),
			InheritedFrom:                     r.ToNullString(link.InheritedFrom),
			ShareID:                           r.ToNullString(link.ShareID),
			ShareToken:                        r.ToNullString(link.ShareToken),
			SharingLinkStatus:                 r.intPtrToNullInt64(link.SharingLinkStatus),
			AuditRunID:                        sql.NullInt64{Int64: auditRunID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("save sharing link: %w", err)
		}

		// Clear existing members for this link
		if err := r.WriteQueries().ClearMembersForLink(ctx, db.ClearMembersForLinkParams{
			SiteID: siteID,
			LinkID: linkID,
		}); err != nil {
			return fmt.Errorf("clear link members: %w", err)
		}

		// Save member principals first, then add them to the link
		for _, member := range link.Members {
			// Set site ID for member principal
			member.SiteID = siteID
			// Ensure the principal exists in the database before adding to link
			if err := r.SavePrincipal(ctx, auditRunID, member); err != nil {
				return fmt.Errorf("save principal %d for link member: %w", member.ID, err)
			}

			// Now add the member to the link
			if err := r.WriteQueries().AddMemberToLink(ctx, db.AddMemberToLinkParams{
				SiteID:      siteID,
				LinkID:      linkID,
				PrincipalID: member.ID,
				AuditRunID:  sql.NullInt64{Int64: auditRunID, Valid: true},
			}); err != nil {
				return fmt.Errorf("add member to link: %w", err)
			}
		}

		// Note: audit_run_id is already set in the UpsertSharingLinkByUrlKindScope operation above

		// Note: audit_run_id is already set in the AddMemberToLink operations above
	}
	return nil
}

// ClearSharingLinks removes existing sharing links for an item
func (r *SqlcAuditRepository) ClearSharingLinks(ctx context.Context, siteID int64, itemGUID string) error {
	// Note: This would typically require a DELETE query for sharing_links by item_guid
	// We rely on the ON CONFLICT handling in UpsertSharingLinkByUrlKindScope
	// which updates existing links rather than creating duplicates
	return nil
}

// GetAllSharingLinks retrieves all sharing links from the principals table
func (r *SqlcAuditRepository) GetAllSharingLinks(ctx context.Context, siteID int64) ([]*sharepoint.Principal, error) {
	rows, err := r.ReadQueries().GetAllSharingLinks(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("query all sharing links: %w", err)
	}

	var principals []*sharepoint.Principal
	for _, row := range rows {
		principals = append(principals, &sharepoint.Principal{
			ID:        row.PrincipalID,
			LoginName: r.FromNullString(row.LoginName),
			SiteID:    siteID,
		})
	}
	return principals, nil
}

// GetFlexibleSharingLinks retrieves flexible sharing links from the principals table
func (r *SqlcAuditRepository) GetFlexibleSharingLinks(ctx context.Context, siteID int64) ([]*sharepoint.Principal, error) {
	rows, err := r.ReadQueries().GetFlexibleSharingLinks(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("query flexible sharing links: %w", err)
	}

	var principals []*sharepoint.Principal
	for _, row := range rows {
		principals = append(principals, &sharepoint.Principal{
			ID:        row.PrincipalID,
			LoginName: r.FromNullString(row.LoginName),
			SiteID:    siteID,
		})
	}
	return principals, nil
}

// GetItemByGUID retrieves an item by its GUID
func (r *SqlcAuditRepository) GetItemByGUID(ctx context.Context, siteID int64, itemGUID string) (*sharepoint.Item, error) {
	row, err := r.ReadQueries().GetItemByGUID(ctx, db.GetItemByGUIDParams{
		SiteID:   siteID,
		ItemGuid: itemGUID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Item not found
		}
		return nil, fmt.Errorf("query item by GUID: %w", err)
	}

	return &sharepoint.Item{
		ID:           int(row.ItemID),
		GUID:         row.ItemGuid,
		ListID:       row.ListID,
		ListItemGUID: r.FromNullString(row.ListItemGuid),
		SiteID:       siteID,
	}, nil
}

// GetItemByListItemGUID retrieves an item by its list item GUID
func (r *SqlcAuditRepository) GetItemByListItemGUID(ctx context.Context, siteID int64, listItemGUID string) (*sharepoint.Item, error) {
	row, err := r.ReadQueries().GetItemByListItemGUID(ctx, db.GetItemByListItemGUIDParams{
		SiteID:       siteID,
		ListItemGuid: sql.NullString{String: listItemGUID, Valid: true},
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Item not found
		}
		return nil, fmt.Errorf("query item by list item GUID: %w", err)
	}

	return &sharepoint.Item{
		ID:           int(row.ItemID),
		GUID:         row.ItemGuid,
		ListID:       row.ListID,
		ListItemGUID: r.FromNullString(row.ListItemGuid),
		SiteID:       siteID,
	}, nil
}

// GetItemByListAndID retrieves an item by list ID and item ID
func (r *SqlcAuditRepository) GetItemByListAndID(ctx context.Context, siteID int64, listID string, itemID int64) (*sharepoint.Item, error) {
	row, err := r.ReadQueries().GetItemByListAndID(ctx, db.GetItemByListAndIDParams{
		SiteID: siteID,
		ListID: listID,
		ItemID: itemID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Item not found
		}
		return nil, fmt.Errorf("query item by list and ID: %w", err)
	}

	return &sharepoint.Item{
		ID:           int(row.ItemID),
		GUID:         row.ItemGuid,
		ListID:       row.ListID,
		ListItemGUID: r.FromNullString(row.ListItemGuid),
		SiteID:       siteID,
	}, nil
}

// principalToNullInt64 converts a Principal pointer to sql.NullInt64 for database storage
func (r *SqlcAuditRepository) principalToNullInt64(principal *sharepoint.Principal) sql.NullInt64 {
	if principal == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: principal.ID, Valid: true}
}

// intPtrToNullInt64 converts an int pointer to sql.NullInt64 for database storage
func (r *SqlcAuditRepository) intPtrToNullInt64(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

// SaveSharingGovernance persists site-level sharing governance data
func (r *SqlcAuditRepository) SaveSharingGovernance(ctx context.Context, siteID int64, sharingInfo *sharepoint.SharingInfo) error {
	if sharingInfo == nil {
		return nil // No governance data to save
	}

	// Convert slice to JSON string for database storage
	var segmentIDs string
	if len(sharingInfo.SiteIBSegmentIDs) > 0 {
		// Simple JSON array representation
		segmentIDs = `["` + strings.Join(sharingInfo.SiteIBSegmentIDs, `","`) + `"]`
	}

	return r.WriteQueries().UpsertSharingGovernance(ctx, db.UpsertSharingGovernanceParams{
		SiteID:                                 siteID,
		TenantID:                               r.ToNullString(sharingInfo.TenantID),
		TenantDisplayName:                      r.ToNullString(sharingInfo.TenantDisplayName),
		SharepointSiteID:                       r.ToNullString(sharingInfo.SharePointSiteID),
		AnonymousLinkExpirationRestrictionDays: r.ToNullInt64(int64(sharingInfo.AnonymousLinkExpirationRestrictionDays)),
		AnyoneLinkTrackUsers:                   r.ToNullBool(sharingInfo.AnyoneLinkTrackUsers),
		CanAddExternalPrincipal:                r.ToNullBool(sharingInfo.CanAddExternalPrincipal),
		CanAddInternalPrincipal:                r.ToNullBool(sharingInfo.CanAddInternalPrincipal),
		BlockPeoplePickerAndSharing:            r.ToNullBool(sharingInfo.BlockPeoplePickerAndSharing),
		CanRequestAccessForGrantAccess:         r.ToNullBool(sharingInfo.CanRequestAccessForGrantAccess),
		SiteIbMode:                             r.ToNullString(sharingInfo.SiteIBMode),
		SiteIbSegmentIds:                       r.ToNullString(segmentIDs),
		EnforceIbSegmentFiltering:              r.ToNullBool(sharingInfo.EnforceIBSegmentFiltering),
	})
}

// SaveSharingAbilities persists site-level sharing abilities data as JSON
func (r *SqlcAuditRepository) SaveSharingAbilities(ctx context.Context, siteID int64, abilities *sharepoint.SharingAbilities) error {
	if abilities == nil {
		return nil // No abilities data to save
	}

	// Convert to JSON for database storage
	anonymousJSON, _ := json.Marshal(abilities.AnonymousLinkAbilities)
	anyoneJSON, _ := json.Marshal(abilities.AnyoneLinkAbilities)
	orgJSON, _ := json.Marshal(abilities.OrganizationLinkAbilities)
	peopleJSON, _ := json.Marshal(abilities.PeopleSharingLinkAbilities)
	directJSON, _ := json.Marshal(abilities.DirectSharingAbilities)

	return r.WriteQueries().UpsertSharingAbilities(ctx, db.UpsertSharingAbilitiesParams{
		SiteID:                     siteID,
		CanStopSharing:             r.ToNullBool(abilities.CanStopSharing),
		AnonymousLinkAbilities:     r.ToNullString(string(anonymousJSON)),
		AnyoneLinkAbilities:        r.ToNullString(string(anyoneJSON)),
		OrganizationLinkAbilities:  r.ToNullString(string(orgJSON)),
		PeopleSharingLinkAbilities: r.ToNullString(string(peopleJSON)),
		DirectSharingAbilities:     r.ToNullString(string(directJSON)),
	})
}

// SaveRecipientLimits persists recipient limits data as JSON
func (r *SqlcAuditRepository) SaveRecipientLimits(ctx context.Context, siteID int64, limits *sharepoint.RecipientLimits) error {
	if limits == nil {
		return nil // No limits data to save
	}

	// Convert to JSON for database storage
	checkPermJSON, _ := json.Marshal(limits.CheckPermissions)
	grantAccessJSON, _ := json.Marshal(limits.GrantDirectAccess)
	shareLinkJSON, _ := json.Marshal(limits.ShareLink)
	deferRedeemJSON, _ := json.Marshal(limits.ShareLinkWithDeferRedeem)

	return r.WriteQueries().UpsertRecipientLimits(ctx, db.UpsertRecipientLimitsParams{
		SiteID:                   siteID,
		CheckPermissions:         r.ToNullString(string(checkPermJSON)),
		GrantDirectAccess:        r.ToNullString(string(grantAccessJSON)),
		ShareLink:                r.ToNullString(string(shareLinkJSON)),
		ShareLinkWithDeferRedeem: r.ToNullString(string(deferRedeemJSON)),
	})
}

// SaveSensitivityLabel persists sharing-related sensitivity label data (legacy format)
func (r *SqlcAuditRepository) SaveSensitivityLabel(ctx context.Context, siteID int64, itemGUID string, label *sharepoint.SensitivityLabelInformation) error {
	if label == nil {
		return nil // No label data to save
	}

	return r.WriteQueries().UpsertSensitivityLabel(ctx, db.UpsertSensitivityLabelParams{
		SiteID:                         siteID,
		ItemGuid:                       itemGUID,
		SensitivityLabelID:             r.ToNullString(label.ID),
		DisplayName:                    r.ToNullString(label.DisplayName),
		Color:                          r.ToNullString(label.Color),
		Tooltip:                        r.ToNullString(label.Tooltip),
		HasIrmProtection:               r.ToNullBool(label.HasIRMProtection),
		SensitivityLabelProtectionType: r.ToNullString(label.SensitivityLabelProtectionType),
	})
}

// SaveItemSensitivityLabel persists item-level sensitivity label data discovered during list processing
func (r *SqlcAuditRepository) SaveItemSensitivityLabel(ctx context.Context, label *sharepoint.ItemSensitivityLabel) error {
	if label == nil {
		return nil // No label data to save
	}

	return r.WriteQueries().UpsertItemSensitivityLabel(ctx, db.UpsertItemSensitivityLabelParams{
		SiteID:           label.SiteID,
		ItemGuid:         label.ItemGUID,
		LabelID:          r.ToNullString(label.LabelID),
		DisplayName:      r.ToNullString(label.DisplayName),
		OwnerEmail:       r.ToNullString(label.OwnerEmail),
		SetDate:          r.ToNullTime(label.SetDate),
		AssignmentMethod: r.ToNullString(label.AssignmentMethod),
		HasIrmProtection: r.ToNullBool(label.HasIRMProtection),
		ContentBits:      r.ToNullInt64(int64(label.ContentBits)),
		LabelFlags:       r.ToNullInt64(int64(label.LabelFlags)),
		DiscoveredAt:     r.ToNullTime(&label.DiscoveredAt),
		PromotionVersion: r.ToNullInt64(int64(label.PromotionVersion)),
		LabelHash:        r.ToNullString(label.LabelHash),
	})
}

// GetSitesByAuditRun retrieves all sites from a specific audit run
func (r *SqlcAuditRepository) GetSitesByAuditRun(ctx context.Context, auditRunID int64) ([]*sharepoint.Site, error) {
	rows, err := r.BaseRepository.db.ReadDB().QueryContext(ctx,
		"SELECT site_id, site_url, title, created_at, updated_at FROM sites WHERE audit_run_id = ?",
		auditRunID)
	if err != nil {
		return nil, fmt.Errorf("query sites by audit run: %w", err)
	}
	defer rows.Close()

	var sites []*sharepoint.Site
	for rows.Next() {
		site := &sharepoint.Site{}
		err := rows.Scan(&site.ID, &site.URL, &site.Title, &site.CreatedAt, &site.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan site: %w", err)
		}
		sites = append(sites, site)
	}
	return sites, nil
}

// GetWebsByAuditRun retrieves all webs from a specific audit run
func (r *SqlcAuditRepository) GetWebsByAuditRun(ctx context.Context, auditRunID int64) ([]*sharepoint.Web, error) {
	rows, err := r.BaseRepository.db.ReadDB().QueryContext(ctx,
		"SELECT site_id, web_id, title, url, template, has_unique FROM webs WHERE audit_run_id = ?",
		auditRunID)
	if err != nil {
		return nil, fmt.Errorf("query webs by audit run: %w", err)
	}
	defer rows.Close()

	var webs []*sharepoint.Web
	for rows.Next() {
		web := &sharepoint.Web{}
		var hasUnique sql.NullBool
		err := rows.Scan(&web.SiteID, &web.ID, &web.Title, &web.URL, &web.Template, &hasUnique)
		if err != nil {
			return nil, fmt.Errorf("scan web: %w", err)
		}
		web.HasUnique = r.FromNullBool(hasUnique)
		webs = append(webs, web)
	}
	return webs, nil
}

// GetListsByAuditRun retrieves all lists from a specific audit run
func (r *SqlcAuditRepository) GetListsByAuditRun(ctx context.Context, auditRunID int64) ([]*sharepoint.List, error) {
	rows, err := r.BaseRepository.db.ReadDB().QueryContext(ctx,
		"SELECT site_id, list_id, web_id, title, base_template, url, item_count, has_unique FROM lists WHERE audit_run_id = ?",
		auditRunID)
	if err != nil {
		return nil, fmt.Errorf("query lists by audit run: %w", err)
	}
	defer rows.Close()

	var lists []*sharepoint.List
	for rows.Next() {
		list := &sharepoint.List{}
		var baseTemplate, itemCount sql.NullInt64
		var hasUnique sql.NullBool
		var url sql.NullString
		err := rows.Scan(&list.SiteID, &list.ID, &list.WebID, &list.Title, &baseTemplate, &url, &itemCount, &hasUnique)
		if err != nil {
			return nil, fmt.Errorf("scan list: %w", err)
		}
		list.BaseTemplate = int(r.FromNullInt64(baseTemplate))
		list.URL = r.FromNullString(url)
		list.ItemCount = int(r.FromNullInt64(itemCount))
		list.HasUnique = r.FromNullBool(hasUnique)
		lists = append(lists, list)
	}
	return lists, nil
}

// GetItemsByAuditRun retrieves all items from a specific audit run
func (r *SqlcAuditRepository) GetItemsByAuditRun(ctx context.Context, auditRunID int64) ([]*sharepoint.Item, error) {
	rows, err := r.BaseRepository.db.ReadDB().QueryContext(ctx,
		"SELECT site_id, item_guid, list_id, item_id, list_item_guid, title, url, name, is_file, is_folder, has_unique FROM items WHERE audit_run_id = ?",
		auditRunID)
	if err != nil {
		return nil, fmt.Errorf("query items by audit run: %w", err)
	}
	defer rows.Close()

	var items []*sharepoint.Item
	for rows.Next() {
		item := &sharepoint.Item{}
		var listItemGUID, title, url, name sql.NullString
		var isFile, isFolder, hasUnique sql.NullBool
		err := rows.Scan(&item.SiteID, &item.GUID, &item.ListID, &item.ID, &listItemGUID, &title, &url, &name, &isFile, &isFolder, &hasUnique)
		if err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		item.ListItemGUID = r.FromNullString(listItemGUID)
		item.URL = r.FromNullString(url)
		item.Name = r.FromNullString(name)
		item.IsFile = r.FromNullBool(isFile)
		item.IsFolder = r.FromNullBool(isFolder)
		item.HasUnique = r.FromNullBool(hasUnique)
		items = append(items, item)
	}
	return items, nil
}
