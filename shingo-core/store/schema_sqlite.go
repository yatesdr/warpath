package store

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS nodes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    vendor_location TEXT NOT NULL DEFAULT '',
    node_type   TEXT NOT NULL DEFAULT 'storage',
    zone        TEXT NOT NULL DEFAULT '',
    capacity    INTEGER NOT NULL DEFAULT 0,
    enabled     INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);

CREATE TABLE IF NOT EXISTS materials (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    code        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    unit        TEXT NOT NULL DEFAULT 'ea',
    created_at  TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);

CREATE TABLE IF NOT EXISTS orders (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    edge_uuid    TEXT NOT NULL,
    station_id      TEXT NOT NULL DEFAULT '',
    factory_id      TEXT NOT NULL DEFAULT '',
    order_type      TEXT NOT NULL DEFAULT 'retrieve',
    status          TEXT NOT NULL DEFAULT 'pending',
    material_id     INTEGER REFERENCES materials(id),
    material_code   TEXT NOT NULL DEFAULT '',
    quantity        REAL NOT NULL DEFAULT 1,
    source_node_id  INTEGER REFERENCES nodes(id),
    dest_node_id    INTEGER REFERENCES nodes(id),
    pickup_node     TEXT NOT NULL DEFAULT '',
    delivery_node   TEXT NOT NULL DEFAULT '',
    vendor_order_id TEXT NOT NULL DEFAULT '',
    vendor_state    TEXT NOT NULL DEFAULT '',
    robot_id        TEXT NOT NULL DEFAULT '',
    priority        INTEGER NOT NULL DEFAULT 0,
    payload_desc    TEXT NOT NULL DEFAULT '',
    error_detail    TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    completed_at    TEXT,
    payload_type_id INTEGER REFERENCES payload_types(id),
    payload_id      INTEGER REFERENCES payloads(id)
);
CREATE INDEX IF NOT EXISTS idx_orders_uuid ON orders(edge_uuid);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_vendor ON orders(vendor_order_id);

CREATE TABLE IF NOT EXISTS node_inventory (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id         INTEGER NOT NULL REFERENCES nodes(id),
    material_id     INTEGER NOT NULL REFERENCES materials(id),
    quantity        REAL NOT NULL DEFAULT 0,
    is_partial      INTEGER NOT NULL DEFAULT 0,
    delivered_at    TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    source_order_id INTEGER REFERENCES orders(id),
    metadata        TEXT NOT NULL DEFAULT '{}',
    notes           TEXT NOT NULL DEFAULT '',
    claimed_by      INTEGER REFERENCES orders(id),
    created_at      TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_inventory_node ON node_inventory(node_id);
CREATE INDEX IF NOT EXISTS idx_inventory_material ON node_inventory(material_id);
CREATE INDEX IF NOT EXISTS idx_inventory_fifo ON node_inventory(material_id, is_partial DESC, delivered_at ASC);

CREATE TABLE IF NOT EXISTS order_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id    INTEGER NOT NULL REFERENCES orders(id),
    status      TEXT NOT NULL,
    detail      TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_order_history_order ON order_history(order_id);

CREATE TABLE IF NOT EXISTS outbox (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    topic       TEXT NOT NULL,
    payload     BLOB NOT NULL,
    msg_type    TEXT NOT NULL DEFAULT '',
    station_id  TEXT NOT NULL DEFAULT '',
    retries     INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    sent_at     TEXT
);
CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox(sent_at) WHERE sent_at IS NULL;

CREATE TABLE IF NOT EXISTS audit_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id   INTEGER NOT NULL DEFAULT 0,
    action      TEXT NOT NULL,
    old_value   TEXT NOT NULL DEFAULT '',
    new_value   TEXT NOT NULL DEFAULT '',
    actor       TEXT NOT NULL DEFAULT 'system',
    created_at  TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_log(entity_type, entity_id);

CREATE TABLE IF NOT EXISTS corrections (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    correction_type  TEXT NOT NULL,
    node_id          INTEGER NOT NULL REFERENCES nodes(id),
    material_id      INTEGER REFERENCES materials(id),
    inventory_id     INTEGER REFERENCES node_inventory(id),
    payload_id       INTEGER REFERENCES payloads(id),
    manifest_item_id INTEGER REFERENCES manifest_items(id),
    cat_id           TEXT NOT NULL DEFAULT '',
    description      TEXT NOT NULL DEFAULT '',
    quantity         REAL NOT NULL DEFAULT 0,
    reason           TEXT NOT NULL,
    actor            TEXT NOT NULL DEFAULT 'system',
    created_at       TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);

CREATE TABLE IF NOT EXISTS admin_users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);

