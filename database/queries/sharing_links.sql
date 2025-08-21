-- name: InsertSharingLink :one
INSERT INTO sharing_links (
  site_id,
  link_id,
  item_guid,
  file_folder_unique_id,
  url,
  link_kind,
  scope,
  is_active,
  is_default,
  is_edit_link,
  is_review_link,
  is_inherited,
  created_at,
  created_by_principal_id,
  last_modified_at,
  last_modified_by_principal_id,
  total_members_count,
  -- Enhanced governance fields
  expiration,
  password_last_modified,
  password_last_modified_by_principal_id,
  has_external_guest_invitees,
  track_link_users,
  is_ephemeral,
  is_unhealthy,
  is_address_bar_link,
  is_create_only_link,
  is_forms_link,
  is_main_link,
  is_manage_list_link,
  allows_anonymous_access,
  embeddable,
  limit_use_to_application,
  restrict_to_existing_relationships,
  blocks_download,
  requires_password,
  restricted_membership,
  inherited_from,
  share_id,
  share_token,
  sharing_link_status,
  audit_run_id
)
VALUES (
  sqlc.arg(site_id),
  sqlc.arg(link_id),
  sqlc.arg(item_guid),
  sqlc.arg(file_folder_unique_id),
  sqlc.arg(url),
  sqlc.arg(link_kind),
  sqlc.arg(scope),
  sqlc.arg(is_active),
  sqlc.arg(is_default),
  sqlc.arg(is_edit_link),
  sqlc.arg(is_review_link),
  sqlc.arg(is_inherited),
  sqlc.arg(created_at),
  sqlc.arg(created_by_principal_id),
  sqlc.arg(last_modified_at),
  sqlc.arg(last_modified_by_principal_id),
  sqlc.arg(total_members_count),
  -- Enhanced governance fields
  sqlc.arg(expiration),
  sqlc.arg(password_last_modified),
  sqlc.arg(password_last_modified_by_principal_id),
  sqlc.arg(has_external_guest_invitees),
  sqlc.arg(track_link_users),
  sqlc.arg(is_ephemeral),
  sqlc.arg(is_unhealthy),
  sqlc.arg(is_address_bar_link),
  sqlc.arg(is_create_only_link),
  sqlc.arg(is_forms_link),
  sqlc.arg(is_main_link),
  sqlc.arg(is_manage_list_link),
  sqlc.arg(allows_anonymous_access),
  sqlc.arg(embeddable),
  sqlc.arg(limit_use_to_application),
  sqlc.arg(restrict_to_existing_relationships),
  sqlc.arg(blocks_download),
  sqlc.arg(requires_password),
  sqlc.arg(restricted_membership),
  sqlc.arg(inherited_from),
  sqlc.arg(share_id),
  sqlc.arg(share_token),
  sqlc.arg(sharing_link_status),
  sqlc.arg(audit_run_id)
)
RETURNING link_id;

-- name: GetLinkIDByUrlKindScope :one
SELECT link_id
FROM sharing_links
WHERE site_id = sqlc.arg(site_id)
  AND file_folder_unique_id = sqlc.arg(file_folder_unique_id)
  AND url       = sqlc.arg(url)
  AND link_kind = sqlc.arg(link_kind)
  AND scope     = sqlc.arg(scope)
LIMIT 1;

-- name: ClearMembersForLink :exec
DELETE FROM sharing_link_members WHERE site_id = sqlc.arg(site_id) AND link_id = sqlc.arg(link_id);

-- name: AddMemberToLink :exec
INSERT INTO sharing_link_members (site_id, link_id, principal_id, audit_run_id)
VALUES (sqlc.arg(site_id), sqlc.arg(link_id), sqlc.arg(principal_id), sqlc.arg(audit_run_id));

-- name: GetFlexibleSharingLinks :many
-- Find principals with Flexible sharing link patterns in login_name
SELECT site_id, principal_id, login_name, title, email
FROM principals 
WHERE site_id = sqlc.arg(site_id)
  AND login_name LIKE '%SharingLinks.%.Flexible.%'
  AND login_name IS NOT NULL;

-- name: GetAllSharingLinks :many
-- Find all principals with any SharingLinks patterns in login_name
SELECT site_id, principal_id, login_name, title, email
FROM principals 
WHERE site_id = sqlc.arg(site_id)
  AND login_name LIKE '%SharingLinks.%.%'
  AND login_name IS NOT NULL;

-- name: GetSharingLinksForList :many
-- Get all sharing links for items in a specific list with item and principal details
SELECT 
  sl.site_id,
  sl.link_id,
  sl.item_guid,
  sl.file_folder_unique_id,
  sl.url,
  sl.link_kind,
  sl.scope,
  sl.is_active,
  sl.is_default,
  sl.is_edit_link,
  sl.is_review_link,
  sl.created_at,
  sl.last_modified_at,
  sl.total_members_count,
  (SELECT COUNT(*) FROM sharing_link_members WHERE site_id = sl.site_id AND link_id = sl.link_id) as actual_members_count,
  i.name as item_name,
  i.url as item_url,
  i.is_file,
  i.is_folder,
  cb.title as created_by_title,
  cb.login_name as created_by_login,
  mb.title as modified_by_title,
  mb.login_name as modified_by_login
