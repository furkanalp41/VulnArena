-- Community Forge: user-generated challenge submissions
CREATE TABLE community_challenges (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title                TEXT NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    difficulty           INT NOT NULL CHECK (difficulty BETWEEN 1 AND 10),
    language_slug        TEXT NOT NULL,
    vuln_category_slug   TEXT NOT NULL,
    vulnerable_code      TEXT NOT NULL,
    target_vulnerability TEXT NOT NULL,
    conceptual_fix       TEXT NOT NULL DEFAULT '',
    vulnerable_lines     TEXT NOT NULL DEFAULT '',
    hints                JSONB NOT NULL DEFAULT '[]',
    points               INT NOT NULL DEFAULT 100,
    status               TEXT NOT NULL DEFAULT 'pending'
                         CHECK (status IN ('pending', 'approved', 'rejected', 'published')),
    reviewer_id          UUID REFERENCES users(id),
    reviewer_notes       TEXT NOT NULL DEFAULT '',
    challenge_id         UUID REFERENCES challenges(id),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_community_challenges_status ON community_challenges(status);
CREATE INDEX idx_community_challenges_author ON community_challenges(author_id);
