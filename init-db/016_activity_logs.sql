CREATE TABLE IF NOT EXISTS activity_logs (
    id          VARCHAR(30)  PRIMARY KEY,
    store_id    VARCHAR(30)  NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    user_id     VARCHAR(30)  NOT NULL DEFAULT '',
    user_name   VARCHAR(255) NOT NULL DEFAULT '',
    action      VARCHAR(100) NOT NULL DEFAULT '',
    module      VARCHAR(100) NOT NULL DEFAULT '',
    resource_id VARCHAR(30)  NOT NULL DEFAULT '',
    method      VARCHAR(10)  NOT NULL DEFAULT '',
    path        TEXT         NOT NULL DEFAULT '',
    ip_address  VARCHAR(50)  NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_activity_logs_store_id   ON activity_logs(store_id);
CREATE INDEX IF NOT EXISTS idx_activity_logs_created_at ON activity_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_activity_logs_module     ON activity_logs(store_id, module);
