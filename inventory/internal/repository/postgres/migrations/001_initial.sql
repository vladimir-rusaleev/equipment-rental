CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE TABLE models (
    id text PRIMARY KEY,
    name text NOT NULL,
    category text NOT NULL,
    description text NOT NULL,
    daily_price bigint NOT NULL CHECK (daily_price > 0 AND daily_price <= 100000000),
    image_urls jsonb NOT NULL DEFAULT '[]'
);
CREATE INDEX models_category_idx ON models(category);
CREATE TABLE units (
    id text PRIMARY KEY,
    model_id text NOT NULL REFERENCES models(id),
    state text NOT NULL CHECK (state IN ('ready','maintenance','retired'))
);
CREATE INDEX units_model_idx ON units(model_id);
CREATE TABLE reservations (
    order_id text PRIMARY KEY,
    state text NOT NULL CHECK (state IN ('reserved','active','completed','cancelled')),
    request jsonb NOT NULL
);
CREATE TABLE reservation_items (
    order_id text NOT NULL REFERENCES reservations(order_id),
    line_no integer NOT NULL,
    model_id text NOT NULL REFERENCES models(id),
    name text NOT NULL,
    quantity integer NOT NULL CHECK (quantity > 0),
    starts_on date NOT NULL,
    ends_on date NOT NULL CHECK (ends_on > starts_on),
    daily_price bigint NOT NULL CHECK (daily_price > 0),
    total bigint NOT NULL CHECK (total > 0),
    PRIMARY KEY(order_id,line_no)
);
CREATE TABLE allocations (
    order_id text NOT NULL,
    line_no integer NOT NULL,
    unit_id text NOT NULL REFERENCES units(id),
    period daterange NOT NULL,
    cancelled boolean NOT NULL DEFAULT false,
    issued boolean NOT NULL DEFAULT false,
    returned boolean NOT NULL DEFAULT false,
    PRIMARY KEY(order_id,line_no,unit_id),
    FOREIGN KEY(order_id,line_no) REFERENCES reservation_items(order_id,line_no),
    EXCLUDE USING gist (unit_id WITH =, period WITH &&) WHERE (NOT cancelled)
);
CREATE INDEX allocations_order_idx ON allocations(order_id);
CREATE INDEX allocations_outstanding_idx ON allocations(unit_id) WHERE issued AND NOT returned AND NOT cancelled;
CREATE TABLE seed_history (name text PRIMARY KEY);
