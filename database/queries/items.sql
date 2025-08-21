-- name: InsertItem :exec
INSERT INTO items (site_id, item_guid, list_item_guid, list_id, item_id, url, is_file, is_folder, has_unique, name, audit_run_id)
VALUES (sqlc.arg(site_id), sqlc.arg(item_guid), sqlc.arg(list_item_guid), sqlc.arg(list_id), sqlc.arg(item_id), sqlc.arg(url), sqlc.arg(is_file), sqlc.arg(is_folder), sqlc.arg(has_unique), sqlc.arg(name), sqlc.arg(audit_run_id));

-- name: ItemsWithUniqueForList :many
SELECT site_id, item_guid, list_item_guid, list_id, item_id, url, is_file, is_folder, has_unique, name, audit_run_id
FROM items
WHERE site_id = sqlc.arg(site_id) AND list_id = sqlc.arg(list_id) AND has_unique = 1
ORDER BY item_id
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: ItemsForList :many
SELECT site_id, item_guid, list_item_guid, list_id, item_id, url, is_file, is_folder, has_unique, name, audit_run_id
FROM items
WHERE site_id = sqlc.arg(site_id) AND list_id = sqlc.arg(list_id)
ORDER BY item_id
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: GetItemByGUID :one
SELECT site_id, item_guid, list_item_guid, list_id, item_id, url, is_file, is_folder, has_unique, name, audit_run_id
FROM items
WHERE site_id = sqlc.arg(site_id) AND item_guid = sqlc.arg(item_guid);

-- name: GetItemByListAndID :one
SELECT site_id, item_guid, list_item_guid, list_id, item_id, url, is_file, is_folder, has_unique, name, audit_run_id
FROM items
WHERE site_id = sqlc.arg(site_id) AND list_id = sqlc.arg(list_id) AND item_id = sqlc.arg(item_id);

-- name: GetItemByListAndGUID :one
SELECT site_id, item_guid, list_item_guid, list_id, url, is_file, is_folder, has_unique, name, audit_run_id
FROM items
WHERE site_id = sqlc.arg(site_id) AND list_id = sqlc.arg(list_id) AND item_guid = sqlc.arg(item_guid);

-- name: GetItemByListItemGUID :one
SELECT site_id, item_guid, list_item_guid, list_id, item_id, url, is_file, is_folder, has_unique, name, audit_run_id
FROM items
WHERE site_id = sqlc.arg(site_id) AND list_item_guid = sqlc.arg(list_item_guid);
