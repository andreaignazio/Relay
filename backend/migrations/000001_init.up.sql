-- -------------------------------------------------------
-- Event Store (append-only log)
-- -------------------------------------------------------
CREATE TABLE event_stores (
    event_id       UUID        PRIMARY KEY,
    aggregate_type TEXT        NOT NULL,
    aggregate_id   UUID        NOT NULL,
    version        INTEGER     NOT NULL,
    action_key     TEXT        NOT NULL,
    payload        JSONB,
    metadata       JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_aggregate_version UNIQUE (aggregate_id, version)
);

CREATE INDEX idx_event_stores_aggregate_id ON event_stores (aggregate_id);

-- -------------------------------------------------------
-- Snapshots (read models / CQRS view)
-- -------------------------------------------------------

CREATE TABLE user_snapshots (
    id              UUID        PRIMARY KEY,
    email           TEXT        NOT NULL UNIQUE,
    username        TEXT        NOT NULL UNIQUE,
    display_name    TEXT        NOT NULL,
    avatar_url      TEXT,
    password_hash   TEXT        NOT NULL,
    auth_provider   TEXT        NOT NULL,
    snap_version    INTEGER     NOT NULL,
    snap_updated_at TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    deleted_at      TIMESTAMPTZ
);

CREATE TABLE workspace_snapshots (
    id              UUID        PRIMARY KEY,
    name            TEXT        NOT NULL,
    slug            TEXT        NOT NULL UNIQUE,
    icon_url        TEXT,
    snap_version    INTEGER     NOT NULL,
    snap_updated_at TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    deleted_at      TIMESTAMPTZ
);

CREATE TABLE workspace_membership_snapshots (
    id                UUID        PRIMARY KEY,
    user_id           UUID        NOT NULL,
    workspace_id      UUID        NOT NULL,
    role              TEXT        NOT NULL,
    status_emoji      TEXT,
    status_text       TEXT,
    status_expires_at TIMESTAMPTZ,
    joined_at         TIMESTAMPTZ NOT NULL,
    snap_version      INTEGER     NOT NULL,
    snap_updated_at   TIMESTAMPTZ NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL,
    deleted_at        TIMESTAMPTZ,
    CONSTRAINT uq_workspace_membership UNIQUE (user_id, workspace_id)
);

CREATE TABLE channel_snapshots (
    id              UUID        PRIMARY KEY,
    workspace_id    UUID        NOT NULL,
    created_by      UUID        NOT NULL,
    name            TEXT,
    description     TEXT,
    topic           TEXT,
    type            TEXT        NOT NULL,
    is_archived     BOOLEAN     NOT NULL DEFAULT FALSE,
    snap_version    INTEGER     NOT NULL,
    snap_updated_at TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_channel_snapshots_workspace_id ON channel_snapshots (workspace_id);

CREATE TABLE channel_membership_snapshots (
    id                 UUID        PRIMARY KEY,
    user_id            UUID        NOT NULL,
    channel_id         UUID        NOT NULL,
    role               TEXT        NOT NULL,
    notifications_pref TEXT        NOT NULL DEFAULT 'all',
    last_read_at       TIMESTAMPTZ,
    joined_at          TIMESTAMPTZ NOT NULL,
    snap_version       INTEGER     NOT NULL,
    snap_updated_at    TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL,
    updated_at         TIMESTAMPTZ NOT NULL,
    deleted_at         TIMESTAMPTZ,
    CONSTRAINT uq_channel_membership UNIQUE (user_id, channel_id)
);

CREATE TABLE message_snapshots (
    id                UUID        PRIMARY KEY,
    channel_id        UUID        NOT NULL,
    user_id           UUID        NOT NULL,
    parent_message_id UUID,
    content           TEXT        NOT NULL,
    is_edited         BOOLEAN     NOT NULL DEFAULT FALSE,
    reply_count       INTEGER     NOT NULL DEFAULT 0,
    last_reply_at     TIMESTAMPTZ,
    snap_version      INTEGER     NOT NULL,
    snap_updated_at   TIMESTAMPTZ NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL,
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX idx_message_snapshots_channel_id        ON message_snapshots (channel_id);
CREATE INDEX idx_message_snapshots_parent_message_id ON message_snapshots (parent_message_id);

CREATE TABLE reaction_snapshots (
    id              UUID        PRIMARY KEY,
    message_id      UUID        NOT NULL,
    user_id         UUID        NOT NULL,
    emoji           TEXT        NOT NULL,
    snap_version    INTEGER     NOT NULL,
    snap_updated_at TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT uq_reaction UNIQUE (message_id, user_id, emoji)
);

-- Attachments are immutable after upload (no updated_at / deleted_at)
CREATE TABLE attachment_snapshots (
    id              UUID    PRIMARY KEY,
    message_id      UUID    NOT NULL,
    user_id         UUID    NOT NULL,
    filename        TEXT    NOT NULL,
    file_url        TEXT    NOT NULL,
    mime_type       TEXT    NOT NULL,
    size_bytes      BIGINT  NOT NULL,
    version         INTEGER NOT NULL,
    snap_version    INTEGER     NOT NULL,
    snap_updated_at TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL
);

CREATE TABLE notification_snapshots (
    id              UUID        PRIMARY KEY,
    user_id         UUID        NOT NULL,
    workspace_id    UUID        NOT NULL,
    type            TEXT        NOT NULL,
    reference_id    UUID        NOT NULL,
    reference_type  TEXT        NOT NULL,
    is_read         BOOLEAN     NOT NULL DEFAULT FALSE,
    snap_version    INTEGER     NOT NULL,
    snap_updated_at TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_notification_snapshots_user_id ON notification_snapshots (user_id);

CREATE TABLE invite_snapshots (
    id              UUID        PRIMARY KEY,
    workspace_id    UUID        NOT NULL,
    invited_by      UUID        NOT NULL,
    token           TEXT        NOT NULL UNIQUE,
    email           TEXT,
    expires_at      TIMESTAMPTZ NOT NULL,
    max_uses        INTEGER     NOT NULL DEFAULT 0,
    use_count       INTEGER     NOT NULL DEFAULT 0,
    snap_version    INTEGER     NOT NULL,
    snap_updated_at TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    deleted_at      TIMESTAMPTZ
);
