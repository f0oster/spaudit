package repositories

import (
	"context"

	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/gen/db"
)

// ScopedSharingRepository wraps a SharingRepository with automatic site and audit run scoping
type ScopedSharingRepository struct {
	*BaseRepository
	queries     *db.Queries
	siteID      int64
	auditRunID  int64
}

// NewScopedSharingRepository creates a new scoped sharing repository
func NewScopedSharingRepository(baseRepo *BaseRepository, queries *db.Queries, siteID, auditRunID int64) contracts.SharingRepository {
	return &ScopedSharingRepository{
		BaseRepository: baseRepo,
		queries:       queries,
		siteID:        siteID,
		auditRunID:    auditRunID,
	}
}

// GetSharingLinksForList retrieves sharing links for a list scoped to audit run
func (r *ScopedSharingRepository) GetSharingLinksForList(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLink, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	rows, err := r.queries.GetSharingLinksForListByAuditRun(ctx, db.GetSharingLinksForListByAuditRunParams{
		SiteID: r.siteID,
		ListID: listID,
		AuditRunID: r.auditRunID,
	})
	if err != nil {
		return nil, err
	}

	// Transform SQLC rows to domain objects
	var links []*sharepoint.SharingLink
	for _, row := range rows {
		
		var createdBy *sharepoint.Principal
		if row.CreatedByTitle.Valid || row.CreatedByLogin.Valid {
			createdBy = &sharepoint.Principal{
				SiteID:     r.siteID,
				Title:      r.FromNullString(row.CreatedByTitle),
				LoginName:  r.FromNullString(row.CreatedByLogin),
							}
		}

		link := &sharepoint.SharingLink{
			SiteID:             r.siteID,
			ID:                 row.LinkID,
			ItemGUID:           r.FromNullString(row.ItemGuid),
			FileFolderUniqueID: r.FromNullString(row.FileFolderUniqueID),
			URL:                r.FromNullString(row.Url),
			LinkKind:           int(r.FromNullInt64(row.LinkKind)),
			Scope:              int(r.FromNullInt64(row.Scope)),
			IsActive:           r.FromNullBool(row.IsActive),
			IsDefault:          r.FromNullBool(row.IsDefault),
			IsEditLink:         r.FromNullBool(row.IsEditLink),
			IsReviewLink:       r.FromNullBool(row.IsReviewLink),
			CreatedAt:          r.FromNullTime(row.CreatedAt),
			CreatedBy:          createdBy,
			TotalMembersCount:  int(row.ActualMembersCount),
		}
		
		links = append(links, link)
	}
	
	return links, nil
}

// GetSharingLinksWithItemDataForList retrieves sharing links with item data for UI display scoped to audit run
func (r *ScopedSharingRepository) GetSharingLinksWithItemDataForList(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLinkWithItemData, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	rows, err := r.queries.GetSharingLinksForListByAuditRun(ctx, db.GetSharingLinksForListByAuditRunParams{
		SiteID: r.siteID,
		ListID: listID,
		AuditRunID: r.auditRunID,
	})
	if err != nil {
		return nil, err
	}

	// Transform SQLC rows to domain SharingLinkWithItemData
	var links []*sharepoint.SharingLinkWithItemData
	for _, row := range rows {
		// Create the base sharing link
		var createdBy *sharepoint.Principal
		if row.CreatedByTitle.Valid || row.CreatedByLogin.Valid {
			createdBy = &sharepoint.Principal{
				SiteID:     r.siteID,
				Title:      r.FromNullString(row.CreatedByTitle),
				LoginName:  r.FromNullString(row.CreatedByLogin),
							}
		}

		link := &sharepoint.SharingLink{
			SiteID:             r.siteID,
			ID:                 row.LinkID,
			ItemGUID:           r.FromNullString(row.ItemGuid),
			FileFolderUniqueID: r.FromNullString(row.FileFolderUniqueID),
			URL:                r.FromNullString(row.Url),
			LinkKind:           int(r.FromNullInt64(row.LinkKind)),
			Scope:              int(r.FromNullInt64(row.Scope)),
			IsActive:           r.FromNullBool(row.IsActive),
			IsDefault:          r.FromNullBool(row.IsDefault),
			IsEditLink:         r.FromNullBool(row.IsEditLink),
			IsReviewLink:       r.FromNullBool(row.IsReviewLink),
			CreatedAt:          r.FromNullTime(row.CreatedAt),
			CreatedBy:          createdBy,
			TotalMembersCount:  int(row.ActualMembersCount),
		}

		// Extract item data from the row
		itemName := r.FromNullString(row.ItemName)
		isFile := r.FromNullBool(row.IsFile)
		isFolder := r.FromNullBool(row.IsFolder)

		// Return enriched domain model
		linkWithData := &sharepoint.SharingLinkWithItemData{
			SharingLink:  link,
			ItemName:     itemName,
			ItemIsFile:   isFile,
			ItemIsFolder: isFolder,
		}
		
		links = append(links, linkWithData)
	}
	
	return links, nil
}

// GetSharingLinkMembers retrieves members of a sharing link scoped to audit run
func (r *ScopedSharingRepository) GetSharingLinkMembers(ctx context.Context, siteID int64, linkID string) ([]*sharepoint.Principal, error) {
	// Verify the requested siteID matches our scoped siteID
	if siteID != r.siteID {
		return nil, contracts.ErrSiteScopeMismatch
	}

	rows, err := r.queries.GetSharingLinkMembersByAuditRun(ctx, db.GetSharingLinkMembersByAuditRunParams{
		SiteID: r.siteID,
		LinkID: linkID,
		AuditRunID: r.auditRunID,
	})
	if err != nil {
		return nil, err
	}

	// Transform SQLC rows to domain Principals
	var principals []*sharepoint.Principal
	for _, row := range rows {
		
		principal := &sharepoint.Principal{
			SiteID:        r.siteID,
			ID:            row.PrincipalID,
			Title:         r.FromNullString(row.Title),
			LoginName:     r.FromNullString(row.LoginName),
			Email:         r.FromNullString(row.Email),
			PrincipalType: row.PrincipalType,
		}
		
		principals = append(principals, principal)
	}
	
	return principals, nil
}

