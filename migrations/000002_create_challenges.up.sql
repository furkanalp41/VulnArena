-- Vulnerability categories (OWASP 2025 + extended)
CREATE TABLE vuln_categories (
    id          SERIAL PRIMARY KEY,
    slug        VARCHAR(50) UNIQUE NOT NULL,
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    owasp_ref   VARCHAR(30)
);

INSERT INTO vuln_categories (slug, name, description, owasp_ref) VALUES
    ('injection',           'Injection',                    'SQL, NoSQL, OS, LDAP injection where untrusted data is sent as part of a command or query.', 'A03:2021'),
    ('broken-auth',         'Broken Authentication',        'Flaws in authentication mechanisms allowing identity compromise.',                          'A07:2021'),
    ('xss',                 'Cross-Site Scripting',         'XSS flaws when untrusted data is included in a web page without proper validation.',         'A03:2021'),
    ('insecure-deser',      'Insecure Deserialization',     'Deserialization flaws leading to RCE, replay attacks, or privilege escalation.',             'A08:2021'),
    ('broken-access',       'Broken Access Control',        'Restrictions on what authenticated users can do are not properly enforced.',                 'A01:2021'),
    ('security-misconfig',  'Security Misconfiguration',    'Missing security hardening, open cloud storage, verbose error messages.',                    'A05:2021'),
    ('crypto-failures',     'Cryptographic Failures',       'Failures related to cryptography which often lead to sensitive data exposure.',              'A02:2021'),
    ('ssrf',                'Server-Side Request Forgery',  'Web application fetches a remote resource without validating the user-supplied URL.',        'A10:2021'),
    ('cmd-injection',       'OS Command Injection',         'Application passes unsafe user-supplied data to a system shell.',                           'A03:2021'),
    ('memory-corruption',   'Memory Corruption',            'Buffer overflows, use-after-free, format string attacks in native code.',                   NULL),
    ('race-condition',      'Race Condition',               'TOCTOU and other concurrency bugs leading to privilege escalation or data corruption.',     NULL),
    ('rce',                 'Remote Code Execution',        'Flaws allowing execution of arbitrary code on the target system.',                          NULL);

-- Programming languages
CREATE TABLE languages (
    id      SERIAL PRIMARY KEY,
    slug    VARCHAR(30) UNIQUE NOT NULL,
    name    VARCHAR(50) NOT NULL
);

INSERT INTO languages (slug, name) VALUES
    ('go',         'Go'),
    ('rust',       'Rust'),
    ('nodejs',     'Node.js / JavaScript'),
    ('csharp',     'C#'),
    ('c',          'C'),
    ('cpp',        'C++'),
    ('assembly',   'Assembly'),
    ('perl',       'Perl'),
    ('cobol',      'COBOL'),
    ('flutter',    'Flutter / Dart'),
    ('python',     'Python'),
    ('ruby',       'Ruby'),
    ('java',       'Java');

-- Arena challenges
CREATE TABLE challenges (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title               VARCHAR(200) NOT NULL,
    slug                VARCHAR(200) UNIQUE NOT NULL,
    description         TEXT NOT NULL,
    difficulty          SMALLINT NOT NULL CHECK (difficulty BETWEEN 1 AND 10),
    language_id         INT NOT NULL REFERENCES languages(id),
    vuln_category_id    INT NOT NULL REFERENCES vuln_categories(id),
    vulnerable_code     TEXT NOT NULL,
    target_vulnerability TEXT NOT NULL,
    conceptual_fix      TEXT NOT NULL,
    hints               JSONB DEFAULT '[]'::jsonb,
    points              INT NOT NULL DEFAULT 100,
    line_count          INT NOT NULL DEFAULT 0,
    is_published        BOOLEAN DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_challenges_difficulty ON challenges(difficulty);
CREATE INDEX idx_challenges_language ON challenges(language_id);
CREATE INDEX idx_challenges_vuln_cat ON challenges(vuln_category_id);
CREATE INDEX idx_challenges_published ON challenges(is_published) WHERE is_published = TRUE;

-- Submissions
CREATE TABLE submissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    challenge_id    UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    answer_text     TEXT NOT NULL,
    score           DECIMAL(5,2) NOT NULL DEFAULT 0,
    is_correct      BOOLEAN NOT NULL DEFAULT FALSE,
    feedback        JSONB DEFAULT '{}'::jsonb,
    time_spent_sec  INT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_submissions_user ON submissions(user_id);
CREATE INDEX idx_submissions_challenge ON submissions(challenge_id);
CREATE INDEX idx_submissions_user_challenge ON submissions(user_id, challenge_id);

-- User progress per challenge
CREATE TABLE user_challenge_progress (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    challenge_id    UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    status          VARCHAR(20) NOT NULL DEFAULT 'not_started',
    best_score      DECIMAL(5,2) DEFAULT 0,
    attempt_count   INT DEFAULT 0,
    first_solved_at TIMESTAMPTZ,
    last_attempted  TIMESTAMPTZ,
    PRIMARY KEY (user_id, challenge_id)
);
