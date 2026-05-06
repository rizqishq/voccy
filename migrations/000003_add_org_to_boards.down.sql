ALTER TABLE boards DROP CONSTRAINT IF EXISTS boards_org_slug_unique;
ALTER TABLE boards ADD CONSTRAINT boards_slug_key UNIQUE(slug);
ALTER TABLE boards DROP COLUMN org_id;
