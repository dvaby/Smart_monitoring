CREATE TABLE routers (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  host TEXT NOT NULL,
  api_port INTEGER NOT NULL DEFAULT 8728,
  use_tls INTEGER NOT NULL DEFAULT 0,
  verify_cert INTEGER NOT NULL DEFAULT 0,
  username TEXT NOT NULL,
  password_enc BLOB NOT NULL,
  syslog_source_ip TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'unknown',
  last_poll_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('admin','technician')),
  is_active INTEGER NOT NULL DEFAULT 1,
  failed_attempts INTEGER NOT NULL DEFAULT 0,
  locked_until INTEGER,
  created_at INTEGER NOT NULL
);

CREATE TABLE sessions (
  token_hash TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE TABLE devices (
  id INTEGER PRIMARY KEY,
  router_id INTEGER NOT NULL REFERENCES routers(id),
  name TEXT NOT NULL,
  ip_address TEXT NOT NULL,
  mac_address TEXT,
  type TEXT NOT NULL CHECK (type IN ('access_point','switch','server','printer','other')),
  location TEXT,
  priority TEXT NOT NULL DEFAULT 'normal' CHECK (priority IN ('low','normal','high')),
  current_state TEXT NOT NULL DEFAULT 'unknown' CHECK (current_state IN ('up','down','unknown')),
  state_since INTEGER,
  is_monitored INTEGER NOT NULL DEFAULT 1,
  UNIQUE (router_id, ip_address)
);

CREATE TABLE hosts (
  id INTEGER PRIMARY KEY,
  router_id INTEGER NOT NULL REFERENCES routers(id),
  dhcp_server TEXT,
  ip_address TEXT NOT NULL,
  hostname TEXT,
  lease_mac TEXT,
  arp_mac TEXT,
  is_randomized_mac INTEGER NOT NULL DEFAULT 0,
  source TEXT NOT NULL CHECK (source IN ('dhcp','static')),
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','indication')),
  interface TEXT,
  lease_status TEXT,
  first_seen INTEGER NOT NULL,
  last_seen INTEGER NOT NULL,
  UNIQUE (router_id, ip_address)
);
CREATE INDEX idx_hosts_mac ON hosts(lease_mac, arp_mac);

CREATE TABLE arp_lease_log (
  id INTEGER PRIMARY KEY,
  router_id INTEGER NOT NULL REFERENCES routers(id),
  ip_address TEXT NOT NULL,
  lease_mac TEXT,
  arp_mac TEXT,
  interface TEXT,
  change_type TEXT NOT NULL,
  observed_at INTEGER NOT NULL
);
CREATE INDEX idx_arplog_ip_time ON arp_lease_log(ip_address, observed_at);

CREATE TABLE syslog_events (
  id INTEGER PRIMARY KEY,
  router_id INTEGER REFERENCES routers(id),
  topics TEXT NOT NULL,
  message TEXT NOT NULL,
  parsed_ip TEXT,
  parsed_mac TEXT,
  received_at INTEGER NOT NULL
);
CREATE INDEX idx_syslog_time ON syslog_events(received_at);

CREATE TABLE incidents (
  id INTEGER PRIMARY KEY,
  type TEXT NOT NULL CHECK (type IN ('IP_CONFLICT','DEVICE_DOWN')),
  router_id INTEGER NOT NULL REFERENCES routers(id),
  device_id INTEGER REFERENCES devices(id),
  dhcp_server TEXT,
  ip_address TEXT NOT NULL,
  severity TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('info','warning','critical')),
  status TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new','acknowledged','in_progress','resolved')),
  source TEXT NOT NULL,
  lease_mac TEXT,
  arp_mac TEXT,
  interface TEXT,
  lease_status TEXT,
  occurrence_count INTEGER NOT NULL DEFAULT 1,
  first_detected_at INTEGER NOT NULL,
  last_detected_at INTEGER NOT NULL,
  acknowledged_by INTEGER REFERENCES users(id),
  acknowledged_at INTEGER,
  resolved_by INTEGER REFERENCES users(id),
  resolved_at INTEGER,
  resolution TEXT,
  resolution_note TEXT,
  downtime_seconds INTEGER,
  checklist_state TEXT
);
CREATE UNIQUE INDEX uq_open_incident
  ON incidents(type, router_id, ip_address) WHERE status <> 'resolved';
CREATE INDEX idx_incidents_status ON incidents(status, last_detected_at);

CREATE TABLE incident_events (
  id INTEGER PRIMARY KEY,
  incident_id INTEGER NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  user_id INTEGER REFERENCES users(id),
  action TEXT NOT NULL,
  detail TEXT,
  created_at INTEGER NOT NULL
);

CREATE TABLE notification_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  dashboard_enabled INTEGER NOT NULL DEFAULT 1,
  telegram_enabled INTEGER NOT NULL DEFAULT 0,
  telegram_bot_token_enc BLOB,
  telegram_chat_id TEXT,
  telegram_proxy_enc BLOB,
  whatsapp_enabled INTEGER NOT NULL DEFAULT 0,
  whatsapp_webhook_url TEXT,
  whatsapp_secret_enc BLOB,
  email_enabled INTEGER NOT NULL DEFAULT 0,
  smtp_config_enc BLOB,
  email_digest_time TEXT DEFAULT '07:30',
  validation_interval_sec INTEGER NOT NULL DEFAULT 60 CHECK (validation_interval_sec BETWEEN 15 AND 600),
  cooldown_minutes INTEGER NOT NULL DEFAULT 10 CHECK (cooldown_minutes BETWEEN 1 AND 120),
  quiet_hours_start TEXT,
  quiet_hours_end TEXT,
  public_base_url TEXT
);

CREATE TABLE notification_log (
  id INTEGER PRIMARY KEY,
  incident_id INTEGER REFERENCES incidents(id),
  channel TEXT NOT NULL,
  template TEXT NOT NULL,
  dedup_key TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('queued','sent','failed','suppressed')),
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,
  created_at INTEGER NOT NULL,
  sent_at INTEGER
);
CREATE INDEX idx_notif_dedup ON notification_log(dedup_key, created_at);

CREATE TABLE app_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
