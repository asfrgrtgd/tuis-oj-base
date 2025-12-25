-- Base schema: users, problems, testcases, submissions, submission_results, notices
-- 依存: PostgreSQL

-- updated_at を自動更新する共通関数
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ユーザー
CREATE TABLE IF NOT EXISTS users (
    id              BIGSERIAL PRIMARY KEY,
    username        VARCHAR(64) NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    role            VARCHAR(16) NOT NULL DEFAULT 'user',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_users_updated
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

-- 問題
CREATE TABLE IF NOT EXISTS problems (
    id               BIGSERIAL PRIMARY KEY,
    slug             VARCHAR(128) NOT NULL UNIQUE,
    title            TEXT NOT NULL,
    statement_path   TEXT,
    statement_md     TEXT,
    time_limit_ms    INTEGER NOT NULL DEFAULT 2000,
    memory_limit_kb  INTEGER NOT NULL DEFAULT 262144,
    is_public        BOOLEAN NOT NULL DEFAULT TRUE,
    checker_type     TEXT NOT NULL DEFAULT 'exact',
    checker_eps      DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_problems_updated
    BEFORE UPDATE ON problems
    FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

-- テストケース
CREATE TABLE IF NOT EXISTS testcases (
    id           BIGSERIAL PRIMARY KEY,
    problem_id   BIGINT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    input_path   TEXT,
    output_path  TEXT,
    input_text   TEXT,
    output_text  TEXT,
    is_sample    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_testcases_problem ON testcases(problem_id);

-- 提出
CREATE TABLE IF NOT EXISTS submissions (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    problem_id     BIGINT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    language       VARCHAR(32) NOT NULL,
    source_path    TEXT NOT NULL,
    status         VARCHAR(16) NOT NULL CHECK (status IN ('pending','running','succeeded','failed')),
    retry_count    INTEGER NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_submissions_user ON submissions(user_id);
CREATE INDEX IF NOT EXISTS idx_submissions_problem ON submissions(problem_id);
CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);
CREATE TRIGGER trg_submissions_updated
    BEFORE UPDATE ON submissions
    FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

-- 提出結果 (1:1)
CREATE TABLE IF NOT EXISTS submission_results (
    submission_id  BIGINT PRIMARY KEY REFERENCES submissions(id) ON DELETE CASCADE,
    verdict        VARCHAR(8),
    time_ms        INTEGER,
    memory_kb      INTEGER,
    stdout_path    TEXT,
    stderr_path    TEXT,
    exit_code      INTEGER,
    error_message  TEXT,
    passed_count   INTEGER NOT NULL DEFAULT 0,
    total_count    INTEGER NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- お知らせ
CREATE TABLE IF NOT EXISTS notices (
    id          BIGSERIAL PRIMARY KEY,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION set_notice_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_notice_updated_at
    BEFORE UPDATE ON notices
    FOR EACH ROW EXECUTE PROCEDURE set_notice_updated_at();
