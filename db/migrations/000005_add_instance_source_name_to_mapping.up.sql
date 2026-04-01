ALTER TABLE dhis2_mappings ADD IF NOT EXISTS instance_name TEXT;
ALTER TABLE dhis2_mappings ADD IF NOT EXISTS source_name TEXT;
UPDATE dhis2_mappings SET instance_name = 'default', source_name = 'default';
ALTER TABLE dhis2_mappings ALTER COLUMN instance_name SET NOT NULL;
ALTER TABLE dhis2_mappings ALTER COLUMN source_name SET NOT NULL;

