-- +goose Up
ALTER TABLE users
ADD COLUMN refresh_token TEXT,
ADD COLUMN refresh_token_expires TIMESTAMP;

CREATE INDEX idx_users_refresh_token ON users(refresh_token);
CREATE INDEX idx_users_refresh_token_expires ON users(refresh_token_expires);
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

-- +goose Down
ALTER TABLE users
DROP COLUMN refresh_token,
DROP COLUMN refresh_token_expires;
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd