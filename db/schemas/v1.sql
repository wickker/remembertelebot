CREATE OR REPLACE FUNCTION update_updated_at() RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TABLE chats
(
    id               SERIAL PRIMARY KEY,
    telegram_chat_id INTEGER NOT NULL,
    context          JSONB     DEFAULT '{}',
    created_at       TIMESTAMP DEFAULT current_timestamp,
    updated_at       TIMESTAMP DEFAULT NULL,
    deleted_at       TIMESTAMP DEFAULT NULL
);

CREATE TRIGGER update_updated_at
    BEFORE UPDATE
    ON chats
    FOR EACH ROW
EXECUTE PROCEDURE update_updated_at();

CREATE INDEX chats_chat_id_idx ON chats (telegram_chat_id);

CREATE TABLE jobs
(
    id               SERIAL PRIMARY KEY,
    telegram_chat_id INTEGER      NOT NULL,
    is_recurring     BOOL         NOT NULL,
    river_job_id     BIGINT,
    message          TEXT         NOT NULL,
    schedule         VARCHAR(191) NOT NULL,
    name             VARCHAR(191) NOT NULL COLLATE "unicode",
    created_at       TIMESTAMP DEFAULT current_timestamp,
    updated_at       TIMESTAMP DEFAULT NULL,
    deleted_at       TIMESTAMP DEFAULT NULL
);

CREATE TRIGGER update_updated_at
    BEFORE UPDATE
    ON jobs
    FOR EACH ROW
EXECUTE PROCEDURE update_updated_at();

CREATE INDEX jobs_chat_id_idx ON jobs (telegram_chat_id);