FROM sharing_links sl
LEFT JOIN items i ON (sl.site_id = i.site_id AND (sl.item_guid = i.item_guid OR sl.file_folder_unique_id = i.item_guid))
LEFT JOIN principals cb ON sl.site_id = cb.site_id AND sl.created_by_principal_id = cb.principal_id
LEFT JOIN principals mb ON sl.site_id = mb.site_id AND sl.last_modified_by_principal_id = mb.principal_id
WHERE sl.site_id = sqlc.arg(site_id) AND i.list_id = sqlc.arg(list_id)
  AND sl.is_active = 1
ORDER BY sl.created_at DESC, sl.link_id;

-- name: GetSharingLinkMembers :many
-- Get all members (principals) for a specific sharing link
SELECT 
  p.site_id,
  p.principal_id,
  p.title,
  p.login_name,
  p.email,
  p.principal_type
FROM sharing_link_members slm
JOIN principals p ON slm.site_id = p.site_id AND slm.principal_id = p.principal_id
WHERE slm.site_id = sqlc.arg(site_id) AND slm.link_id = sqlc.arg(link_id)
ORDER BY p.title;

-- ==================================
-- Governance table queries
-- ==================================

-- name: UpsertSharingGovernance :exec
INSERT INTO sharing_governance (
  site_id,
  tenant_id,
  tenant_display_name,
  sharepoint_site_id,
  anonymous_link_expiration_restriction_days,
  anyone_link_track_users,
  can_add_external_principal,
  can_add_internal_principal,
  block_people_picker_and_sharing,
  can_request_access_for_grant_access,
  site_ib_mode,
  site_ib_segment_ids,
  enforce_ib_segment_filtering
) VALUES (
  sqlc.arg(site_id),
  sqlc.arg(tenant_id),
  sqlc.arg(tenant_display_name),
  sqlc.arg(sharepoint_site_id),
  sqlc.arg(anonymous_link_expiration_restriction_days),
  sqlc.arg(anyone_link_track_users),
  sqlc.arg(can_add_external_principal),
  sqlc.arg(can_add_internal_principal),
  sqlc.arg(block_people_picker_and_sharing),
  sqlc.arg(can_request_access_for_grant_access),
  sqlc.arg(site_ib_mode),
  sqlc.arg(site_ib_segment_ids),
  sqlc.arg(enforce_ib_segment_filtering)
)
ON CONFLICT(site_id) DO UPDATE SET
  tenant_id                                  = excluded.tenant_id,
  tenant_display_name                        = excluded.tenant_display_name,
  sharepoint_site_id                         = excluded.sharepoint_site_id,
  anonymous_link_expiration_restriction_days = excluded.anonymous_link_expiration_restriction_days,
  anyone_link_track_users                    = excluded.anyone_link_track_users,
  can_add_external_principal                 = excluded.can_add_external_principal,
  can_add_internal_principal                 = excluded.can_add_internal_principal,
  block_people_picker_and_sharing            = excluded.block_people_picker_and_sharing,
  can_request_access_for_grant_access        = excluded.can_request_access_for_grant_access,
  site_ib_mode                               = excluded.site_ib_mode,
  site_ib_segment_ids                        = excluded.site_ib_segment_ids,
  enforce_ib_segment_filtering               = excluded.enforce_ib_segment_filtering,
  updated_at                                 = CURRENT_TIMESTAMP;

-- name: GetSharingGovernance :one
SELECT 
  site_id,
  tenant_id,
  tenant_display_name,
  sharepoint_site_id,
  anonymous_link_expiration_restriction_days,
  anyone_link_track_users,
  can_add_external_principal,
  can_add_internal_principal,
  block_people_picker_and_sharing,
  can_request_access_for_grant_access,
  site_ib_mode,
  site_ib_segment_ids,
  enforce_ib_segment_filtering
FROM sharing_governance
WHERE site_id = sqlc.arg(site_id);

-- name: UpsertSharingAbilities :exec
INSERT INTO sharing_abilities (
  site_id,
  can_stop_sharing,
  anonymous_link_abilities,
  anyone_link_abilities,
  organization_link_abilities,
  people_sharing_link_abilities,
  direct_sharing_abilities
) VALUES (
  sqlc.arg(site_id),
  sqlc.arg(can_stop_sharing),
  sqlc.arg(anonymous_link_abilities),
  sqlc.arg(anyone_link_abilities),
  sqlc.arg(organization_link_abilities),
  sqlc.arg(people_sharing_link_abilities),
  sqlc.arg(direct_sharing_abilities)
)
ON CONFLICT(site_id) DO UPDATE SET
  can_stop_sharing                  = excluded.can_stop_sharing,
  anonymous_link_abilities          = excluded.anonymous_link_abilities,
  anyone_link_abilities             = excluded.anyone_link_abilities,
  organization_link_abilities       = excluded.organization_link_abilities,
  people_sharing_link_abilities     = excluded.people_sharing_link_abilities,
  direct_sharing_abilities          = excluded.direct_sharing_abilities,
  updated_at                        = CURRENT_TIMESTAMP;

