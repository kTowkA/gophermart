BEGIN;
CREATE TABLE IF NOT EXISTS users (
    user_id uuid,
    login text,
    password_hash text,
    adding_at timestamp,
    PRIMARY KEY(user_id),
    UNIQUE(login)
);
COMMIT;