CREATE TABLE IF NOT EXISTS payload_types (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    name                  TEXT NOT NULL UNIQUE,
    description           TEXT NOT NULL DEFAULT '',
    form_factor           TEXT NOT NULL DEFAULT 'other',
    default_manifest_json TEXT NOT NULL DEFAULT '{}',
    created_at            TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at            TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);

CREATE TABLE IF NOT EXISTS payloads (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    payload_type_id INTEGER NOT NULL REFERENCES payload_types(id),
    node_id         INTEGER REFERENCES nodes(id),
    status          TEXT NOT NULL DEFAULT 'empty',
    claimed_by      INTEGER REFERENCES orders(id),
    delivered_at    TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_payloads_type ON payloads(payload_type_id);
CREATE INDEX IF NOT EXISTS idx_payloads_node ON payloads(node_id);
CREATE INDEX IF NOT EXISTS idx_payloads_status ON payloads(status);

CREATE TABLE IF NOT EXISTS manifest_items (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    payload_id      INTEGER NOT NULL REFERENCES payloads(id) ON DELETE CASCADE,
    part_number     TEXT NOT NULL DEFAULT '',
    quantity        REAL NOT NULL DEFAULT 0,
    production_date TEXT,
    lot_code        TEXT,
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_manifest_payload ON manifest_items(payload_id);

CREATE TABLE IF NOT EXISTS scene_points (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    area_name       TEXT NOT NULL,
    instance_name   TEXT NOT NULL,
    class_name      TEXT NOT NULL,
    point_name      TEXT NOT NULL DEFAULT '',
    group_name      TEXT NOT NULL DEFAULT '',
    label           TEXT NOT NULL DEFAULT '',
    pos_x           REAL NOT NULL DEFAULT 0,
    pos_y           REAL NOT NULL DEFAULT 0,
    pos_z           REAL NOT NULL DEFAULT 0,
    dir             REAL NOT NULL DEFAULT 0,
    properties_json TEXT NOT NULL DEFAULT '{}',
    synced_at       TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    UNIQUE(area_name, instance_name)
);
CREATE INDEX IF NOT EXISTS idx_scene_points_class ON scene_points(class_name);
CREATE INDEX IF NOT EXISTS idx_scene_points_area ON scene_points(area_name);

CREATE TABLE IF NOT EXISTS edge_registry (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    station_id      TEXT NOT NULL UNIQUE,
    factory_id      TEXT NOT NULL,
    hostname        TEXT NOT NULL DEFAULT '',
    version         TEXT NOT NULL DEFAULT '',
    line_ids        TEXT NOT NULL DEFAULT '[]',
    registered_at   TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    last_heartbeat  TEXT,
    status          TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS demands (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    cat_id       TEXT NOT NULL UNIQUE,
    description  TEXT NOT NULL DEFAULT '',
    demand_qty   REAL NOT NULL DEFAULT 0,
    produced_qty REAL NOT NULL DEFAULT 0,
    created_at   TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);

CREATE TABLE IF NOT EXISTS production_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    cat_id      TEXT NOT NULL,
    station_id  TEXT NOT NULL,
    quantity    REAL NOT NULL,
    reported_at TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_production_log_cat ON production_log(cat_id);

CREATE TABLE IF NOT EXISTS test_commands (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    command_type    TEXT NOT NULL,
    robot_id        TEXT NOT NULL,
    vendor_order_id TEXT NOT NULL DEFAULT '',
    vendor_state    TEXT NOT NULL DEFAULT '',
    location        TEXT NOT NULL DEFAULT '',
    config_id       TEXT NOT NULL DEFAULT '',
    detail          TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    completed_at    TEXT
);
`
