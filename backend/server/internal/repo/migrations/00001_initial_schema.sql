-- +goose Up
CREATE TABLE organizations (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    document JSONB NOT NULL
);

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    role TEXT NOT NULL,
    document JSONB NOT NULL
);
CREATE UNIQUE INDEX users_email_lower_idx ON users (lower(email));

CREATE TABLE organization_members (
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    PRIMARY KEY (organization_id, user_id)
);

CREATE TABLE projects (
    organization_id TEXT NOT NULL,
    id TEXT NOT NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    document JSONB NOT NULL,
    PRIMARY KEY (organization_id, id),
    UNIQUE (organization_id, slug)
);

CREATE TABLE project_members (
    organization_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    PRIMARY KEY (organization_id, project_id, user_id)
);

CREATE TABLE rulesets (
    organization_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    version TEXT NOT NULL DEFAULT '',
    published_version INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    document JSONB NOT NULL,
    PRIMARY KEY (organization_id, project_id, id)
);

CREATE TABLE ruleset_versions (
    organization_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    ruleset_id TEXT NOT NULL,
    version_number INTEGER NOT NULL,
    snapshot JSONB NOT NULL,
    content_hash TEXT NOT NULL,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    change_note TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (organization_id, project_id, ruleset_id, version_number)
);

CREATE TABLE project_credentials (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    prefix TEXT NOT NULL DEFAULT '',
    secret_hash TEXT NOT NULL DEFAULT '',
    document JSONB NOT NULL
);

CREATE TABLE alert_channels (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    document JSONB NOT NULL
);

CREATE TABLE notifications (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL DEFAULT 'default',
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL,
    document JSONB NOT NULL
);

CREATE TABLE invocation_logs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL DEFAULT '',
    project_id TEXT NOT NULL DEFAULT '',
    ruleset_id TEXT NOT NULL DEFAULT '',
    ruleset_version INTEGER NOT NULL DEFAULT 0,
    request_id TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    status_code INTEGER NOT NULL DEFAULT 0,
    matched BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL,
    document JSONB NOT NULL
);

CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    project_id TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    document JSONB NOT NULL
);

CREATE TABLE platform_settings (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    name TEXT NOT NULL,
    tagline TEXT NOT NULL,
    logo_url TEXT NOT NULL DEFAULT '',
    document JSONB NOT NULL
);

CREATE TABLE license_state (id SMALLINT PRIMARY KEY CHECK (id = 1), document JSONB NOT NULL);
CREATE TABLE jwt_signing_keys (id SMALLINT PRIMARY KEY CHECK (id = 1), document JSONB NOT NULL);

CREATE INDEX invocation_logs_org_created_idx ON invocation_logs (organization_id, created_at DESC);
CREATE INDEX invocation_logs_project_created_idx ON invocation_logs (organization_id, project_id, created_at DESC);
CREATE INDEX ruleset_versions_created_idx ON ruleset_versions (organization_id, project_id, ruleset_id, created_at DESC);
CREATE INDEX invocation_logs_filter_idx ON invocation_logs (organization_id, project_id, ruleset_id, created_at DESC);
CREATE INDEX invocation_logs_request_id_idx ON invocation_logs (organization_id, project_id, request_id) WHERE request_id <> '';
CREATE INDEX invocation_logs_source_idx ON invocation_logs (organization_id, project_id, source, created_at DESC);
CREATE INDEX invocation_logs_created_idx ON invocation_logs (created_at);
CREATE INDEX notifications_organization_created_idx ON notifications (organization_id, created_at DESC);

-- +goose Down
DROP TABLE jwt_signing_keys;
DROP TABLE license_state;
DROP TABLE platform_settings;
DROP TABLE sessions;
DROP TABLE invocation_logs;
DROP TABLE notifications;
DROP TABLE alert_channels;
DROP TABLE project_credentials;
DROP TABLE ruleset_versions;
DROP TABLE rulesets;
DROP TABLE project_members;
DROP TABLE projects;
DROP TABLE organization_members;
DROP TABLE users;
DROP TABLE organizations;
