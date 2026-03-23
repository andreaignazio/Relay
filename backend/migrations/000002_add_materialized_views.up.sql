-- -------------------------------------------------------
-- Materialized Views (read models / CQRS query side)
-- -------------------------------------------------------

CREATE TABLE workspace_views (
    workspace_id UUID        NOT NULL,
    user_id      UUID        NOT NULL,
    name         TEXT        NOT NULL,
    slug         TEXT        NOT NULL,
    icon_url     TEXT,
    role         TEXT        NOT NULL,
    joined_at    TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, workspace_id)
);

CREATE INDEX idx_workspace_views_user_id ON workspace_views (user_id);
