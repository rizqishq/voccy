CREATE TABLE IF NOT EXISTS votes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feedback_id UUID NOT NULL REFERENCES feedbacks(id) ON DELETE CASCADE,
    fingerprint VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(feedback_id, fingerprint)
);

ALTER TABLE feedbacks ADD COLUMN vote_count INT DEFAULT 0;

CREATE INDEX idx_feedbacks_board_id ON feedbacks(board_id);
CREATE INDEX idx_boards_org_id ON boards(org_id);
