PRAGMA foreign_keys = ON;

-- ======================
-- Core hierarchy tables
-- ======================

CREATE TABLE sites (
  site_id    INTEGER PRIMARY KEY AUTOINCREMENT,
  site_url   TEXT NOT NULL UNIQUE,
  title      TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE webs (
  site_id         INTEGER NOT NULL REFERENCES sites(site_id),
  web_id          TEXT NOT NULL,
  title           TEXT,
  server_relative_url TEXT,
  url             TEXT,
  template        TEXT,
  has_unique      BOOLEAN DEFAULT FALSE,
  created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
  audit_run_id    INTEGER REFERENCES audit_runs(audit_run_id),
  PRIMARY KEY (site_id, web_id, audit_run_id)
);

CREATE TABLE lists (
  site_id         INTEGER NOT NULL REFERENCES sites(site_id),
  list_id         TEXT NOT NULL,
  web_id          TEXT NOT NULL,
  title           TEXT NOT NULL,
  base_template   INTEGER,
  url             TEXT,
  item_count      INTEGER DEFAULT 0,
  has_unique      BOOLEAN DEFAULT FALSE,
  hidden          BOOLEAN DEFAULT FALSE,
  created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
  audit_run_id    INTEGER REFERENCES audit_runs(audit_run_id),
  PRIMARY KEY (site_id, list_id, audit_run_id),
  FOREIGN KEY (site_id, web_id, audit_run_id) REFERENCES webs(site_id, web_id, audit_run_id)
);

CREATE TABLE items (
  site_id          INTEGER NOT NULL REFERENCES sites(site_id),
  item_guid        TEXT NOT NULL,
  list_id          TEXT NOT NULL,
  item_id          INTEGER NOT NULL,
  list_item_guid   TEXT,
  title            TEXT,
  url              TEXT,
  name             TEXT,
  file_type        TEXT,
  is_file          BOOLEAN DEFAULT FALSE,
  is_folder        BOOLEAN DEFAULT FALSE,
  has_unique       BOOLEAN DEFAULT FALSE,
  created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
  modified_at      DATETIME,
  audit_run_id     INTEGER REFERENCES audit_runs(audit_run_id),
  PRIMARY KEY (site_id, item_guid, audit_run_id),
  FOREIGN KEY (site_id, list_id, audit_run_id) REFERENCES lists(site_id, list_id, audit_run_id)
);

-- ====================
-- Security tables
-- ====================

CREATE TABLE principals (
  site_id        INTEGER NOT NULL REFERENCES sites(site_id),
  principal_id   INTEGER NOT NULL,
  title          TEXT,
  login_name     TEXT,
  email          TEXT,
  principal_type INTEGER NOT NULL,
  created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
  audit_run_id   INTEGER REFERENCES audit_runs(audit_run_id),
  PRIMARY KEY (site_id, principal_id, audit_run_id)
);

CREATE TABLE role_definitions (
  site_id     INTEGER NOT NULL REFERENCES sites(site_id),
  role_def_id INTEGER NOT NULL,
  name        TEXT NOT NULL,
  description TEXT,
  base_permissions INTEGER,
  created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
  audit_run_id INTEGER REFERENCES audit_runs(audit_run_id),
  PRIMARY KEY (site_id, role_def_id, audit_run_id)
);

CREATE TABLE role_assignments (
  site_id      INTEGER NOT NULL REFERENCES sites(site_id),
  object_type  TEXT NOT NULL,
  object_key   TEXT NOT NULL,
  principal_id INTEGER NOT NULL,
  role_def_id  INTEGER NOT NULL,
  inherited    BOOLEAN DEFAULT FALSE,
  created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
  audit_run_id INTEGER REFERENCES audit_runs(audit_run_id),
  PRIMARY KEY (site_id, object_type, object_key, principal_id, role_def_id, audit_run_id),
  FOREIGN KEY (site_id, principal_id, audit_run_id) REFERENCES principals(site_id, principal_id, audit_run_id),
  FOREIGN KEY (site_id, role_def_id, audit_run_id) REFERENCES role_definitions(site_id, role_def_id, audit_run_id)
);

-- ====================
-- Sharing tables
-- ====================

CREATE TABLE sharing_links (
  site_id                       INTEGER NOT NULL REFERENCES sites(site_id),
  link_id                       TEXT NOT NULL,
  item_guid                     TEXT,
  file_folder_unique_id         TEXT,
  url                           TEXT,
  link_kind                     INTEGER,
  scope                         INTEGER,
  is_active                     BOOLEAN DEFAULT TRUE,
  is_default                    BOOLEAN DEFAULT FALSE,
  is_edit_link                  BOOLEAN DEFAULT FALSE,
  is_review_link                BOOLEAN DEFAULT FALSE,
  is_inherited                  BOOLEAN DEFAULT FALSE,
  created_at                    DATETIME,
  created_by_principal_id       INTEGER,
  last_modified_at              DATETIME,
  last_modified_by_principal_id INTEGER,
  total_members_count           INTEGER DEFAULT 0,
  audited_at                    DATETIME DEFAULT CURRENT_TIMESTAMP,
  
  -- Governance fields
  expiration                    DATETIME,
  password_last_modified        DATETIME,
  password_last_modified_by_principal_id INTEGER,
  has_external_guest_invitees   BOOLEAN DEFAULT FALSE,
  track_link_users              BOOLEAN DEFAULT FALSE,
  is_ephemeral                  BOOLEAN DEFAULT FALSE,
  is_unhealthy                  BOOLEAN DEFAULT FALSE,
  is_address_bar_link           BOOLEAN DEFAULT FALSE,
  is_create_only_link           BOOLEAN DEFAULT FALSE,
  is_forms_link                 BOOLEAN DEFAULT FALSE,
  is_main_link                  BOOLEAN DEFAULT FALSE,
  is_manage_list_link           BOOLEAN DEFAULT FALSE,
  allows_anonymous_access       BOOLEAN DEFAULT FALSE,
  embeddable                    BOOLEAN DEFAULT FALSE,
  limit_use_to_application      BOOLEAN DEFAULT FALSE,
  restrict_to_existing_relationships BOOLEAN DEFAULT FALSE,
  blocks_download               BOOLEAN DEFAULT FALSE,
  requires_password             BOOLEAN DEFAULT FALSE,
  restricted_membership         BOOLEAN DEFAULT FALSE,
  inherited_from                TEXT,
  share_id                      TEXT,
  share_token                   TEXT,
  sharing_link_status           INTEGER,
  
  audit_run_id                  INTEGER REFERENCES audit_runs(audit_run_id),
  
  PRIMARY KEY (site_id, link_id, audit_run_id),
  UNIQUE (site_id, file_folder_unique_id, url, link_kind, scope, audit_run_id),
  FOREIGN KEY (site_id, item_guid, audit_run_id) REFERENCES items(site_id, item_guid, audit_run_id),
  FOREIGN KEY (site_id, created_by_principal_id, audit_run_id) REFERENCES principals(site_id, principal_id, audit_run_id),
  FOREIGN KEY (site_id, last_modified_by_principal_id, audit_run_id) REFERENCES principals(site_id, principal_id, audit_run_id)
);

CREATE TABLE sharing_link_members (
  site_id      INTEGER NOT NULL REFERENCES sites(site_id),
  link_id      TEXT NOT NULL,
  principal_id INTEGER NOT NULL,
  created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
  audit_run_id INTEGER REFERENCES audit_runs(audit_run_id),
  PRIMARY KEY (site_id, link_id, principal_id, audit_run_id),
  FOREIGN KEY (site_id, link_id) REFERENCES sharing_links(site_id, link_id),
  FOREIGN KEY (site_id, principal_id, audit_run_id) REFERENCES principals(site_id, principal_id, audit_run_id)
);

CREATE TABLE sharing_link_invitations (
  site_id      INTEGER NOT NULL REFERENCES sites(site_id),
  link_id      TEXT NOT NULL,
  email        TEXT NOT NULL,
  created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
  audit_run_id INTEGER REFERENCES audit_runs(audit_run_id),
  PRIMARY KEY (site_id, link_id, email, audit_run_id),
  FOREIGN KEY (site_id, link_id) REFERENCES sharing_links(site_id, link_id)
);

-- ====================
-- Governance tables
-- ====================

-- Sensitivity labeling information
CREATE TABLE sensitivity_labels (
  site_id                              INTEGER NOT NULL REFERENCES sites(site_id),
  item_guid                           TEXT NOT NULL,
  sensitivity_label_id                TEXT,
  display_name                        TEXT,
  color                               TEXT,
  tooltip                             TEXT,
  has_irm_protection                  BOOLEAN DEFAULT FALSE,
  sensitivity_label_protection_type   TEXT,
  
  label_id                            TEXT, -- vti_x005f_iplabelid
  owner_email                         TEXT, -- vti_x005f_iplabelowneremail
  set_date                            DATETIME, -- MSIP_x005f_Label_x005f_..._x005f_SetDate
  assignment_method                   TEXT, -- MSIP_x005f_Label_x005f_..._x005f_Method
  content_bits                        INTEGER DEFAULT 0, -- MSIP_x005f_Label_x005f_..._x005f_ContentBits
  label_flags                         INTEGER DEFAULT 0, -- vti_x005f_iplabelflags
  discovered_at                       DATETIME, -- When discovered during processing
  promotion_version                   INTEGER DEFAULT 0, -- vti_x005f_iplabelpromotionversion
  label_hash                          TEXT, -- vti_x005f_iplabelhash
  
  created_at                          DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (site_id, item_guid),
  FOREIGN KEY (site_id, item_guid) REFERENCES items(site_id, item_guid)
);

-- Sharing governance information per site
CREATE TABLE sharing_governance (
  site_id                                         INTEGER PRIMARY KEY REFERENCES sites(site_id),
  tenant_id                                       TEXT,
  tenant_display_name                             TEXT,
  sharepoint_site_id                              TEXT,
  anonymous_link_expiration_restriction_days      INTEGER,
  anyone_link_track_users                         BOOLEAN DEFAULT FALSE,
  can_add_external_principal                      BOOLEAN DEFAULT FALSE,
  can_add_internal_principal                      BOOLEAN DEFAULT TRUE,
  block_people_picker_and_sharing                 BOOLEAN DEFAULT FALSE,
  can_request_access_for_grant_access             BOOLEAN DEFAULT FALSE,
  site_ib_mode                                    TEXT,
  site_ib_segment_ids                             TEXT, -- JSON array
  enforce_ib_segment_filtering                    BOOLEAN DEFAULT FALSE,
  created_at                                      DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at                                      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Sharing abilities matrix
CREATE TABLE sharing_abilities (
  site_id                      INTEGER PRIMARY KEY REFERENCES sites(site_id),
  can_stop_sharing             BOOLEAN DEFAULT FALSE,
  anonymous_link_abilities     TEXT, -- JSON
  anyone_link_abilities        TEXT, -- JSON  
  organization_link_abilities  TEXT, -- JSON
  people_sharing_link_abilities TEXT, -- JSON
  direct_sharing_abilities     TEXT, -- JSON
  created_at                   DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at                   DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Recipient limits
CREATE TABLE recipient_limits (
  site_id                       INTEGER PRIMARY KEY REFERENCES sites(site_id),
  check_permissions             TEXT, -- JSON: RecipientLimitsInfo
  grant_direct_access           TEXT, -- JSON: RecipientLimitsInfo
  share_link                    TEXT, -- JSON: RecipientLimitsInfo
  share_link_with_defer_redeem  TEXT, -- JSON: RecipientLimitsInfo
  created_at                    DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at                    DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ====================
-- Jobs table
-- ====================

CREATE TABLE jobs (
  job_id       TEXT PRIMARY KEY,
  site_id      INTEGER,
  site_url     TEXT NOT NULL,
  job_type     TEXT NOT NULL,
  status       TEXT NOT NULL DEFAULT 'pending',
  item_guid    TEXT,
  progress     INTEGER DEFAULT 0,
  result       TEXT,
  error        TEXT,
  started_at   DATETIME,
  completed_at DATETIME,
  created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
  
  -- Enhanced state tracking from migration 0002
  state_json   TEXT,
  
  FOREIGN KEY (site_id) REFERENCES sites(site_id)
);

-- ====================
-- Audit Runs table
-- ====================

CREATE TABLE audit_runs (
  audit_run_id              INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id                    TEXT NOT NULL REFERENCES jobs(job_id),
  site_id                   INTEGER NOT NULL REFERENCES sites(site_id),
  
  -- When & context
  started_at                DATETIME NOT NULL,
  completed_at              DATETIME,
  audit_duration_minutes    INTEGER,
  audit_trigger             TEXT DEFAULT 'manual', -- scheduled, manual, investigation
  
  -- Size & scale metrics
  total_sites_audited       INTEGER DEFAULT 1,
  total_lists               INTEGER,
  total_items               INTEGER,
  total_unique_permissions  INTEGER,
  
  -- Security posture
  external_sharing_links    INTEGER,
  anonymous_sharing_links   INTEGER,
  high_risk_items          INTEGER,
  sensitive_labeled_items   INTEGER,
  
  -- Risk summary
  overall_risk_score        REAL,
  content_risk_level        TEXT,
  permission_risk_level     TEXT,
  sharing_risk_level        TEXT,
  
  -- Audit quality
  coverage_percentage       REAL,
  errors_encountered        INTEGER,
  
  created_at                DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ====================
-- Audit Log table for tracking audit run operations
-- ====================

CREATE TABLE audit_run_events (
  event_id        INTEGER PRIMARY KEY AUTOINCREMENT,
  audit_run_id    INTEGER NOT NULL REFERENCES audit_runs(audit_run_id),
  event_type      TEXT NOT NULL, -- 'created', 'started', 'completed', 'failed'
  event_data      TEXT, -- JSON with additional context
  created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
  created_by      TEXT NOT NULL -- job_id, user_id, or 'system'
);

-- ====================
-- Indexes for performance
-- ====================

-- Site lookups
CREATE UNIQUE INDEX idx_sites_url ON sites(site_url);

-- Web lookups
CREATE INDEX idx_webs_site_id ON webs(site_id);
CREATE INDEX idx_webs_audit_run ON webs(audit_run_id);

-- List lookups
CREATE INDEX idx_lists_site_id ON lists(site_id);
CREATE INDEX idx_lists_web_id ON lists(site_id, web_id);
CREATE INDEX idx_lists_unique ON lists(site_id, has_unique) WHERE has_unique = TRUE;
CREATE INDEX idx_lists_audit_run ON lists(audit_run_id);

-- Item lookups
CREATE INDEX idx_items_site_id ON items(site_id);
CREATE INDEX idx_items_list_id ON items(site_id, list_id);
CREATE INDEX idx_items_unique ON items(site_id, has_unique) WHERE has_unique = TRUE;
CREATE INDEX idx_items_list_item_guid ON items(site_id, list_item_guid) WHERE list_item_guid IS NOT NULL;
CREATE INDEX idx_items_audit_run ON items(audit_run_id);

-- Principal lookups
CREATE INDEX idx_principals_site_id ON principals(site_id);
CREATE INDEX idx_principals_login_name ON principals(site_id, login_name);
CREATE INDEX idx_principals_audit_run ON principals(audit_run_id);

-- Role definition lookups
CREATE INDEX idx_role_definitions_audit_run ON role_definitions(audit_run_id);

-- Role assignment lookups
CREATE INDEX idx_role_assignments_object ON role_assignments(site_id, object_type, object_key);
CREATE INDEX idx_role_assignments_principal ON role_assignments(site_id, principal_id);
CREATE INDEX idx_role_assignments_audit_run ON role_assignments(audit_run_id);

-- Sharing link lookups
CREATE INDEX idx_sharing_links_site_id ON sharing_links(site_id);
CREATE INDEX idx_sharing_links_item ON sharing_links(site_id, item_guid) WHERE item_guid IS NOT NULL;
CREATE INDEX idx_sharing_links_file_folder ON sharing_links(site_id, file_folder_unique_id) WHERE file_folder_unique_id IS NOT NULL;
CREATE INDEX idx_sharing_links_audit_run ON sharing_links(audit_run_id);

-- Governance indexes
CREATE INDEX idx_sharing_links_expiration ON sharing_links(site_id, expiration) WHERE expiration IS NOT NULL;
CREATE INDEX idx_sharing_links_external_guests ON sharing_links(site_id, has_external_guest_invitees) WHERE has_external_guest_invitees = TRUE;
CREATE INDEX idx_sharing_links_anonymous ON sharing_links(site_id, allows_anonymous_access) WHERE allows_anonymous_access = TRUE;
CREATE INDEX idx_sharing_links_unhealthy ON sharing_links(site_id, is_unhealthy) WHERE is_unhealthy = TRUE;
CREATE INDEX idx_sharing_links_password_protected ON sharing_links(site_id, requires_password) WHERE requires_password = TRUE;

-- Sensitivity label indexes
CREATE INDEX idx_sensitivity_labels_site_id ON sensitivity_labels(site_id);
CREATE INDEX idx_sensitivity_labels_label_id ON sensitivity_labels(site_id, label_id);
CREATE INDEX idx_sensitivity_labels_owner ON sensitivity_labels(site_id, owner_email) WHERE owner_email IS NOT NULL;
CREATE INDEX idx_sensitivity_labels_set_date ON sensitivity_labels(site_id, set_date) WHERE set_date IS NOT NULL;
CREATE INDEX idx_sensitivity_labels_discovered ON sensitivity_labels(discovered_at);

-- Governance table indexes
CREATE INDEX idx_sharing_governance_tenant ON sharing_governance(tenant_id);
CREATE INDEX idx_sharing_governance_sp_site ON sharing_governance(sharepoint_site_id);

-- Job lookups
CREATE INDEX idx_jobs_site_url ON jobs(site_url);
CREATE INDEX idx_jobs_site_id ON jobs(site_id) WHERE site_id IS NOT NULL;
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_created_at ON jobs(created_at);

-- Enhanced job state indexes (from migration 0002)
CREATE INDEX IF NOT EXISTS idx_jobs_state_stage ON jobs(json_extract(state_json, '$.stage'));
CREATE INDEX IF NOT EXISTS idx_jobs_state_current_operation ON jobs(json_extract(state_json, '$.current_operation'));

-- Audit run indexes (from migration 0005)
CREATE INDEX idx_audit_runs_site_id ON audit_runs(site_id);
CREATE INDEX idx_audit_runs_completed_at ON audit_runs(site_id, completed_at);
CREATE UNIQUE INDEX idx_audit_runs_job_id ON audit_runs(job_id);

-- Audit run event indexes
CREATE INDEX idx_audit_run_events_audit_run_id ON audit_run_events(audit_run_id);
CREATE INDEX idx_audit_run_events_event_type ON audit_run_events(event_type);
CREATE INDEX idx_audit_run_events_created_at ON audit_run_events(created_at);
CREATE INDEX idx_audit_run_events_created_by ON audit_run_events(created_by);

-- ====================
-- Triggers
-- ====================

-- Update sites.updated_at trigger
CREATE TRIGGER trg_sites_updated_at 
  AFTER UPDATE ON sites 
  FOR EACH ROW
BEGIN
  UPDATE sites SET updated_at = CURRENT_TIMESTAMP WHERE site_id = NEW.site_id;
END;

-- ====================
-- Schema version
-- ====================

PRAGMA user_version = 1;