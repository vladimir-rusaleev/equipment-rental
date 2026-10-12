CREATE TABLE users (
    id text PRIMARY KEY,
    email text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    role text NOT NULL CHECK (role IN ('customer','staff','admin'))
);
CREATE TABLE sessions (
    token_hash text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id),
    expires_at timestamptz NOT NULL
);
CREATE INDEX sessions_expiration_idx ON sessions(expires_at);
CREATE TABLE carts (
    user_id text PRIMARY KEY REFERENCES users(id),
    version bigint NOT NULL CHECK (version >= 0),
    items jsonb NOT NULL DEFAULT '[]'
);
CREATE TABLE orders (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id),
    request_id text NOT NULL,
    state text NOT NULL CHECK (state IN ('pending','confirmed','active','completed','cancelled','rejected','cancelling','issuing','returning')),
    version bigint NOT NULL DEFAULT 0,
    payload jsonb NOT NULL,
    UNIQUE(user_id,request_id)
);
CREATE INDEX orders_user_idx ON orders(user_id);
CREATE INDEX orders_pending_idx ON orders(state) WHERE state IN ('pending','cancelling','issuing','returning');
CREATE TABLE outbox (
    id text PRIMARY KEY,
    user_id text NOT NULL REFERENCES users(id),
    order_id text NOT NULL REFERENCES orders(id),
    state text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
