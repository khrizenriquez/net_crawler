CREATE TABLE IF NOT EXISTS authorized_radios (
  id text PRIMARY KEY,
  ssid text NOT NULL,
  bssid macaddr NOT NULL UNIQUE,
  band text NOT NULL,
  channel integer NOT NULL CHECK (channel > 0),
  confirmed boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS capture_schedules (
  id text PRIMARY KEY,
  name text NOT NULL,
  days smallint[] NOT NULL,
  start_minute integer NOT NULL CHECK (start_minute BETWEEN 0 AND 1439),
  duration_minutes integer NOT NULL CHECK (duration_minutes BETWEEN 1 AND 120),
  channels integer[] NOT NULL,
  rotation_minutes integer NOT NULL DEFAULT 5 CHECK (rotation_minutes > 0),
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS capture_sessions (
  id text PRIMARY KEY,
  origin text NOT NULL,
  status text NOT NULL,
  channel integer,
  band text,
  coverage text NOT NULL,
  started_at timestamptz NOT NULL,
  ended_at timestamptz,
  bytes_observed bigint NOT NULL DEFAULT 0,
  packets_observed bigint NOT NULL DEFAULT 0,
  error text
);

CREATE TABLE IF NOT EXISTS metric_buckets (
  id text PRIMARY KEY,
  captured_at timestamptz NOT NULL,
  source text NOT NULL,
  device_mac macaddr,
  remote_ip inet,
  domain text,
  protocol text NOT NULL,
  bytes_up bigint NOT NULL DEFAULT 0,
  bytes_down bigint NOT NULL DEFAULT 0,
  packets_up bigint NOT NULL DEFAULT 0,
  packets_down bigint NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS metric_buckets_captured_at_idx ON metric_buckets(captured_at);

CREATE TABLE IF NOT EXISTS findings (
  id text PRIMARY KEY,
  category text NOT NULL,
  severity text NOT NULL,
  protocol text NOT NULL,
  session_id text REFERENCES capture_sessions(id) ON DELETE CASCADE,
  device_mac macaddr,
  remote_ip inet,
  redacted_sample text NOT NULL,
  content_hash text,
  created_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS findings_created_at_idx ON findings(created_at);

CREATE TABLE IF NOT EXISTS router_samples (
  id text PRIMARY KEY,
  status text NOT NULL,
  captured_at timestamptz NOT NULL,
  bytes_up bigint NOT NULL DEFAULT 0,
  bytes_down bigint NOT NULL DEFAULT 0,
  delta_up bigint NOT NULL DEFAULT 0,
  delta_down bigint NOT NULL DEFAULT 0,
  message text
);

CREATE TABLE IF NOT EXISTS host_commands (
  id text PRIMARY KEY,
  action text NOT NULL CHECK (action IN ('start', 'stop', 'probe', 'restore')),
  args jsonb NOT NULL DEFAULT '{}'::jsonb,
  status text NOT NULL,
  created_at timestamptz NOT NULL,
  result text
);

CREATE TABLE IF NOT EXISTS app_state (
  id text PRIMARY KEY,
  payload jsonb NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
