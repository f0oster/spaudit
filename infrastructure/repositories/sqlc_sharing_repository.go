package repositories

import (
	"context"

	"spaudit/database"
	"spaudit/domain/contracts"
	"spaudit/domain/sharepoint"
	"spaudit/gen/db"
)

// SqlcSharingRepository implements contracts.SharingRepository using sqlc with read/write separation
type SqlcSharingRepository struct {
	*BaseRepository
}

// NewSqlcSharingRepository creates a new sharing repository with read/write database separation
func NewSqlcSharingRepository(database *database.Database) contracts.SharingRepository {
	return &SqlcSharingRepository{
		BaseRepository: NewBaseRepository(database),
	}
}

// GetSharingLinksForList retrieves sharing links for a list
func (r *SqlcSharingRepository) GetSharingLinksForList(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLink, error) {
	rows, err := r.ReadQueries().GetSharingLinksForList(ctx, db.GetSharingLinksForListParams{
		SiteID: siteID,
		ListID: listID,
	})
	if err != nil {
		return nil, err
	}

	// Transform SQLC rows to domain SharingLinks
	links := make([]*sharepoint.SharingLink, len(rows))
	for i, row := range rows {
		var createdBy *sharepoint.Principal
		if row.CreatedByTitle.Valid || row.CreatedByLogin.Valid {
			createdBy = &sharepoint.Principal{
				SiteID:    row.SiteID, // ✅ Complete construction with required context
				Title:     r.FromNullString(row.CreatedByTitle),
				LoginName: r.FromNullString(row.CreatedByLogin),
			}
		}

		links[i] = &sharepoint.SharingLink{
			SiteID:             row.SiteID,
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
	}
	return links, nil
}

// GetSharingLinksWithItemDataForList retrieves sharing links with item data for UI display
func (r *SqlcSharingRepository) GetSharingLinksWithItemDataForList(ctx context.Context, siteID int64, listID string) ([]*sharepoint.SharingLinkWithItemData, error) {
	rows, err := r.ReadQueries().GetSharingLinksForList(ctx, db.GetSharingLinksForListParams{
		SiteID: siteID,
		ListID: listID,
	})
	if err != nil {
		return nil, err
	}

	// Transform SQLC rows to domain SharingLinkWithItemData
	links := make([]*sharepoint.SharingLinkWithItemData, len(rows))
	for i, row := range rows {
		// Create the base sharing link (same as above method)
		var createdBy *sharepoint.Principal
		if row.CreatedByTitle.Valid || row.CreatedByLogin.Valid {
			createdBy = &sharepoint.Principal{
				SiteID:    row.SiteID,
				Title:     r.FromNullString(row.CreatedByTitle),
				LoginName: r.FromNullString(row.CreatedByLogin),
			}
		}

		link := &sharepoint.SharingLink{
			SiteID:             row.SiteID,
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
		links[i] = &sharepoint.SharingLinkWithItemData{
			SharingLink:  link,
			ItemName:     itemName,
			ItemIsFile:   isFile,
			ItemIsFolder: isFolder,
		}
	}
	return links, nil
}

// GetSharingLinkMembers retrieves members of a sharing link
func (r *SqlcSharingRepository) GetSharingLinkMembers(ctx context.Context, siteID int64, linkID string) ([]*sharepoint.Principal, error) {
	rows, err := r.ReadQueries().GetSharingLinkMembers(ctx, db.GetSharingLinkMembersParams{
		SiteID: siteID,
		LinkID: linkID,
	})
	if err != nil {
		return nil, err
	}

	// Transform SQLC rows to domain Principals
	principals := make([]*sharepoint.Principal, len(rows))
	for i, row := range rows {
		principals[i] = &sharepoint.Principal{
			SiteID:        row.SiteID,
			ID:            row.PrincipalID,
			Title:         r.FromNullString(row.Title),
			LoginName:     r.FromNullString(row.LoginName),
			Email:         r.FromNullString(row.Email),
			PrincipalType: row.PrincipalType,
		}
	}
	return principals, nil
}
