-- +goose Up
ALTER TABLE category_rules ADD CONSTRAINT category_rules_match_text_key UNIQUE (match_text);

-- +goose Down
ALTER TABLE category_rules DROP CONSTRAINT category_rules_match_text_key;
