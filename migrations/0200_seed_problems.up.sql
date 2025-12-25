-- 開発・デモ用の初期問題セットを投入する。
-- hello-world / simple-a-plus-b / sort-numbers（最終版: 2行入力・複数生成ケース）

DO $$
DECLARE
    pid BIGINT;
    input_line TEXT;
    output_line TEXT;
    rec RECORD;
BEGIN
    ------------------------------------------------------------------
    -- Hello World
    ------------------------------------------------------------------
    INSERT INTO problems (slug, title, statement_md, time_limit_ms, memory_limit_kb, is_public)
    VALUES (
        'hello-world',
        'Hello World',
        E'## 問題文\n標準出力に `Hello, World!` を 1 行で出力してください。入力はありません。\n\n## 制約\n- 制約はありません\n\n## 入力\n入力は与えられません。\n\n## 出力\n```\nHello, World!\n```\n',
        1000,
        262144,
        TRUE
    )
    ON CONFLICT (slug) DO NOTHING;

    SELECT id INTO pid FROM problems WHERE slug = 'hello-world';
    IF pid IS NOT NULL THEN
        DELETE FROM testcases WHERE problem_id = pid;
        INSERT INTO testcases (problem_id, input_path, output_path, input_text, output_text, is_sample)
        VALUES
          (pid, '', '', '', E'Hello, World!\n', TRUE),
          (pid, '', '', '', E'Hello, World!\n', FALSE);
    END IF;

    ------------------------------------------------------------------
    -- Simple A+B
    ------------------------------------------------------------------
    INSERT INTO problems (slug, title, statement_md, time_limit_ms, memory_limit_kb, is_public)
    VALUES (
        'simple-a-plus-b',
        'Simple A+B',
        E'## 問題文\n2 つの整数 A, B が与えられます。A + B を出力してください。\n\n## 制約\n- 1 ≤ A ≤ 1,000,000\n- 1 ≤ B ≤ 1,000,000\n- 入力はすべて整数\n\n## 入力\n```\nA B\n```\n\n## 出力\n```\nA + B の値を出力せよ。\n```\n',
        2000,
        262144,
        TRUE
    )
    ON CONFLICT (slug) DO NOTHING;

    SELECT id INTO pid FROM problems WHERE slug = 'simple-a-plus-b';
    IF pid IS NOT NULL THEN
        DELETE FROM testcases WHERE problem_id = pid;
        INSERT INTO testcases (problem_id, input_path, output_path, input_text, output_text, is_sample)
        VALUES
          (pid, '', '', E'1 41\n', E'42\n', TRUE),
          (pid, '', '', E'0 0\n', E'0\n', FALSE),
          (pid, '', '', E'999999 1\n', E'1000000\n', FALSE);
    END IF;

    ------------------------------------------------------------------
    -- Number Sorting (最終版: 2行入力固定、生成ケース複数)
    ------------------------------------------------------------------
    INSERT INTO problems (slug, title, statement_md, time_limit_ms, memory_limit_kb, is_public)
    VALUES (
        'sort-numbers',
        'Number Sorting',
        E'## 問題文\n与えられた整数列を昇順に並べて出力してください。\n\n## 制約\n- 1 ≤ N ≤ 100,000\n- 0 ≤ Ai ≤ 100,000\n\n## 入力\n```\nN\nA1 A2 ... AN\n```\n\n## 出力\n```\nA1 ≤ A2 ≤ ... ≤ AN\n```\n',
        2000,
        262144,
        TRUE
    )
    ON CONFLICT (slug) DO NOTHING;

    SELECT id INTO pid FROM problems WHERE slug = 'sort-numbers';
    IF pid IS NULL THEN
        RAISE NOTICE 'sort-numbers already exists; skipping seed';
    ELSE
        DELETE FROM testcases WHERE problem_id = pid;

        -- サンプル2件
        INSERT INTO testcases (problem_id, input_path, output_path, input_text, output_text, is_sample) VALUES
          (pid, '', '', E'5\n3 1 4 1 5\n', E'1 1 3 4 5\n', TRUE),
          (pid, '', '', E'1\n42\n', E'42\n', TRUE);

        -- 固定ケース（小〜中規模）
        INSERT INTO testcases (problem_id, input_path, output_path, input_text, output_text, is_sample) VALUES
          (pid, '', '', E'10\n10 9 8 7 6 5 4 3 2 1\n', E'1 2 3 4 5 6 7 8 9 10\n', FALSE),
          (pid, '', '', E'12\n0 100000 1 99999 50000 2 3 100000 0 42 73 99998\n', E'0 0 1 2 3 42 73 50000 99998 99999 100000 100000\n', FALSE);

        -- 生成ケース（全て 2 行構成）
        FOR rec IN
            SELECT * FROM (VALUES
                (30, 48271),   -- small random
                (200, 69621),  -- medium random
                (1000, 12345), -- 1k random
                (5000, 7777),  -- 5k random
                (12000, 31415),-- 12k random
                (75000, 27183),-- 75k random
                (100000, 48271) -- 100k hidden
            ) AS t(n, mul)
        LOOP
            SELECT rec.n || E'\n' || string_agg(v::text, ' ' ORDER BY i) || E'\n'
              INTO input_line
              FROM (
                SELECT i, (((i::bigint * rec.mul + 1) % 100001)::int) AS v
                FROM generate_series(1, rec.n) AS gs(i)
              ) s;

            SELECT string_agg(v::text, ' ' ORDER BY v, i) || E'\n'
              INTO output_line
              FROM (
                SELECT i, (((i::bigint * rec.mul + 1) % 100001)::int) AS v
                FROM generate_series(1, rec.n) AS gs(i)
              ) s;

            INSERT INTO testcases (problem_id, input_path, output_path, input_text, output_text, is_sample)
            VALUES (pid, '', '', input_line, output_line, FALSE);
        END LOOP;
    END IF;

    ------------------------------------------------------------------
    -- Stairs Coloring DP
    ------------------------------------------------------------------
    INSERT INTO problems (slug, title, statement_md, time_limit_ms, memory_limit_kb, is_public)
    VALUES (
        'stairs-coloring',
        'Stairs Coloring',
        E'## 問題文\n長さ N の階段を赤 (R)・緑 (G)・青 (B) のいずれか 1 色で塗り、隣り合う段のうち同じ色で塗られているペアがちょうど K 個になるような塗り方の数を求めよ。答えは 998244353 で割った余りを出力せよ。\n\n## 制約\n- 1 ≤ N ≤ 2000\n- 0 ≤ K ≤ N-1\n\n## 入力\n```\nN K\n```\n\n## 出力\n```\n条件を満たす塗り方の数（998244353 で割った余り）を 1 行に出力せよ。\n```\n',
        2000,
        262144,
        TRUE
    )
    ON CONFLICT (slug) DO NOTHING;

    SELECT id INTO pid FROM problems WHERE slug = 'stairs-coloring';
    IF pid IS NOT NULL THEN
        DELETE FROM testcases WHERE problem_id = pid;
        INSERT INTO testcases (problem_id, input_path, output_path, input_text, output_text, is_sample) VALUES
          (pid, '', '', E'4 1\n', E'36\n', TRUE),
          (pid, '', '', E'3 0\n', E'12\n', TRUE),
          (pid, '', '', E'3 2\n', E'3\n', TRUE),
          (pid, '', '', E'1 0\n', E'3\n', FALSE),
          (pid, '', '', E'2 1\n', E'3\n', FALSE),
          (pid, '', '', E'5 0\n', E'48\n', FALSE),
          (pid, '', '', E'5 2\n', E'72\n', FALSE),
          (pid, '', '', E'10 3\n', E'16128\n', FALSE),
          (pid, '', '', E'200 50\n', E'324791188\n', FALSE),
          (pid, '', '', E'2000 0\n', E'564324881\n', FALSE),
          (pid, '', '', E'2000 1000\n', E'35548189\n', FALSE),
          (pid, '', '', E'2000 1999\n', E'3\n', FALSE);
    END IF;
END $$;
