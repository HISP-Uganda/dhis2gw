ALTER TABLE dhis2_mappings DROP IF EXISTS dhis2_name;
ALTER TABLE dhis2_mappings ADD COLUMN dimension_type TEXT DEFAULT 'none';

CREATE TABLE IF NOT EXISTS dhis2_mapping_dimension
(
    id                    BIGSERIAL PRIMARY KEY,
    mapping_id            BIGINT NOT NULL REFERENCES dhis2_mappings (id) ON DELETE CASCADE,
    source_field          TEXT   NOT NULL,                   -- e.g. "Release", "Male", "Urban"
    source_label          TEXT,                              -- optional display name from source
    category_option       TEXT   NOT NULL,                   -- DHIS2 UID for CategoryOption
    category_option_combo TEXT   NOT NULL,                   -- DHIS2 UID for CategoryOptionCombo
    dhis2_name            TEXT,                              -- DHIS2 name of the categoryOption or combo
    type                  TEXT        DEFAULT 'attribution', -- 'attribution', 'disaggregation', etc.
    dimension_group       TEXT,                              -- optional grouping e.g. 'financial', 'sex', 'age'
    created               TIMESTAMPTZ DEFAULT now(),
    updated               TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_dhis2_mapping_dim_mapping_id ON dhis2_mapping_dimension(mapping_id);
CREATE INDEX IF NOT EXISTS idx_dhis2_mapping_dim_type ON dhis2_mapping_dimension(type);
