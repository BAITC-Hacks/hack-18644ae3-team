CREATE TABLE IF NOT EXISTS application_roles (
    id SMALLSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE CHECK (code IN ('HR', 'EMPLOYEE', 'LD_SPECIALIST'))
);

CREATE TABLE IF NOT EXISTS job_roles (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS grades (
    id SMALLSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    rank SMALLINT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS skills (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('hard', 'soft')),
    category TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS role_skill_requirements (
    job_role_id BIGINT NOT NULL REFERENCES job_roles(id),
    grade_id SMALLINT NOT NULL REFERENCES grades(id),
    skill_id TEXT NOT NULL REFERENCES skills(id),
    required_level SMALLINT NOT NULL CHECK (required_level BETWEEN 0 AND 5),
    critical BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (job_role_id, grade_id, skill_id)
);

CREATE TABLE IF NOT EXISTS employees (
    id TEXT PRIMARY KEY,
    full_name TEXT NOT NULL,
    email TEXT NULL,
    phone TEXT NULL,
    department TEXT NOT NULL DEFAULT '',
    team TEXT NOT NULL DEFAULT '',
    manager_id TEXT NULL REFERENCES employees(id),
    location TEXT NOT NULL DEFAULT '',
    job_role_id BIGINT NOT NULL REFERENCES job_roles(id),
    grade_id SMALLINT NOT NULL REFERENCES grades(id),
    hire_date DATE NULL,
    tenure_months INTEGER NOT NULL DEFAULT 0,
    work_format TEXT NOT NULL DEFAULT '',
    preferred_language TEXT NOT NULL DEFAULT '',
    last_review_date DATE NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS employees_email_unique ON employees (LOWER(email)) WHERE email IS NOT NULL AND email <> '';
CREATE INDEX IF NOT EXISTS employees_search_idx ON employees (department, team, job_role_id, grade_id);

CREATE TABLE IF NOT EXISTS employee_skills (
    employee_id TEXT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    skill_id TEXT NOT NULL REFERENCES skills(id),
    level SMALLINT NOT NULL CHECK (level BETWEEN 0 AND 5),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (employee_id, skill_id)
);

CREATE TABLE IF NOT EXISTS career_goals (
    employee_id TEXT PRIMARY KEY REFERENCES employees(id) ON DELETE CASCADE,
    target_job_role_id BIGINT NOT NULL REFERENCES job_roles(id),
    target_grade_id SMALLINT NOT NULL REFERENCES grades(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'ACTIVE', 'REJECTED')),
    application_role_id SMALLINT NOT NULL REFERENCES application_roles(id),
    employee_id TEXT NULL UNIQUE REFERENCES employees(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique ON users (LOWER(email));
CREATE INDEX IF NOT EXISTS users_status_idx ON users (status);

CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL,
    format TEXT NOT NULL CHECK (format IN ('online', 'offline', 'self_paced')),
    duration_hours NUMERIC(8,2) NOT NULL CHECK (duration_hours > 0),
    mandatory BOOLEAN NOT NULL DEFAULT FALSE,
    learning_link TEXT NOT NULL DEFAULT '',
    created_by TEXT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS event_target_roles (
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    job_role_id BIGINT NOT NULL REFERENCES job_roles(id),
    PRIMARY KEY (event_id, job_role_id)
);

CREATE TABLE IF NOT EXISTS event_target_grades (
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    grade_id SMALLINT NOT NULL REFERENCES grades(id),
    PRIMARY KEY (event_id, grade_id)
);

CREATE TABLE IF NOT EXISTS event_skill_effects (
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    skill_id TEXT NOT NULL REFERENCES skills(id),
    gain SMALLINT NOT NULL CHECK (gain > 0),
    max_level SMALLINT NOT NULL CHECK (max_level BETWEEN 1 AND 5),
    PRIMARY KEY (event_id, skill_id)
);

CREATE TABLE IF NOT EXISTS event_prerequisites (
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    skill_id TEXT NOT NULL REFERENCES skills(id),
    minimum_level SMALLINT NOT NULL CHECK (minimum_level BETWEEN 0 AND 5),
    PRIMARY KEY (event_id, skill_id)
);

CREATE TABLE IF NOT EXISTS course_sessions (
    id BIGSERIAL PRIMARY KEY,
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    session_date DATE NOT NULL,
    external_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, session_date)
);

CREATE TABLE IF NOT EXISTS enrollments (
    id BIGSERIAL PRIMARY KEY,
    employee_id TEXT NOT NULL REFERENCES employees(id),
    event_id TEXT NOT NULL REFERENCES events(id),
    session_id BIGINT NULL REFERENCES course_sessions(id),
    status TEXT NOT NULL,
    completion_pct SMALLINT NOT NULL DEFAULT 0 CHECK (completion_pct BETWEEN 0 AND 100),
    assigned_by TEXT NOT NULL DEFAULT 'self',
    due_date DATE NULL,
    source_record_id TEXT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS enrollments_employee_idx ON enrollments (employee_id, status);
CREATE INDEX IF NOT EXISTS enrollments_event_idx ON enrollments (event_id, status);

CREATE TABLE IF NOT EXISTS activity_history (
    id TEXT PRIMARY KEY,
    enrollment_id BIGINT NULL REFERENCES enrollments(id),
    employee_id TEXT NOT NULL REFERENCES employees(id),
    event_id TEXT NOT NULL REFERENCES events(id),
    activity_date DATE NOT NULL,
    due_date DATE NULL,
    status TEXT NOT NULL,
    completion_pct SMALLINT NOT NULL CHECK (completion_pct BETWEEN 0 AND 100),
    score SMALLINT NULL CHECK (score BETWEEN 0 AND 100),
    feedback_rating SMALLINT NULL CHECK (feedback_rating BETWEEN 1 AND 5),
    assigned_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS activity_history_employee_idx ON activity_history (employee_id, activity_date DESC);

CREATE TABLE IF NOT EXISTS assessments (
    id BIGSERIAL PRIMARY KEY,
    enrollment_id BIGINT NOT NULL UNIQUE REFERENCES enrollments(id),
    employee_id TEXT NOT NULL REFERENCES employees(id),
    event_id TEXT NOT NULL REFERENCES events(id),
    result TEXT NULL CHECK (result IN ('PASSED', 'FAILED')),
    score SMALLINT NULL CHECK (score BETWEEN 0 AND 100),
    feedback TEXT NOT NULL DEFAULT '',
    assessed_by TEXT NULL REFERENCES users(id),
    assessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    skill_rewards_applied BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS app_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
