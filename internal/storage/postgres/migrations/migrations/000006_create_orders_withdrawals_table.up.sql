BEGIN;
CREATE TABLE IF NOT EXISTS withdrawals (
    withdrawn_id uuid,
    order_num text,
    sum real,
    user_id uuid,
    withdrawn_at timestamp,
    PRIMARY KEY(withdrawn_id)
);
COMMIT;