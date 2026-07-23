-- +goose Up

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    institution TEXT NOT NULL DEFAULT '',
    account_type TEXT NOT NULL CHECK (account_type IN ('checking','credit','cash','investment','other')),
    currency TEXT NOT NULL DEFAULT 'JPY',
    opening_balance_minor BIGINT NOT NULL DEFAULT 0,
    opening_balance_date DATE NOT NULL DEFAULT CURRENT_DATE,
    is_archived BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    parent_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    kind TEXT NOT NULL CHECK (kind IN ('expense','income','transfer')) DEFAULT 'expense',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(name, parent_id)
);

CREATE TABLE category_rules (
    id BIGSERIAL PRIMARY KEY,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    match_text TEXT NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE import_profiles (
    account_id BIGINT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    date_col TEXT NOT NULL,
    desc_col TEXT NOT NULL,
    amount_col TEXT NOT NULL,
    date_layout TEXT NOT NULL DEFAULT '2006/01/02',
    invert_amount BOOLEAN NOT NULL DEFAULT false,
    has_header BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE import_batches (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    source_filename TEXT NOT NULL,
    row_count INT NOT NULL DEFAULT 0,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    txn_date DATE NOT NULL,
    merchant_raw TEXT NOT NULL,
    merchant_normalized TEXT NOT NULL,
    amount_minor BIGINT NOT NULL,
    category_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    note TEXT NOT NULL DEFAULT '',
    import_batch_id BIGINT REFERENCES import_batches(id) ON DELETE SET NULL,
    dedup_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(account_id, dedup_hash)
);
CREATE INDEX idx_transactions_account_date ON transactions(account_id, txn_date);
CREATE INDEX idx_transactions_category ON transactions(category_id);

CREATE TABLE balance_snapshots (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    snapshot_date DATE NOT NULL,
    balance_minor BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(account_id, snapshot_date)
);

INSERT INTO categories (name, kind) VALUES
 ('Groceries','expense'),
 ('Dining Out','expense'),
 ('Transport','expense'),
 ('Shopping','expense'),
 ('Subscriptions','expense'),
 ('Utilities','expense'),
 ('Housing','expense'),
 ('Health','expense'),
 ('Entertainment','expense'),
 ('Income','income'),
 ('Transfer','transfer'),
 ('Uncategorized','expense');

-- +goose Down
DROP TABLE balance_snapshots;
DROP TABLE transactions;
DROP TABLE import_batches;
DROP TABLE import_profiles;
DROP TABLE category_rules;
DROP TABLE categories;
DROP TABLE accounts;
DROP TABLE sessions;
DROP TABLE users;
