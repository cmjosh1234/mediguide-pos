-- +goose Up
-- Document kinds are shared by guideline documents and outbreak documents.
ALTER TABLE guideline_document_kinds RENAME TO document_kinds;
ALTER INDEX guideline_document_kinds_pkey RENAME TO document_kinds_pkey;
ALTER TABLE document_kinds RENAME CONSTRAINT guideline_document_kinds_name_check TO document_kinds_name_check;
ALTER TABLE document_kinds RENAME CONSTRAINT guideline_document_kinds_sort_order_check TO document_kinds_sort_order_check;
ALTER TABLE document_kinds RENAME CONSTRAINT guideline_document_kinds_status_check TO document_kinds_status_check;

-- Slugs are the stable codes clients filter and group by (for example
-- contact_tracing_guide), so they allow underscores, stay unique even after a
-- kind is archived, and are the target of the outbreak document foreign key.
ALTER TABLE document_kinds DROP CONSTRAINT guideline_document_kinds_slug_check;
ALTER TABLE document_kinds ADD CONSTRAINT document_kinds_slug_check CHECK (slug ~ '^[a-z0-9]+([_-][a-z0-9]+)*$');
DROP INDEX idx_guideline_document_kinds_slug;
ALTER TABLE document_kinds ADD CONSTRAINT document_kinds_slug_key UNIQUE (slug);

INSERT INTO document_kinds (id, name, slug, description, sort_order) VALUES
  ('94000000-0000-4000-8000-000000000003', 'SOP', 'sop', 'Standard operating procedures.', 20),
  ('94000000-0000-4000-8000-000000000004', 'Case Definition', 'case_definition', 'Standard, suspected, probable and confirmed case definitions.', 30),
  ('94000000-0000-4000-8000-000000000005', 'IPC Protocol', 'ipc_protocol', 'Infection prevention and control protocols.', 40),
  ('94000000-0000-4000-8000-000000000006', 'Laboratory Protocol', 'laboratory_protocol', 'Specimen collection, handling and testing protocols.', 50),
  ('94000000-0000-4000-8000-000000000007', 'Surveillance Protocol', 'surveillance_protocol', 'Surveillance and reporting protocols.', 60),
  ('94000000-0000-4000-8000-000000000008', 'Contact Tracing Guide', 'contact_tracing_guide', 'Contact identification, listing and follow-up guides.', 70),
  ('94000000-0000-4000-8000-000000000009', 'Treatment Protocol', 'treatment_protocol', 'Clinical management and treatment protocols.', 80),
  ('94000000-0000-4000-8000-000000000010', 'Referral Protocol', 'referral_protocol', 'Referral and patient transfer protocols.', 90),
  ('94000000-0000-4000-8000-000000000011', 'Training Material', 'training_material', 'Training and orientation materials.', 100),
  ('94000000-0000-4000-8000-000000000012', 'Checklist', 'checklist', 'Operational and clinical checklists.', 110),
  ('94000000-0000-4000-8000-000000000013', 'Communication Material', 'communication_material', 'Risk communication and community engagement materials.', 120),
  ('94000000-0000-4000-8000-000000000014', 'Policy', 'policy', 'Policies and directives.', 140),
  ('94000000-0000-4000-8000-000000000015', 'Situation Report Attachment', 'situation_report_attachment', 'Attachments published with situation reports.', 150),
  ('94000000-0000-4000-8000-000000000016', 'Other', 'other', 'Documents that fit no other kind.', 160)
ON CONFLICT (slug) DO NOTHING;
-- Place Form among the outbreak kinds unless an editor already reordered it.
UPDATE document_kinds SET sort_order = 130
WHERE id = '94000000-0000-4000-8000-000000000002' AND sort_order = 20;

-- Outbreak documents reference a document kind instead of a fixed list.
ALTER TABLE outbreak_resources DROP CONSTRAINT outbreak_resources_document_kind_check;
ALTER TABLE outbreak_resources
  ADD CONSTRAINT outbreak_resources_document_kind_fkey
  FOREIGN KEY (document_kind) REFERENCES document_kinds(slug) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE outbreak_resources DROP CONSTRAINT IF EXISTS outbreak_resources_document_kind_fkey;
UPDATE outbreak_resources SET document_kind = 'other'
WHERE document_kind NOT IN (
  'sop','case_definition','ipc_protocol','laboratory_protocol','surveillance_protocol',
  'contact_tracing_guide','treatment_protocol','referral_protocol','training_material',
  'checklist','communication_material','form','policy','situation_report_attachment','other'
);
ALTER TABLE outbreak_resources
  ADD CONSTRAINT outbreak_resources_document_kind_check CHECK (
    document_kind IN (
      'sop','case_definition','ipc_protocol','laboratory_protocol','surveillance_protocol',
      'contact_tracing_guide','treatment_protocol','referral_protocol','training_material',
      'checklist','communication_material','form','policy','situation_report_attachment','other'
    )
  );

-- Kinds added by this migration, or whose slugs the previous rules reject,
-- are removed; guideline documents using them fall back to Guideline.
UPDATE guideline_documents SET document_kind_id = '94000000-0000-4000-8000-000000000001'
WHERE document_kind_id IN (
  SELECT id FROM document_kinds
  WHERE id BETWEEN '94000000-0000-4000-8000-000000000003' AND '94000000-0000-4000-8000-000000000016'
     OR slug !~ '^[a-z0-9]+(-[a-z0-9]+)*$'
);
DELETE FROM document_kinds
WHERE id BETWEEN '94000000-0000-4000-8000-000000000003' AND '94000000-0000-4000-8000-000000000016'
   OR slug !~ '^[a-z0-9]+(-[a-z0-9]+)*$';
UPDATE document_kinds SET sort_order = 20
WHERE id = '94000000-0000-4000-8000-000000000002' AND sort_order = 130;

ALTER TABLE document_kinds DROP CONSTRAINT document_kinds_slug_key;
CREATE UNIQUE INDEX idx_guideline_document_kinds_slug ON document_kinds (slug) WHERE deleted_at IS NULL;
ALTER TABLE document_kinds DROP CONSTRAINT document_kinds_slug_check;
ALTER TABLE document_kinds ADD CONSTRAINT guideline_document_kinds_slug_check CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$');
ALTER TABLE document_kinds RENAME CONSTRAINT document_kinds_status_check TO guideline_document_kinds_status_check;
ALTER TABLE document_kinds RENAME CONSTRAINT document_kinds_sort_order_check TO guideline_document_kinds_sort_order_check;
ALTER TABLE document_kinds RENAME CONSTRAINT document_kinds_name_check TO guideline_document_kinds_name_check;
ALTER INDEX document_kinds_pkey RENAME TO guideline_document_kinds_pkey;
ALTER TABLE document_kinds RENAME TO guideline_document_kinds;
