-- name: CreateAuditRun :one
INSERT INTO audit_runs (job_id, site_id, started_at, audit_trigger)
VALUES (sqlc.arg(job_id), sqlc.arg(site_id), sqlc.arg(started_at), sqlc.arg(audit_trigger))
RETURNING audit_run_id;

-- name: GetAuditRun :one
SELECT audit_run_id, job_id, site_id, started_at, completed_at, audit_trigger
FROM audit_runs
WHERE audit_run_id = sqlc.arg(audit_run_id);

-- name: GetAuditRunsForSite :many
SELECT audit_run_id, job_id, site_id, started_at, completed_at, audit_trigger
FROM audit_runs
WHERE site_id = sqlc.arg(site_id)
ORDER BY started_at DESC
LIMIT sqlc.arg(limit_count);

-- name: GetLatestAuditRunForSite :one
SELECT audit_run_id, job_id, site_id, started_at, completed_at, audit_trigger
FROM audit_runs
WHERE site_id = sqlc.arg(site_id)
ORDER BY started_at DESC
LIMIT 1;

-- name: CompleteAuditRun :exec
UPDATE audit_runs
SET completed_at = CURRENT_TIMESTAMP
WHERE audit_run_id = sqlc.arg(audit_run_id);

-- name: CompleteAuditRunByJobID :exec
UPDATE audit_runs
SET completed_at = CURRENT_TIMESTAMP
WHERE job_id = sqlc.arg(job_id);

-- name: MigrateCompletedAuditRuns :exec
UPDATE audit_runs 
SET completed_at = (
    SELECT j.completed_at 
    FROM jobs j 
    WHERE j.job_id = audit_runs.job_id 
    AND j.completed_at IS NOT NULL
)
WHERE completed_at IS NULL 
AND job_id IN (
    SELECT job_id 
    FROM jobs 
    WHERE completed_at IS NOT NULL
);