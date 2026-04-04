-- 002_seed.sql: seed с кодами приглашений.
-- ON CONFLICT делает скрипт идемпотентным - безопасно запускать несколько раз.

INSERT INTO invitations (code, max_uses)
VALUES
    ('twitter-reg1',    100),
    ('telegram-test',    50),
    ('instagram-hello', 200)
ON CONFLICT (code) DO NOTHING;
