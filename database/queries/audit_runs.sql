-- name: CreateAuditRun :one
INSERT INTO audit_runs (job_id, site_id, started_at, audit_trigger)
VALUES (sqlc.arg(job_id), sqlc.arg(site_id), sqlc.arg(started_at), sqlc.arg(audit_trigger))
RETURNING audit_run_id;

-- name: GetAuditRun :one
SELECT audit_run_id, job_id, site_id, started_at, completed_at, audit_trigger
FROM audit_runs
WHERE audit_run_id = sqlc.arg(audit_run_id);