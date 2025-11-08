ALTER TABLE dhis2_mappings ADD IF NOT EXISTS what TEXT DEFAULT '';
ALTER TABLE dhis2_mappings ADD IF NOT EXISTS source_orgunit TEXT DEFAULT '';
ALTER TABLE dhis2_mappings ADD IF NOT EXISTS dest_orgunit TEXT DEFAULT '';
ALTER TABLE dhis2_mappings ADD IF NOT EXISTS category_option TEXT DEFAULT '';
ALTER TABLE dhis2_mappings ADD IF NOT EXISTS category_combo TEXT DEFAULT '';

CREATE INDEX dhis2_mappings_what_idx ON dhis2_mappings(what);
CREATE INDEX dhis2_mappings_source_orgunit_idx ON dhis2_mappings(source_orgunit);
CREATE INDEX dhis2_mappings_dest_orgunit_idx ON dhis2_mappings(dest_orgunit);