ALTER TABLE message_snapshots
    DROP COLUMN IF EXISTS mentioned_user_ids,
    DROP COLUMN IF EXISTS mention_channel,
    DROP COLUMN IF EXISTS mention_here;
