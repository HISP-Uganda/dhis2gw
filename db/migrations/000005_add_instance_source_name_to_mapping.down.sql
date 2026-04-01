
ALTER TABLE  dhis2_mappings DROP COLUMN IF EXISTS instance_name;
ALTER TABLE  dhis2_mappings DROP COLUMN IF EXISTS source_name;

DROP INDEX IF EXISTS dhis2_mappings_what_idx;
DROP INDEX IF EXISTS dhis2_mappings_source_orgunit_idx;
DROP INDEX IF EXISTS dhis2_mappings_dest_orgunit_idx;

