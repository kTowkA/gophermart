BEGIN;
CREATE TABLE IF NOT EXISTS orders_statuses (
    order_id uuid,
    status_id integer,
    adding_at timestamp,
    update_at timestamp,
    PRIMARY KEY(order_id,status_id)
);
COMMIT;