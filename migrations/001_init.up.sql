-- Users
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username    VARCHAR(30)  UNIQUE NOT NULL,
    email       VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT        NOT NULL,
    fcm_token   TEXT         DEFAULT '',
    timezone    VARCHAR(50)  DEFAULT 'Asia/Jakarta',
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  DEFAULT NOW()
);

-- Groups
CREATE TABLE groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(50)  NOT NULL,
    created_by  UUID         NOT NULL REFERENCES users(id),
    max_members INT          DEFAULT 6,
    created_at  TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE group_members (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id  UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id),
    role      VARCHAR(20) DEFAULT 'member',
    status    VARCHAR(20) DEFAULT 'active',
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(group_id, user_id)
);

CREATE TABLE group_invites (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id   UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    invited_by UUID NOT NULL REFERENCES users(id),
    token      VARCHAR(64) UNIQUE NOT NULL,
    status     VARCHAR(20) DEFAULT 'pending',
    max_uses   INT DEFAULT 1,
    used_count INT DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- App Limits
CREATE TABLE app_limits (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID NOT NULL REFERENCES users(id),
    group_id           UUID NOT NULL REFERENCES groups(id),
    package_name       VARCHAR(255) NOT NULL,
    app_label          VARCHAR(100) NOT NULL,
    daily_limit_minutes INT NOT NULL DEFAULT 60,
    is_active          BOOLEAN DEFAULT TRUE,
    created_at         TIMESTAMPTZ DEFAULT NOW(),
    updated_at         TIMESTAMPTZ DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ
);

-- Limit Extensions (vote requests)
CREATE TABLE limit_extensions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    limit_id      UUID NOT NULL REFERENCES app_limits(id),
    requested_by  UUID NOT NULL REFERENCES users(id),
    extra_minutes INT  NOT NULL,
    reason        TEXT DEFAULT '',
    status        VARCHAR(20) DEFAULT 'pending',
    votes_needed  INT NOT NULL DEFAULT 2,
    votes_yes     INT DEFAULT 0,
    votes_no      INT DEFAULT 0,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    resolved_at   TIMESTAMPTZ
);

CREATE TABLE extension_votes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    extension_id UUID NOT NULL REFERENCES limit_extensions(id) ON DELETE CASCADE,
    voter_id     UUID NOT NULL REFERENCES users(id),
    vote         VARCHAR(10) NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(extension_id, voter_id)
);

-- Usage logs
CREATE TABLE usage_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id),
    package_name VARCHAR(255) NOT NULL,
    duration_sec INT NOT NULL,
    recorded_at  TIMESTAMPTZ NOT NULL,
    synced_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_usage_logs_user_date ON usage_logs(user_id, recorded_at);

-- Rewards
CREATE TABLE user_streaks (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID UNIQUE NOT NULL REFERENCES users(id),
    current_days INT DEFAULT 0,
    longest_days INT DEFAULT 0,
    last_clean   DATE,
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE user_badges (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID NOT NULL REFERENCES users(id),
    badge     VARCHAR(50) NOT NULL,
    earned_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, badge)
);
