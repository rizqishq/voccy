DROP INDEX IF EXISTS idx_boards_org_id;
DROP INDEX IF EXISTS idx_feedbacks_board_id;
ALTER TABLE feedbacks DROP COLUMN IF EXISTS vote_count;
DROP TABLE IF EXISTS votes;
