ALTER TABLE jobs ADD COLUMN asynq_job_id VARCHAR(255);

CREATE INDEX jobs_asynq_job_id_idx ON chats (async_job_id);