-- +goose Up
-- Seed admin user for demo purposes
-- Email: admin@example.com
-- Password: admin123
INSERT INTO users (id, created_at, updated_at, email, hashed_password, role)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    NOW(),
    NOW(),
    'admin@example.com',
    '$argon2id$v=19$m=65536,t=1,p=18$E7WbT5nBcscmDMuaMnjG1g$kypkhRwMUGevCKMVXD71+CDW3MIAnJY1wZ0BwUsCAKg',
    'admin'
) ON CONFLICT (email) DO UPDATE SET
    hashed_password = EXCLUDED.hashed_password,
    role = EXCLUDED.role;

-- +goose Down
DELETE FROM users WHERE email = 'admin@example.com';
