-- +goose Up
ALTER TABLE notification_preference_settings
  ADD COLUMN interests_json jsonb NOT NULL DEFAULT '{}',
  ADD COLUMN routine_digest boolean NOT NULL DEFAULT false,
  ADD COLUMN digest_time text NOT NULL DEFAULT '18:00';
CREATE TABLE notification_digests (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  firebase_device_id uuid NOT NULL REFERENCES firebase_devices(id) ON DELETE CASCADE,
  window_at timestamptz NOT NULL,
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','retry','accepted','failed')),
  next_attempt_at timestamptz NOT NULL,
  locked_at timestamptz,
  locked_by text,
  attempt_count integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  UNIQUE (firebase_device_id, window_at)
);
ALTER TABLE notification_outbox_jobs ADD COLUMN digest_id uuid REFERENCES notification_digests(id) ON DELETE SET NULL;
CREATE INDEX idx_notification_digest_claim ON notification_digests(status,next_attempt_at);
CREATE INDEX idx_notification_outbox_digest ON notification_outbox_jobs(digest_id) WHERE digest_id IS NOT NULL;
CREATE INDEX idx_notification_provider_message ON notification_deliveries(provider_message_id) WHERE channel = 'push' AND deleted_at IS NULL;
ALTER TABLE notification_delivery_events DROP CONSTRAINT chk_notification_delivery_event_type;
ALTER TABLE notification_delivery_events ADD CONSTRAINT chk_notification_delivery_event_type CHECK (event_type IN ('opened','clicked','delivered'));
-- +goose Down
DELETE FROM notification_delivery_events WHERE event_type = 'delivered';
ALTER TABLE notification_delivery_events DROP CONSTRAINT chk_notification_delivery_event_type;
ALTER TABLE notification_delivery_events ADD CONSTRAINT chk_notification_delivery_event_type CHECK (event_type IN ('opened','clicked'));
ALTER TABLE notification_outbox_jobs DROP COLUMN digest_id;
DROP TABLE notification_digests;
DROP INDEX idx_notification_provider_message;
ALTER TABLE notification_preference_settings DROP COLUMN interests_json, DROP COLUMN routine_digest, DROP COLUMN digest_time;
