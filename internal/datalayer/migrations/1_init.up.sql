CREATE TABLE soundcron (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    guild_id BIGINT NOT NULL,
    soundcron_name TEXT NOT NULL,
    cron TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    UNIQUE (guild_id, soundcron_name)
);

CREATE TABLE soundcron_job (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    soundcron_id UUID NOT NULL REFERENCES soundcron(id) ON DELETE CASCADE,
    run_time TIMESTAMPTZ NOT NULL
)
