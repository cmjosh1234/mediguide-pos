-- +goose Up
CREATE TABLE tool_categories (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  name text NOT NULL,
  slug text NOT NULL,
  description text,
  sort_order integer NOT NULL DEFAULT 0,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
  color text,
  icon text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE UNIQUE INDEX idx_tool_categories_slug ON tool_categories(slug) WHERE deleted_at IS NULL;

INSERT INTO tool_categories (name, slug, description, sort_order, color, icon) VALUES
  ('Emergency & Triage', 'emergency-triage', 'Triage, resuscitation and urgent assessment tools.', 10, '#D84315', 'siren'),
  ('Maternal & Newborn', 'maternal-newborn', 'Pregnancy, delivery and newborn assessment tools.', 20, '#AD1457', 'baby'),
  ('Child Health', 'child-health', 'Paediatric assessment, fever and immunisation tools.', 30, '#00838F', 'hand-heart'),
  ('Medication & Dosing', 'medication-dosing', 'Dose calculations and medicine safety tools.', 40, '#2E7D32', 'pill'),
  ('Cardiovascular', 'cardiovascular', 'Blood pressure and cardiovascular risk tools.', 50, '#6A1B9A', 'heart-pulse'),
  ('Neurology', 'neurology', 'Consciousness and neurological assessment tools.', 60, '#4527A0', 'brain'),
  ('Fluids & Nutrition', 'fluids-nutrition', 'Hydration, fluid balance and nutrition tools.', 70, '#0277BD', 'droplets'),
  ('General Assessment', 'general-assessment', 'Pain, wound and other general clinical assessments.', 80, '#455A64', 'stethoscope');

ALTER TABLE calculators
  ADD COLUMN category_id uuid REFERENCES tool_categories(id) ON DELETE RESTRICT,
  ADD COLUMN risk_level text NOT NULL DEFAULT 'standard' CHECK (risk_level IN ('standard', 'high', 'critical'));
CREATE INDEX idx_calculators_category_id ON calculators(category_id);

-- Assign the seeded clinical tools; tools with other names stay uncategorised.
UPDATE calculators SET category_id = c.id
FROM tool_categories c,
  (VALUES
    ('Emergency Triage Assessment Tool', 'emergency-triage'),
    ('APGAR Score Calculator', 'maternal-newborn'),
    ('Pregnancy Due Date Calculator', 'maternal-newborn'),
    ('Pediatric Fever Management Tool', 'child-health'),
    ('Immunization Schedule Checker', 'child-health'),
    ('Medication Dosage Calculator', 'medication-dosing'),
    ('Blood Pressure Risk Assessment', 'cardiovascular'),
    ('Cardiac Risk Assessment Tool', 'cardiovascular'),
    ('Glasgow Coma Scale Assessment', 'neurology'),
    ('Dehydration Assessment Tool', 'fluids-nutrition'),
    ('Fluid Balance Calculator', 'fluids-nutrition'),
    ('BMI Calculator', 'fluids-nutrition'),
    ('Comprehensive Pain Assessment Scale', 'general-assessment'),
    ('Wound Assessment and Care Tool', 'general-assessment')
  ) AS mapping(tool_name, category_slug)
WHERE calculators.name = mapping.tool_name AND c.slug = mapping.category_slug;

-- Start every tool at the level the previous name-based rule implied:
-- dosing, medication, triage and emergency tools are critical, other
-- decision tools needed independent approval (high).
UPDATE calculators SET risk_level = 'critical'
WHERE LOWER(name) LIKE '%dose%' OR LOWER(name) LIKE '%medication%'
   OR LOWER(name) LIKE '%triage%' OR LOWER(name) LIKE '%emergency%';
UPDATE calculators SET risk_level = 'high'
WHERE risk_level = 'standard' AND type = 'decision_tool';

-- +goose Down
DROP INDEX idx_calculators_category_id;
ALTER TABLE calculators DROP COLUMN risk_level, DROP COLUMN category_id;
DROP TABLE tool_categories;
