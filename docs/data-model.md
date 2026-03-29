# Data Model

## Overview

ndc-loader ingests two FDA datasets into 7 PostgreSQL tables plus a checkpoint tracking table.

## NDC Directory Tables

### products

The core table. One row per unique drug product.

| Column | Type | Description |
|--------|------|-------------|
| product_id | TEXT (PK) | FDA product identifier |
| product_ndc | TEXT | 2-segment NDC (labeler-product) |
| product_type | TEXT | HUMAN PRESCRIPTION DRUG, OTC, etc. |
| proprietary_name | TEXT | Brand name |
| proprietary_name_suffix | TEXT | Brand name suffix |
| nonproprietary_name | TEXT | Generic name |
| dosage_form | TEXT | TABLET, CAPSULE, INJECTION, etc. |
| route | TEXT | ORAL, TOPICAL, etc. (semicolon-delimited for multiple) |
| labeler_name | TEXT | Manufacturer / labeler |
| substance_name | TEXT | Active ingredients (semicolon-delimited) |
| strength | TEXT | Strength values (semicolon-delimited) |
| strength_unit | TEXT | Strength units (semicolon-delimited) |
| pharm_classes | TEXT | Pharmacological classes with type codes (e.g., `Biguanide [EPC]`) |
| dea_schedule | TEXT | DEA schedule (CII-CV) |
| marketing_category | TEXT | NDA, ANDA, BLA, OTC monograph |
| application_number | TEXT | FDA application number (e.g., ANDA076543) |
| marketing_start | DATE | Start of marketing |
| marketing_end | DATE | End of marketing (null if still active) |
| ndc_exclude | BOOLEAN | Excluded from NDC directory |
| listing_certified | DATE | Certification date |
| search_vector | TSVECTOR | Full-text search index (auto-populated via trigger) |

**Indexes:** `product_ndc`, `application_number`, GIN on `search_vector`

### packages

One row per package configuration of a product.

| Column | Type | Description |
|--------|------|-------------|
| id | SERIAL (PK) | Auto-increment |
| product_id | TEXT | References products.product_id |
| product_ndc | TEXT | Parent product NDC |
| ndc_package_code | TEXT | Full 3-segment package NDC |
| description | TEXT | Package description (e.g., "100 TABLET in 1 BOTTLE") |
| marketing_start | DATE | Package marketing start |
| marketing_end | DATE | Package marketing end |
| ndc_exclude | BOOLEAN | Excluded flag |
| sample_package | BOOLEAN | Sample package flag |

**Indexes:** `product_id`, `ndc_package_code`, `product_ndc`

## Drugs@FDA Tables

### applications

One row per FDA application (NDA, ANDA, BLA).

| Column | Type | Description |
|--------|------|-------------|
| appl_no | TEXT (PK) | Application number (zero-padded, e.g., 076543) |
| appl_type | TEXT | NDA, ANDA, BLA |
| appl_public_notes | TEXT | Public notes |
| sponsor_name | TEXT | Sponsor company |

### drugsfda_products

Products within an FDA application.

| Column | Type | Description |
|--------|------|-------------|
| id | SERIAL (PK) | Auto-increment |
| appl_no | TEXT | References applications.appl_no |
| product_no | TEXT | Product number within application |
| form | TEXT | Dosage form |
| strength | TEXT | Strength |
| reference_drug | TEXT | Reference drug flag |
| drug_name | TEXT | Drug name |
| active_ingredient | TEXT | Active ingredient |
| reference_standard | TEXT | Reference standard flag |

### submissions, marketing_status, te_codes

See `migrations/001_initial_schema.sql` for complete definitions.

## Joining Across Datasets

NDC Directory and Drugs@FDA use different formats for the application number:

| Dataset | Field | Example |
|---------|-------|---------|
| NDC Directory | `products.application_number` | `ANDA076543` |
| Drugs@FDA | `applications.appl_no` | `076543` |

**Join query:**
```sql
SELECT p.*, a.sponsor_name, a.appl_type
FROM products p
JOIN applications a
  ON LPAD(regexp_replace(p.application_number, '^[A-Za-z]+', ''), 6, '0') = a.appl_no
WHERE p.application_number IS NOT NULL
  AND p.application_number ~ '^[A-Za-z]';
```

~59,000 products match to their FDA applications via this normalized join.

## Pharmacological Classification

The `pharm_classes` column contains a comma-delimited string with bracketed type codes:

```
GLP-1 Receptor Agonist [EPC], Glucagon-Like Peptide 1 [CS], Glucagon-like Peptide-1 (GLP-1) Agonists [MoA]
```

| Code | Meaning | Count in DB |
|------|---------|------------|
| EPC | Established Pharmacologic Class | ~99K |
| MoA | Mechanism of Action | ~62K |
| PE | Physiologic Effect | ~61K |
| CS | Chemical/Ingredient Structure | ~52K |

The internal API returns a `pharm_classes_structured` field that parses this into typed arrays.

## Load Checkpoint Tracking

### load_checkpoints

Tracks progress of each data load operation for resume-from-failure.

| Column | Type | Description |
|--------|------|-------------|
| id | SERIAL (PK) | Auto-increment |
| load_id | TEXT | UUID for the load execution |
| dataset | TEXT | Dataset name (ndc_directory, drugsfda) |
| table_name | TEXT | Target table |
| status | TEXT | pending, downloading, downloaded, loading, loaded, failed |
| row_count | INTEGER | Rows loaded |
| previous_row_count | INTEGER | Previous load count (for safety valve) |
| error_message | TEXT | Error details if failed |
| started_at | TIMESTAMPTZ | Step start |
| completed_at | TIMESTAMPTZ | Step completion |
| created_at | TIMESTAMPTZ | Record creation |

## Data Freshness

The FDA publishes updated NDC data daily. ndc-loader refreshes at 3am (configurable via `LOAD_SCHEDULE`). The `/health` endpoint reports `data_age_hours` so monitoring can alert on stale data (>48h = degraded).
