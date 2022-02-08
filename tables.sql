CREATE TABLE users
(
    id       SERIAL PRIMARY KEY,
    login    VARCHAR(255)    NOT NULL,
    password VARCHAR(255)    NOT NULL,
    balance  DECIMAL(36, 18) NOT NULL DEFAULT 0.0
);

CREATE TABLE orders
(
    id          SERIAL PRIMARY KEY,
    number      BIGINT UNIQUE   NOT NULL,
    user_id     INT             NOT NULL REFERENCES users,
    accrual     DECIMAL(36, 18) NOT NULL DEFAULT 0.0,
    status      VARCHAR(255)    NOT NULL,
    uploaded_at TIMESTAMP       NOT NULL
);

CREATE TABLE withdraw_history
(
    id           SERIAL PRIMARY KEY,
    order_number BIGINT          NOT NULL REFERENCES orders (number),
    user_id      INT             NOT NULL REFERENCES users,
    amount       DECIMAL(36, 18) NOT NULL DEFAULT 0.0,
    withdrawn    DECIMAL(36, 18) NOT NULL DEFAULT 0.0
);