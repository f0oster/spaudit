-- name: CreateJob :exec
INSERT INTO jobs (
  job_id, job_type, status, site_id, site_url, item_guid, progress, state_json, started_at
) VALUES (
  sqlc.arg(job_id), sqlc.arg(job_type), sqlc.arg(status), sqlc.arg(site_id), sqlc.arg(site_url), sqlc.arg(item_guid), sqlc.arg(progress), sqlc.arg(state_json), sqlc.arg(started_at)
);

-- name: UpdateJobStatus :exec
UPDATE jobs 
SET status = sqlc.arg(status), progress = sqlc.arg(progress), state_json = sqlc.arg(state_json)
WHERE job_id = sqlc.arg(job_id);

-- name: CompleteJob :exec
UPDATE jobs 
SET status = 'completed', result = sqlc.arg(result), completed_at = CURRENT_TIMESTAMP
WHERE job_id = sqlc.arg(job_id);

-- name: FailJob :exec
UPDATE jobs 
SET status = 'failed', error = sqlc.arg(error), completed_at = CURRENT_TIMESTAMP
WHERE job_id = sqlc.arg(job_id);

-- name: GetJob :one
SELECT job_id, job_type, status, site_id, site_url, item_guid, progress, state_json, result, error, started_at, completed_at
FROM jobs
WHERE job_id = sqlc.arg(job_id);

-- name: ListActiveJobs :many
SELECT job_id, job_type, status, site_id, site_url, item_guid, progress, state_json, result, error, started_at, completed_at
FROM jobs
WHERE status IN ('pending', 'running')
ORDER BY started_at DESC;

-- name: ListActiveJobsForSite :many
SELECT job_id, job_type, status, site_id, site_url, item_guid, progress, state_json, result, error, started_at, completed_at
FROM jobs
WHERE site_id = sqlc.arg(site_id) AND status IN ('pending', 'running')
ORDER BY started_at DESC;

-- name: ListAllJobs :many
SELECT job_id, job_type, status, site_id, site_url, item_guid, progress, state_json, result, error, started_at, completed_at
FROM jobs
ORDER BY started_at DESC
LIMIT 50;

-- name: ListAllJobsForSite :many
SELECT job_id, job_type, status, site_id, site_url, item_guid, progress, state_json, result, error, started_at, completed_at
FROM jobs
WHERE site_id = sqlc.arg(site_id)
ORDER BY started_at DESC
LIMIT 50;

-- name: DeleteOldJobs :exec
DELETE FROM jobs
WHERE started_at < datetime('now', '-1 day') 
AND status IN ('completed', 'failed');

-- name: DeleteOldJobsForSite :exec
DELETE FROM jobs
WHERE site_id = sqlc.arg(site_id)
AND started_at < datetime('now', '-1 day') 
AND status IN ('completed', 'failed');

-- name: GetLastCompletedJobForSite :one
SELECT job_id, job_type, status, site_id, site_url, item_guid, progress, state_json, result, error, started_at, completed_at
FROM jobs
WHERE (site_id = sqlc.arg(site_id) OR (site_id IS NULL AND site_url = sqlc.arg(site_url))) AND status = 'completed'
ORDER BY completed_at DESC
LIMIT 1;