ALTER TABLE boards ADD COLUMN org_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE boards DROP CONSTRAINT IF EXISTS boards_slug_key;
ALTER TABLE boards ADD CONSTRAINT boards_org_slug_unique UNIQUE(org_id, slug);
