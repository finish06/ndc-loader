-- 001_initial_schema.sql
-- Creates all tables for NDC Directory and Drugs@FDA datasets.

-- NDC Directory: Products
CREATE TABLE IF NOT EXISTS products (
    product_id          TEXT PRIMARY KEY,
    product_ndc         TEXT NOT NULL,
    product_type        TEXT,
    proprietary_name    TEXT,
    nonproprietary_name TEXT,
    dosage_form         TEXT,
    route               TEXT,
    labeler_name        TEXT,
    substance_name      TEXT,
    strength            TEXT,
    strength_unit       TEXT,
    pharm_classes       TEXT,
    dea_schedule        TEXT,
    marketing_category  TEXT,
    application_number  TEXT,
    marketing_start     DATE,
    marketing_end       DATE,
    ndc_exclude         BOOLEAN DEFAULT FALSE,
    listing_certified   DATE,
    search_vector       TSVECTOR
);

-- NDC Directory: Packages
CREATE TABLE IF NOT EXISTS packages (
    id                  SERIAL PRIMARY KEY,
    product_id          TEXT NOT NULL REFERENCES products(product_id) ON DELETE CASCADE,
    product_ndc         TEXT NOT NULL,
    ndc_package_code    TEXT NOT NULL,
    description         TEXT,
    marketing_start     DATE,
    marketing_end       DATE,
    ndc_exclude         BOOLEAN DEFAULT FALSE,
    sample_package      BOOLEAN DEFAULT FALSE
);

-- Drugs@FDA: Applications
CREATE TABLE IF NOT EXISTS applications (
    appl_no             TEXT PRIMARY KEY,
    appl_type           TEXT,
    sponsor_name        TEXT,
    most_recent_submission DATE
);

-- Drugs@FDA: Products
CREATE TABLE IF NOT EXISTS drugsfda_products (
    id                  SERIAL PRIMARY KEY,
    appl_no             TEXT NOT NULL REFERENCES applications(appl_no) ON DELETE CASCADE,
    product_no          TEXT NOT NULL,
    form                TEXT,
    strength            TEXT,
    reference_drug      TEXT,
    drug_name           TEXT,
    active_ingredient   TEXT,
    reference_standard  TEXT
);

-- Drugs@FDA: Submissions
CREATE TABLE IF NOT EXISTS submissions (
    id                              SERIAL PRIMARY KEY,
    appl_no                         TEXT NOT NULL REFERENCES applications(appl_no) ON DELETE CASCADE,
    submission_type                 TEXT,
    submission_no                   TEXT,
    submission_status               TEXT,
    submission_status_date          DATE,
    submission_class_code           TEXT,
    submission_class_code_description TEXT
);

-- Drugs@FDA: Marketing Status
CREATE TABLE IF NOT EXISTS marketing_status (
    id                  SERIAL PRIMARY KEY,
    appl_no             TEXT NOT NULL REFERENCES applications(appl_no) ON DELETE CASCADE,
    product_no          TEXT,
    marketing_status_id TEXT,
    marketing_status    TEXT
);

-- Drugs@FDA: Active Ingredients
CREATE TABLE IF NOT EXISTS active_ingredients (
    id              SERIAL PRIMARY KEY,
    appl_no         TEXT NOT NULL REFERENCES applications(appl_no) ON DELETE CASCADE,
    product_no      TEXT,
    ingredient_name TEXT,
    strength        TEXT
);

-- Drugs@FDA: TE Codes
CREATE TABLE IF NOT EXISTS te_codes (
    id          SERIAL PRIMARY KEY,
    appl_no     TEXT NOT NULL REFERENCES applications(appl_no) ON DELETE CASCADE,
    product_no  TEXT,
    te_code     TEXT
);

-- Load Checkpoints
CREATE TABLE IF NOT EXISTS load_checkpoints (
    id                  SERIAL PRIMARY KEY,
    load_id             TEXT NOT NULL,
    dataset             TEXT NOT NULL,
    table_name          TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending',
    row_count           INTEGER,
    previous_row_count  INTEGER,
    error_message       TEXT,
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes: NDC Directory
CREATE INDEX IF NOT EXISTS idx_products_ndc ON products(product_ndc);
CREATE INDEX IF NOT EXISTS idx_products_appno ON products(application_number);
CREATE INDEX IF NOT EXISTS idx_products_search ON products USING GIN(search_vector);
CREATE INDEX IF NOT EXISTS idx_packages_product ON packages(product_id);
CREATE INDEX IF NOT EXISTS idx_packages_ndc ON packages(ndc_package_code);
CREATE INDEX IF NOT EXISTS idx_packages_product_ndc ON packages(product_ndc);

-- Indexes: Drugs@FDA
CREATE INDEX IF NOT EXISTS idx_drugsfda_products_appno ON drugsfda_products(appl_no);
CREATE INDEX IF NOT EXISTS idx_submissions_appno ON submissions(appl_no);
CREATE INDEX IF NOT EXISTS idx_marketing_status_appno ON marketing_status(appl_no);
CREATE INDEX IF NOT EXISTS idx_active_ingredients_appno ON active_ingredients(appl_no);
CREATE INDEX IF NOT EXISTS idx_te_codes_appno ON te_codes(appl_no);

-- Indexes: Load Checkpoints
CREATE INDEX IF NOT EXISTS idx_checkpoints_load ON load_checkpoints(load_id);
CREATE INDEX IF NOT EXISTS idx_checkpoints_dataset ON load_checkpoints(dataset, status);

-- Trigger to auto-update search_vector on products insert/update
CREATE OR REPLACE FUNCTION products_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := to_tsvector('english',
        coalesce(NEW.proprietary_name, '') || ' ' ||
        coalesce(NEW.nonproprietary_name, '') || ' ' ||
        coalesce(NEW.labeler_name, '') || ' ' ||
        coalesce(NEW.substance_name, '')
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS products_search_vector_trigger ON products;
CREATE TRIGGER products_search_vector_trigger
    BEFORE INSERT OR UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION products_search_vector_update();
