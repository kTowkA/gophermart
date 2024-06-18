BEGIN;
CREATE TABLE IF NOT EXISTS statuses (
    status_id serial,
    value text,
    PRIMARY KEY(status_id),
    UNIQUE(value)
);
COMMIT;