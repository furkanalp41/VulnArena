CREATE TABLE lessons (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       VARCHAR(300) NOT NULL,
    slug        VARCHAR(300) UNIQUE NOT NULL,
    category    VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    content     TEXT NOT NULL,
    difficulty  SMALLINT NOT NULL CHECK (difficulty BETWEEN 1 AND 10),
    read_time_min INT NOT NULL DEFAULT 10,
    tags        TEXT[] DEFAULT '{}',
    is_published BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lessons_category ON lessons(category);
CREATE INDEX idx_lessons_difficulty ON lessons(difficulty);
CREATE INDEX idx_lessons_published ON lessons(is_published) WHERE is_published = TRUE;

-- Track which lessons a user has completed
CREATE TABLE user_lesson_progress (
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lesson_id    UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    completed    BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, lesson_id)
);
