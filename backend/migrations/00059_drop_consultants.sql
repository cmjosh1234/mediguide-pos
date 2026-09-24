-- +goose Up
-- The consultant directory has been removed from every client and the API.
UPDATE notifications
SET action_json = '{"type":"none","parameters":{}}'::jsonb
WHERE action_json->>'route' = '/consultants' OR action_json->>'route' LIKE '/consultants/%';

DROP TABLE IF EXISTS consultant_usage_logs;
DROP TABLE IF EXISTS consultants;

-- +goose Down
-- Recreates the empty tables only; dropped rows and reset notification actions are not restored.
CREATE TABLE consultants (
    id uuid DEFAULT uuid_generate_v4() PRIMARY KEY,
    user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    name text NOT NULL,
    email text NOT NULL,
    phone text NOT NULL,
    alternative_phone text,
    profile_picture_json jsonb,
    avatar_json jsonb,
    specialty text NOT NULL,
    license_number text,
    years_of_experience double precision,
    qualifications text,
    certifications text,
    address text,
    city text,
    region text,
    country text NOT NULL,
    postal_code text,
    organization text,
    department text,
    preferred_language text,
    timezone text,
    availability_json jsonb,
    consultation_types text,
    status text NOT NULL,
    is_verified boolean DEFAULT false NOT NULL,
    rating double precision,
    total_consultations integer DEFAULT 0 NOT NULL,
    notes text,
    usage_count bigint DEFAULT 0 NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    deleted_at timestamptz,
    CONSTRAINT chk_consultants_consultation_types CHECK (consultation_types IS NULL OR consultation_types IN ('In-Person', 'Telemedicine', 'Phone Consultation', 'Emergency Consultation', 'Second Opinion', 'Follow-up', 'Diagnostic Review', 'Treatment Planning', 'Medication Review', 'Health Education')),
    CONSTRAINT chk_consultants_preferred_language CHECK (preferred_language IS NULL OR preferred_language IN ('English', 'French', 'Spanish', 'Portuguese', 'Arabic', 'Swahili', 'Amharic', 'Other')),
    CONSTRAINT chk_consultants_qualifications CHECK (qualifications IS NULL OR qualifications IN ('MD', 'MBBS', 'DO', 'DDS', 'PharmD', 'RN', 'BSN', 'MSN', 'DNP', 'PhD', 'MPH', 'MS', 'MA', 'Diploma', 'Certificate', 'Fellowship', 'Residency', 'Other')),
    CONSTRAINT chk_consultants_specialty CHECK (specialty IS NULL OR specialty IN ('General Practice', 'Internal Medicine', 'Pediatrics', 'Surgery', 'Cardiology', 'Neurology', 'Psychiatry', 'Orthopedics', 'Dermatology', 'Obstetrics & Gynecology', 'Ophthalmology', 'Emergency Medicine', 'Radiology', 'Anesthesiology', 'Pathology', 'Oncology', 'Endocrinology', 'Gastroenterology', 'Pulmonology', 'Nephrology', 'Infectious Diseases', 'Rheumatology', 'Public Health', 'Nursing', 'Pharmacy', 'Laboratory Medicine', 'Other')),
    CONSTRAINT chk_consultants_status CHECK (status IS NULL OR status IN ('active', 'inactive', 'pending_approval', 'suspended'))
);
CREATE INDEX idx_consultants_specialty ON consultants(specialty);
CREATE INDEX idx_consultants_user_id ON consultants(user_id);

CREATE TABLE consultant_usage_logs (
    id uuid DEFAULT uuid_generate_v4() PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    consultant_id uuid NOT NULL REFERENCES consultants(id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    deleted_at timestamptz,
    idempotency_key text
);
CREATE UNIQUE INDEX idx_consultant_usage_idempotency ON consultant_usage_logs(user_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX idx_consultant_usage_logs_consultant_id ON consultant_usage_logs(consultant_id);
CREATE INDEX idx_consultant_usage_logs_user_id ON consultant_usage_logs(user_id);
