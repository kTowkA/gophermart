BEGIN;
CREATE TABLE IF NOT EXISTS orders (
    order_id uuid,
    order_num text,
    status_id integer,
    adding_at timestamp,
    update_at timestamp,
    PRIMARY KEY(order_id)
);
COMMIT;