-- First Blood mechanics: track who first pwned each challenge
ALTER TABLE challenges ADD COLUMN first_blood_user_id UUID REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE submissions ADD COLUMN is_first_blood BOOLEAN NOT NULL DEFAULT FALSE;

-- Index for quick first blood lookups
CREATE INDEX idx_submissions_first_blood ON submissions (challenge_id) WHERE is_first_blood = TRUE;
