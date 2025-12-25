-- Drop all base tables, triggers, and helper functions

-- drop triggers first to avoid dependency errors
DROP TRIGGER IF EXISTS trg_notice_updated_at ON notices;
DROP TRIGGER IF EXISTS trg_submissions_updated ON submissions;
DROP TRIGGER IF EXISTS trg_problems_updated ON problems;
DROP TRIGGER IF EXISTS trg_users_updated ON users;

-- drop functions
DROP FUNCTION IF EXISTS set_notice_updated_at();
DROP FUNCTION IF EXISTS set_updated_at();

-- drop tables (including optional detail table if存在)
DROP TABLE IF EXISTS submission_result_details;
DROP TABLE IF EXISTS submission_results;
DROP TABLE IF EXISTS submissions;
DROP TABLE IF EXISTS testcases;
DROP TABLE IF EXISTS problems;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS notices;
