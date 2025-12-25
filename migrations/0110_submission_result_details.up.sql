-- submission_result_details テーブル（テストケースごとの結果保存）
CREATE TABLE IF NOT EXISTS submission_result_details (
    id             BIGSERIAL PRIMARY KEY,
    submission_id  BIGINT NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    testcase       TEXT NOT NULL,
    status         VARCHAR(8),
    time_ms        INTEGER,
    memory_kb      INTEGER,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_submission_result_details_submission ON submission_result_details(submission_id);
