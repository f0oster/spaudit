-- name: InsertWeb :exec
INSERT INTO webs (site_id, web_id, url, title, template, has_unique, audit_run_id)
VALUES (sqlc.arg(site_id), sqlc.arg(web_id), sqlc.arg(url), sqlc.arg(title), sqlc.arg(template), sqlc.arg(has_unique), sqlc.arg(audit_run_id));

-- name: ListWebs :many
SELECT w.site_id, w.web_id, w.url, w.title, w.template, w.has_unique, w.audit_run_id, s.site_url
FROM webs w
JOIN sites s ON w.site_id = s.site_id
ORDER BY s.site_url, w.title;

-- name: ListWebsForSite :many
SELECT site_id, web_id, url, title, template, has_unique, audit_run_id
FROM webs
WHERE site_id = sqlc.arg(site_id)
ORDER BY title;

-- name: GetWeb :one
SELECT site_id, web_id, url, title, template, has_unique, audit_run_id
FROM webs
WHERE site_id = sqlc.arg(site_id) AND web_id = sqlc.arg(web_id);
