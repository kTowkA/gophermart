BEGIN;
CREATE TABLE IF NOT EXISTS replenishments (
    replenishment_id uuid,
    order_id uuid,
    sum real,
    replenishment_at timestamp,
    PRIMARY KEY(replenishment_id)
);
COMMIT;