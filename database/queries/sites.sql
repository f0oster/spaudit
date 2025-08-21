-- name: UpsertSite :one
INSERT INTO sites (site_url, title, updated_at)
VALUES (sqlc.arg(site_url), sqlc.arg(title), CURRENT_TIMESTAMP)
ON CONFLICT(site_url) DO UPDATE SET
  title=excluded.title,
  updated_at=excluded.updated_at
RETURNING site_id;

-- name: GetSiteByURL :one
SELECT site_id, site_url, title, created_at, updated_at
FROM sites
WHERE site_url = sqlc.arg(site_url);

-- name: GetSiteByID :one
SELECT site_id, site_url, title, created_at, updated_at
FROM sites
WHERE site_id = sqlc.arg(site_id);

-- name: ListSites :many
SELECT site_id, site_url, title, created_at, updated_at
FROM sites
ORDER BY title;
