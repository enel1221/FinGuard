CREATE TABLE IF NOT EXISTS projects (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cost_sources (
    id                TEXT PRIMARY KEY,
    project_id        TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    type              TEXT NOT NULL,
    name              TEXT NOT NULL,
    config_json       TEXT NOT NULL DEFAULT '{}',
    enabled           BOOLEAN NOT NULL DEFAULT TRUE,
    last_collected_at TIMESTAMP,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cost_sources_project ON cost_sources(project_id);

CREATE TABLE IF NOT EXISTS users (
    id           TEXT PRIMARY KEY,
    email        TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    oidc_subject TEXT UNIQUE,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS groups (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    oidc_claim TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id)
);

CREATE TABLE IF NOT EXISTS project_roles (
    project_id   TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    subject_type TEXT NOT NULL,
    subject_id   TEXT NOT NULL,
    role         TEXT NOT NULL,
    PRIMARY KEY (project_id, subject_type, subject_id)
);

CREATE INDEX IF NOT EXISTS idx_project_roles_subject ON project_roles(subject_type, subject_id);

CREATE TABLE IF NOT EXISTS budgets (
    id             TEXT PRIMARY KEY,
    project_id     TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    cost_source_id TEXT REFERENCES cost_sources(id) ON DELETE SET NULL,
    monthly_limit  REAL NOT NULL DEFAULT 0,
    warn_threshold REAL NOT NULL DEFAULT 0.8,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_budgets_project ON budgets(project_id);

CREATE TABLE IF NOT EXISTS cost_records (
    id                 TEXT PRIMARY KEY,
    project_id         TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    cost_source_id     TEXT NOT NULL REFERENCES cost_sources(id) ON DELETE CASCADE,
    provider           TEXT NOT NULL,
    provider_id        TEXT NOT NULL DEFAULT '',
    account_id         TEXT NOT NULL DEFAULT '',
    account_name       TEXT NOT NULL DEFAULT '',
    invoice_entity_id  TEXT NOT NULL DEFAULT '',
    service            TEXT NOT NULL,
    category           TEXT NOT NULL DEFAULT '',
    region             TEXT NOT NULL DEFAULT '',
    availability_zone  TEXT NOT NULL DEFAULT '',
    start_time         TIMESTAMP NOT NULL,
    end_time           TIMESTAMP NOT NULL,
    list_cost          REAL NOT NULL DEFAULT 0,
    net_cost           REAL NOT NULL DEFAULT 0,
    amortized_cost     REAL NOT NULL DEFAULT 0,
    amortized_net_cost REAL NOT NULL DEFAULT 0,
    currency           TEXT NOT NULL DEFAULT 'USD',
    labels_json        TEXT NOT NULL DEFAULT '{}',
    kubernetes_percent REAL NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_cost_records_project ON cost_records(project_id);
CREATE INDEX IF NOT EXISTS idx_cost_records_source ON cost_records(cost_source_id);
CREATE INDEX IF NOT EXISTS idx_cost_records_time ON cost_records(start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_cost_records_provider ON cost_records(provider);