-- name: GetSharingAbilities :one
SELECT 
  site_id,
  can_stop_sharing,
  anonymous_link_abilities,
  anyone_link_abilities,
  organization_link_abilities,
  people_sharing_link_abilities,
  direct_sharing_abilities
FROM sharing_abilities
WHERE site_id = sqlc.arg(site_id);

-- name: UpsertRecipientLimits :exec
INSERT INTO recipient_limits (
  site_id,
  check_permissions,
  grant_direct_access,
  share_link,
  share_link_with_defer_redeem
) VALUES (
  sqlc.arg(site_id),
  sqlc.arg(check_permissions),
  sqlc.arg(grant_direct_access),
  sqlc.arg(share_link),
  sqlc.arg(share_link_with_defer_redeem)
)
ON CONFLICT(site_id) DO UPDATE SET
  check_permissions               = excluded.check_permissions,
  grant_direct_access             = excluded.grant_direct_access,
  share_link                      = excluded.share_link,
  share_link_with_defer_redeem    = excluded.share_link_with_defer_redeem,
  updated_at                      = CURRENT_TIMESTAMP;

-- name: GetRecipientLimits :one
SELECT 
  site_id,
  check_permissions,
  grant_direct_access,
  share_link,
  share_link_with_defer_redeem
FROM recipient_limits
WHERE site_id = sqlc.arg(site_id);

-- name: UpsertItemSensitivityLabel :exec
INSERT INTO sensitivity_labels (
  site_id,
  item_guid,
  label_id,
  display_name,
  owner_email,
  set_date,
  assignment_method,
  has_irm_protection,
  content_bits,
  label_flags,
  discovered_at,
  promotion_version,
  label_hash
) VALUES (
  sqlc.arg(site_id),
  sqlc.arg(item_guid),
  sqlc.arg(label_id),
  sqlc.arg(display_name),
  sqlc.arg(owner_email),
  sqlc.arg(set_date),
  sqlc.arg(assignment_method),
  sqlc.arg(has_irm_protection),
  sqlc.arg(content_bits),
  sqlc.arg(label_flags),
  sqlc.arg(discovered_at),
  sqlc.arg(promotion_version),
  sqlc.arg(label_hash)
)
ON CONFLICT(site_id, item_guid) DO UPDATE SET
  label_id                = excluded.label_id,
  display_name            = excluded.display_name,
  owner_email             = excluded.owner_email,
  set_date                = excluded.set_date,
  assignment_method       = excluded.assignment_method,
  has_irm_protection      = excluded.has_irm_protection,
  content_bits            = excluded.content_bits,
  label_flags             = excluded.label_flags,
  discovered_at           = excluded.discovered_at,
  promotion_version       = excluded.promotion_version,
  label_hash              = excluded.label_hash;

-- name: GetItemSensitivityLabel :one
SELECT 
  site_id,
  item_guid,
  label_id,
  display_name,
  owner_email,
  set_date,
  assignment_method,
  has_irm_protection,
  content_bits,
  label_flags,
  discovered_at,
  promotion_version,
  label_hash
FROM sensitivity_labels
WHERE site_id = sqlc.arg(site_id) AND item_guid = sqlc.arg(item_guid);

-- name: GetSensitivityLabelsForSite :many
SELECT 
  site_id,
  item_guid,
  label_id,
  display_name,
  owner_email,
  set_date,
  assignment_method,
  has_irm_protection,
  content_bits,
  label_flags,
  discovered_at,
  promotion_version,
  label_hash
FROM sensitivity_labels
WHERE site_id = sqlc.arg(site_id)
  AND label_id IS NOT NULL
ORDER BY discovered_at DESC;

-- name: UpsertSensitivityLabel :exec
INSERT INTO sensitivity_labels (
  site_id,
  item_guid,
  sensitivity_label_id,
  display_name,
  color,
  tooltip,
  has_irm_protection,
  sensitivity_label_protection_type
) VALUES (
  sqlc.arg(site_id),
  sqlc.arg(item_guid),
  sqlc.arg(sensitivity_label_id),
  sqlc.arg(display_name),
  sqlc.arg(color),
  sqlc.arg(tooltip),
  sqlc.arg(has_irm_protection),
  sqlc.arg(sensitivity_label_protection_type)
)
ON CONFLICT(site_id, item_guid) DO UPDATE SET
  sensitivity_label_id                = excluded.sensitivity_label_id,
  display_name                        = excluded.display_name,
  color                               = excluded.color,
  tooltip                             = excluded.tooltip,
  has_irm_protection                  = excluded.has_irm_protection,
  sensitivity_label_protection_type   = excluded.sensitivity_label_protection_type;
