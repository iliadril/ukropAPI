CREATE TABLE IF NOT EXISTS comments
(
    id                bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    created_at        timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    recommendation_id bigint                      NOT NULL REFERENCES recommendations ON DELETE CASCADE,
    user_id           bigint                      NOT NULL REFERENCES users ON DELETE CASCADE,
    content           text                        NOT NULL,
    version           integer                     NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS comments__recommendation_id__idx ON comments (recommendation_id);

INSERT INTO permissions (code)
VALUES ('comments:write');

INSERT INTO users_permissions (user_id, permission_id)
SELECT u.id, p.id
FROM users u
         CROSS JOIN permissions p
WHERE p.code = 'comments:write';