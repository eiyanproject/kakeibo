-- +goose Up
INSERT INTO categories (name, kind) VALUES ('Personal Improvement', 'expense');

-- +goose Down
DELETE FROM categories WHERE name = 'Personal Improvement' AND parent_id IS NULL;
