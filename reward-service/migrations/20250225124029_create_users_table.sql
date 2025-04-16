-- +goose Up
CREATE TABLE IF NOT EXISTS users(
    id serial PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    password VARCHAR(255) NOT NULL,
    active INT NOT NULL DEFAULT 1,
    score INT NOT NULL DEFAULT 0,
    referrer VARCHAR(255) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE UNIQUE INDEX idx_users_email ON users(email);
    CREATE INDEX idx_users_referrer ON users(referrer);
    CREATE INDEX idx_users_active_score ON users(active, score);
    CREATE INDEX idx_users_created_at ON users(created_at);
    CREATE INDEX idx_users_inactive ON users(email) WHERE active = 0;
    CREATE INDEX idx_users_name ON users(first_name, last_name);
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS users;
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
