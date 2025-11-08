DROP INDEX IF EXISTS dhis2_mappings_what_idx;
DROP INDEX IF EXISTS dhis2_mappings_source_orgunit_idx;
DROP INDEX IF EXISTS dhis2_mappings_dest_orgunit_idx;

ALTER TABLE  dhis2_mappings DROP COLUMN IF EXISTS what;
ALTER TABLE  dhis2_mappings DROP COLUMN IF EXISTS source_orgunit;
ALTER TABLE  dhis2_mappings DROP COLUMN IF EXISTS dest_orgunit;
ALTER TABLE  dhis2_mappings DROP COLUMN IF EXISTS category_option;
ALTER TABLE  dhis2_mappings DROP COLUMN IF EXISTS category_combo;