ALTER TABLE message_snapshots
    ADD COLUMN mentioned_user_ids UUID[]     NOT NULL DEFAULT '{}',
    ADD COLUMN mention_channel    BOOLEAN    NOT NULL DEFAULT FALSE,
    ADD COLUMN mention_here       BOOLEAN    NOT NULL DEFAULT FALSE;
