-- name: InsertList :exec
INSERT INTO lists (site_id, list_id, web_id, title, url, base_template, item_count, has_unique, audit_run_id)
VALUES (sqlc.arg(site_id), sqlc.arg(list_id), sqlc.arg(web_id), sqlc.arg(title), sqlc.arg(url), sqlc.arg(base_template), sqlc.arg(item_count), sqlc.arg(has_unique), sqlc.arg(audit_run_id));

-- name: ListsWithUnique :many
SELECT l.site_id, l.list_id, l.web_id, l.title, l.url, l.item_count, l.has_unique, w.title AS web_title, s.site_url
FROM lists l
JOIN webs w ON w.site_id = l.site_id AND w.web_id = l.web_id
JOIN sites s ON l.site_id = s.site_id
WHERE l.has_unique = 1
ORDER BY s.site_url, w.title, l.title;

-- name: ListsWithUniqueForSite :many
SELECT l.site_id, l.list_id, l.web_id, l.title, l.url, l.item_count, l.has_unique, w.title AS web_title
FROM lists l
JOIN webs w ON w.site_id = l.site_id AND w.web_id = l.web_id
WHERE l.site_id = sqlc.arg(site_id) AND l.has_unique = 1
ORDER BY w.title, l.title;

-- name: ListsAll :many
SELECT l.site_id, l.list_id, l.web_id, l.title, l.url, l.item_count, l.has_unique, w.title AS web_title, s.site_url
FROM lists l
JOIN webs w ON w.site_id = l.site_id AND w.web_id = l.web_id
JOIN sites s ON l.site_id = s.site_id
ORDER BY s.site_url, w.title, l.title;

-- name: GetListsForSite :many
SELECT l.site_id, l.list_id, l.web_id, l.title, l.url, l.item_count, l.has_unique, w.title AS web_title
FROM lists l
JOIN webs w ON w.site_id = l.site_id AND w.web_id = l.web_id
WHERE l.site_id = sqlc.arg(site_id)
ORDER BY w.title, l.title;

-- name: GetList :one
SELECT site_id, list_id, web_id, title, url, base_template, item_count, has_unique, audit_run_id
FROM lists WHERE site_id = sqlc.arg(site_id) AND list_id = sqlc.arg(list_id);

-- name: GetListsByWebID :many
SELECT site_id, list_id, web_id, title, url, base_template, item_count, has_unique, audit_run_id
FROM lists WHERE site_id = sqlc.arg(site_id) AND web_id = sqlc.arg(web_id)
ORDER BY title;
