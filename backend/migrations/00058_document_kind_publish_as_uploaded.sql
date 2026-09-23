-- +goose Up
-- Kinds that publish as uploaded (for example forms) keep the uploaded PDF or
-- Word file as the published document. Their text is indexed for search and
-- RAG only; it is never turned into editable Markdown, sections or blocks.
ALTER TABLE document_kinds ADD COLUMN publish_as_uploaded boolean NOT NULL DEFAULT false;
UPDATE document_kinds SET publish_as_uploaded = true WHERE slug = 'form';

-- +goose Down
ALTER TABLE document_kinds DROP COLUMN IF EXISTS publish_as_uploaded;
