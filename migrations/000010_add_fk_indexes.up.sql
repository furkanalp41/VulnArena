-- Add indexes for foreign-key / hot-lookup columns that were previously unindexed.
-- These FKs are not covered by a PK leading column, UNIQUE constraint, or an
-- earlier CREATE INDEX, so joins / cascading deletes / filters on them require
-- a sequential scan without these.

-- challenges.first_blood_user_id -> users(id) (000004)
CREATE INDEX IF NOT EXISTS idx_challenges_first_blood_user_id ON challenges (first_blood_user_id);

-- user_challenge_progress PK is (user_id, challenge_id); challenge_id is not a leading
-- column, so per-challenge lookups / FK cascade on challenges are unindexed.
CREATE INDEX IF NOT EXISTS idx_user_challenge_progress_challenge_id ON user_challenge_progress (challenge_id);

-- user_lesson_progress PK is (user_id, lesson_id); lesson_id is not a leading column.
CREATE INDEX IF NOT EXISTS idx_user_lesson_progress_lesson_id ON user_lesson_progress (lesson_id);

-- user_achievements PK is (user_id, achievement_id); achievement_id is not a leading column.
CREATE INDEX IF NOT EXISTS idx_user_achievements_achievement_id ON user_achievements (achievement_id);

-- teams.created_by -> users(id)
CREATE INDEX IF NOT EXISTS idx_teams_created_by ON teams (created_by);

-- community_challenges.reviewer_id -> users(id)
CREATE INDEX IF NOT EXISTS idx_community_challenges_reviewer_id ON community_challenges (reviewer_id);

-- community_challenges.challenge_id -> challenges(id)
CREATE INDEX IF NOT EXISTS idx_community_challenges_challenge_id ON community_challenges (challenge_id);
