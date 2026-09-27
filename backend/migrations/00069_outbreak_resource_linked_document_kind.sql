-- +goose Up
-- Resources that link a library document record that document's kind.
UPDATE outbreak_resources r
SET document_kind = dk.slug
FROM guideline_documents gd
JOIN document_kinds dk ON dk.id = gd.document_kind_id
WHERE r.resource_type = 'guideline'
  AND r.url ~ '^/public/guidelines/[0-9a-fA-F-]{36}$'
  AND gd.id = substring(r.url FROM '[0-9a-fA-F-]{36}$')::uuid;

-- +goose Down
UPDATE outbreak_resources SET document_kind = 'other' WHERE resource_type = 'guideline';
