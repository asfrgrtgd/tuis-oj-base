-- シード問題を削除する（テーブルは保持する）
DO $$
DECLARE
    pid BIGINT;
BEGIN
    -- hello-world
    SELECT id INTO pid FROM problems WHERE slug = 'hello-world';
    IF pid IS NOT NULL THEN
        DELETE FROM testcases WHERE problem_id = pid;
        DELETE FROM problems WHERE id = pid;
    END IF;

    -- simple-a-plus-b
    SELECT id INTO pid FROM problems WHERE slug = 'simple-a-plus-b';
    IF pid IS NOT NULL THEN
        DELETE FROM testcases WHERE problem_id = pid;
        DELETE FROM problems WHERE id = pid;
    END IF;

    -- sort-numbers
    SELECT id INTO pid FROM problems WHERE slug = 'sort-numbers';
    IF pid IS NOT NULL THEN
        DELETE FROM testcases WHERE problem_id = pid;
        DELETE FROM problems WHERE id = pid;
    END IF;
END $$;
