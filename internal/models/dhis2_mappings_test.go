package models

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDhis2MappingCSVIncludesValueType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mappings.csv")
	if err := os.WriteFile(path, []byte("code,dataelement,value_type\nITEM_001,de-uid,INTEGER\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	mappings, err := ParseDhis2MappingCSV(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 1 {
		t.Fatalf("got %d mappings, want 1", len(mappings))
	}
	if mappings[0].ValueType != "INTEGER" {
		t.Fatalf("value type = %q, want INTEGER", mappings[0].ValueType)
	}
}

func TestGenerateDhis2MappingExcelIncludesValueType(t *testing.T) {
	mappings := []Dhis2Mapping{{Code: "ITEM_001", DataElement: "de-uid", ValueType: "INTEGER"}}
	file, err := GenerateDhis2MappingExcel(mappings)
	if err != nil {
		t.Fatal(err)
	}

	rows, err := file.GetRows("Mappings")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	valueTypeColumn := -1
	for i, header := range rows[0] {
		if header == "value_type" {
			valueTypeColumn = i
			break
		}
	}
	if valueTypeColumn < 0 {
		t.Fatal("value_type column not found")
	}
	if rows[1][valueTypeColumn] != "INTEGER" {
		t.Fatalf("value type = %q, want INTEGER", rows[1][valueTypeColumn])
	}

	path := filepath.Join(t.TempDir(), "mappings.xlsx")
	if err := file.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	input, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()

	parsed, err := ParseDhis2MappingExcel(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 1 {
		t.Fatalf("parsed %d mappings, want 1", len(parsed))
	}
	if parsed[0].ValueType != "INTEGER" {
		t.Fatalf("parsed value type = %q, want INTEGER", parsed[0].ValueType)
	}
}
