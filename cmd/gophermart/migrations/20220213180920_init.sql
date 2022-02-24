-- +goose Up
-- +goose StatementBegin
CREATE TABLE users
(
    id        SERIAL PRIMARY KEY,
    login     VARCHAR(255)    NOT NULL,
    password  VARCHAR(255)    NOT NULL,
    balance   DECIMAL(36, 18) NOT NULL DEFAULT 0.0,
    withdrawn DECIMAL(36, 18) NOT NULL DEFAULT 0.0
);

CREATE TABLE orders
(
    id          SERIAL PRIMARY KEY,
    number      VARCHAR(255) UNIQUE NOT NULL,
    user_id     INT                 NOT NULL REFERENCES users,
    accrual     DECIMAL(36, 18)     NOT NULL DEFAULT 0.0,
    status      VARCHAR(255)        NOT NULL,
    uploaded_at TIMESTAMP           NOT NULL
);

CREATE TABLE withdraw_history
(
    id           SERIAL PRIMARY KEY,
    order_number VARCHAR(255)    NOT NULL,
    user_id      INT             NOT NULL REFERENCES users,
    amount       DECIMAL(36, 18) NOT NULL DEFAULT 0.0,
    processed_at TIMESTAMP       NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
DROP TABLE orders;
DROP TABLE withdraw_history;
-- +goose StatementEnd
