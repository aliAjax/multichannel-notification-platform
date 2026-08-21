CREATE TABLE IF NOT EXISTS notifications (
  id text PRIMARY KEY, tenant_id text NOT NULL, idempotency_key text NOT NULL,
  channel text NOT NULL, status text NOT NULL, priority integer NOT NULL,
  payload jsonb NOT NULL, attempts integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
  version bigint NOT NULL DEFAULT 1, deleted_at timestamptz,
  created_by text, updated_by text, UNIQUE (tenant_id, idempotency_key)
);
CREATE TABLE IF NOT EXISTS notification_attempts (
  id text PRIMARY KEY, notification_id text NOT NULL REFERENCES notifications(id),
  provider text NOT NULL, status text NOT NULL, error_class text,
  external_id text, started_at timestamptz NOT NULL, finished_at timestamptz
);
CREATE TABLE IF NOT EXISTS notification_events (
  id text PRIMARY KEY, notification_id text NOT NULL REFERENCES notifications(id),
  sequence bigint NOT NULL, status text NOT NULL, provider text,
  occurred_at timestamptz NOT NULL, metadata jsonb,
  UNIQUE(notification_id, sequence)
);
CREATE TABLE IF NOT EXISTS outbox (
  id text PRIMARY KEY, topic text NOT NULL, aggregate_id text NOT NULL,
  payload jsonb NOT NULL, created_at timestamptz NOT NULL, published_at timestamptz
);
CREATE INDEX IF NOT EXISTS notifications_tenant_created ON notifications(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS outbox_unpublished ON outbox(created_at) WHERE published_at IS NULL;

