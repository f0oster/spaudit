-- name: InsertPrincipal :exec
INSERT INTO principals (site_id, principal_id, principal_type, title, login_name, email, audit_run_id)
VALUES (sqlc.arg(site_id), sqlc.arg(principal_id), sqlc.arg(principal_type), sqlc.arg(title), sqlc.arg(login_name), sqlc.arg(email), sqlc.arg(audit_run_id));

-- name: UpsertPrincipalByLogin :one
INSERT INTO principals (site_id, principal_type, title, login_name, email)
VALUES (sqlc.arg(site_id), sqlc.arg(principal_type), sqlc.arg(title), sqlc.arg(login_name), sqlc.arg(email))
ON CONFLICT(site_id, login_name) DO UPDATE SET
  principal_type = excluded.principal_type,
  title          = COALESCE(excluded.title, principals.title),
  email          = COALESCE(excluded.email, principals.email)
RETURNING principal_id;

-- name: InsertRoleDefinition :exec
INSERT INTO role_definitions (site_id, role_def_id, name, description, audit_run_id)
VALUES (sqlc.arg(site_id), sqlc.arg(role_def_id), sqlc.arg(name), sqlc.arg(description), sqlc.arg(audit_run_id));

-- name: DeleteRoleAssignmentsForObject :exec
DELETE FROM role_assignments
WHERE site_id = sqlc.arg(site_id) AND object_type = sqlc.arg(object_type) AND object_key = sqlc.arg(object_key);

-- name: InsertRoleAssignment :exec
INSERT INTO role_assignments (site_id, object_type, object_key, principal_id, role_def_id, inherited, audit_run_id)
VALUES (sqlc.arg(site_id), sqlc.arg(object_type), sqlc.arg(object_key), sqlc.arg(principal_id), sqlc.arg(role_def_id), sqlc.arg(inherited), sqlc.arg(audit_run_id));

-- name: GetAssignmentsForObject :many
SELECT ra.principal_id, p.title AS principal_title, p.login_name, p.principal_type,
       ra.role_def_id, rd.name AS role_name, rd.description, ra.inherited
FROM role_assignments ra
JOIN principals p ON p.site_id = ra.site_id AND p.principal_id = ra.principal_id
JOIN role_definitions rd ON rd.site_id = ra.site_id AND rd.role_def_id = ra.role_def_id
WHERE ra.site_id = sqlc.arg(site_id) AND ra.object_type = sqlc.arg(object_type) AND ra.object_key = sqlc.arg(object_key)
ORDER BY principal_title, role_name;

-- name: GetWebIdForObject :one
SELECT 
  CASE sqlc.arg(object_type)
    WHEN 'web' THEN sqlc.arg(object_key)
    WHEN 'list' THEN l.web_id
    WHEN 'item' THEN parent_list.web_id
  END as web_id
FROM (SELECT 1 as dummy) d
LEFT JOIN lists l ON sqlc.arg(object_type) = 'list' AND sqlc.arg(site_id) = l.site_id AND sqlc.arg(object_key) = l.list_id
LEFT JOIN items i ON sqlc.arg(object_type) = 'item' AND sqlc.arg(site_id) = i.site_id AND sqlc.arg(object_key) = i.item_guid
LEFT JOIN lists parent_list ON i.site_id = parent_list.site_id AND i.list_id = parent_list.list_id
LIMIT 1;

-- name: GetRootPermissionsForPrincipalInWeb :many
SELECT ra.object_type, ra.object_key, rd.name as role_name,
       CASE ra.object_type
         WHEN 'list' THEN l.title
         WHEN 'web' THEN w.title
         WHEN 'item' THEN i.name
       END as object_name
FROM role_assignments ra
JOIN role_definitions rd ON ra.site_id = rd.site_id AND ra.role_def_id = rd.role_def_id
LEFT JOIN lists l ON ra.object_type = 'list' AND ra.site_id = l.site_id AND ra.object_key = l.list_id AND l.web_id = sqlc.arg(web_id)
LEFT JOIN webs w ON ra.object_type = 'web' AND ra.site_id = w.site_id AND ra.object_key = w.web_id AND w.web_id = sqlc.arg(web_id)
LEFT JOIN items i ON ra.object_type = 'item' AND ra.site_id = i.site_id AND ra.object_key = i.item_guid
LEFT JOIN lists parent_list ON i.site_id = parent_list.site_id AND i.list_id = parent_list.list_id AND parent_list.web_id = sqlc.arg(web_id)
WHERE ra.site_id = sqlc.arg(site_id) AND ra.principal_id = sqlc.arg(principal_id)
  AND rd.name NOT LIKE '%Limited%'
  AND (
    (ra.object_type = 'web' AND w.site_id = sqlc.arg(site_id) AND w.web_id = sqlc.arg(web_id))
    OR (ra.object_type = 'list' AND l.site_id = sqlc.arg(site_id) AND l.web_id = sqlc.arg(web_id))
    OR (ra.object_type = 'item' AND parent_list.site_id = sqlc.arg(site_id) AND parent_list.web_id = sqlc.arg(web_id))
  );

-- name: GetSharedItemForSharingLink :one
SELECT i.name as item_name, l.title as list_title
FROM sharing_links sl
LEFT JOIN items i ON sl.site_id = i.site_id AND sl.item_guid = i.item_guid
LEFT JOIN lists l ON i.site_id = l.site_id AND i.list_id = l.list_id
WHERE sl.site_id = sqlc.arg(site_id) AND sl.file_folder_unique_id = sqlc.arg(file_folder_guid)
LIMIT 1;
