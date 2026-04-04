-- 001_init.sql: initial schema

CREATE TABLE IF NOT EXISTS invitations (
    id         BIGSERIAL PRIMARY KEY,
    code       VARCHAR(255) NOT NULL,
    max_uses   INT          NOT NULL CHECK (max_uses BETWEEN 1 AND 1000),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT invitations_code_unique UNIQUE (code)
);

-- invitation_uses хранит каждую принятую пару (invitation, email).
-- Уникальное ограничение (UNIQUE constraint) на (invitation_id, email) выступает
-- жесткой гарантией на уровне БД в том, что один и тот же email 
-- не может зарегистрироваться дважды по одному и тому же приглашению, 
-- даже при экстремальной конкурентности/параллельности.
CREATE TABLE IF NOT EXISTS invitation_uses (
    id            BIGSERIAL PRIMARY KEY,
    invitation_id BIGINT      NOT NULL REFERENCES invitations (id) ON DELETE CASCADE,
    email         VARCHAR(320) NOT NULL, 
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT invitation_uses_unique UNIQUE (invitation_id, email)
);

CREATE INDEX IF NOT EXISTS idx_invitation_uses_invitation_id
    ON invitation_uses (invitation_id);
