CREATE TABLE IF NOT EXISTS deleted_items (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deleted_items_user_id ON deleted_items(user_id);
CREATE INDEX IF NOT EXISTS idx_deleted_items_deleted_at ON deleted_items(deleted_at);
