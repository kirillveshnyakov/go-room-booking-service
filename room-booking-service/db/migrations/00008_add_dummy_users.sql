-- +goose Up

INSERT INTO users (id,
                   email,
                   role,
                   password_hash)
VALUES ('00000000-0000-0000-0000-000000000001',
        'dummy-admin@local.invalid',
        'admin',
        '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'),
       ('00000000-0000-0000-0000-000000000002',
        'dummy-user@local.invalid',
        'user',
        '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy');

-- +goose Down

DELETE
FROM users
WHERE id IN (
             '00000000-0000-0000-0000-000000000001',
             '00000000-0000-0000-0000-000000000002'
    );