CREATE TABLE channel_membership_views (
    user_id            UUID        NOT NULL,
    channel_id         UUID        NOT NULL,
    workspace_id       UUID        NOT NULL,
    name               TEXT,
    description        TEXT,
    topic              TEXT,
    type               TEXT        NOT NULL,
    is_archived        BOOLEAN     NOT NULL DEFAULT FALSE,
    created_by         UUID        NOT NULL,
    role               TEXT        NOT NULL,
    joined_at          TIMESTAMPTZ NOT NULL,
    notifications_pref TEXT        NOT NULL DEFAULT 'all',
    last_read_at       TIMESTAMPTZ,
    PRIMARY KEY (user_id, channel_id)
);

CREATE INDEX idx_channel_membership_views_user_workspace ON channel_membership_views (user_id, workspace_id);

CREATE TABLE channel_views (
    channel_id   UUID        NOT NULL,
    workspace_id UUID        NOT NULL,
    created_by   UUID        NOT NULL,
    name         TEXT,
    description  TEXT,
    topic        TEXT,
    type         TEXT        NOT NULL,
    is_archived  BOOLEAN     NOT NULL DEFAULT FALSE,
    PRIMARY KEY (channel_id)
);

CREATE INDEX idx_channel_views_workspace_id ON channel_views (workspace_id);

CREATE TABLE direct_message_membership_views (
    user_id            UUID        NOT NULL,
    channel_id         UUID        NOT NULL,
    workspace_id       UUID        NOT NULL,
    joined_at          TIMESTAMPTZ NOT NULL,
    notifications_pref TEXT        NOT NULL DEFAULT 'all',
    PRIMARY KEY (user_id, channel_id)
);

CREATE INDEX idx_dm_membership_views_user_workspace ON direct_message_membership_views (user_id, workspace_id);
