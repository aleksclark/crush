-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN working_dir TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN working_dir;
-- +goose StatementEnd
