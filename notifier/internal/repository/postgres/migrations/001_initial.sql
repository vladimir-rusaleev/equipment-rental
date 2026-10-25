CREATE TABLE notifications (
    id text PRIMARY KEY,
    user_id text NOT NULL,
    order_id text NOT NULL,
    state text NOT NULL,
    sent boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    sent_at timestamptz
);
CREATE INDEX notifications_pending_idx ON notifications(created_at,id) WHERE NOT sent;
