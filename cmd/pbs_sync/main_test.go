package main

import (
	"testing"
	"time"

	"dhis2gw/internal/config"
	"dhis2gw/internal/models"
)

func TestResolveFiscalYearAutomaticallyUsesJulyToJuneCycle(t *testing.T) {
	cfg := config.Config{}
	tests := []struct {
		name string
		now  time.Time
		want string
	}{
		{name: "july", now: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC), want: "2026-2027"},
		{name: "december", now: time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC), want: "2026-2027"},
		{name: "january", now: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), want: "2025-2026"},
		{name: "june", now: time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC), want: "2025-2026"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveFiscalYear(cfg, tt.now)
			if err != nil {
				t.Fatalf("resolveFiscalYear() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveFiscalYear() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveFiscalYearOverride(t *testing.T) {
	cfg := config.Config{}
	cfg.PBS.FiscalYearOverride = true
	cfg.PBS.FiscalYear = " 2024-2025 "

	got, err := resolveFiscalYear(cfg, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("resolveFiscalYear() error = %v", err)
	}
	if got != "2024-2025" {
		t.Fatalf("resolveFiscalYear() = %q, want %q", got, "2024-2025")
	}
}

func TestResolveFiscalYearOverrideRequiresValue(t *testing.T) {
	cfg := config.Config{}
	cfg.PBS.FiscalYearOverride = true

	if _, err := resolveFiscalYear(cfg, time.Now()); err == nil {
		t.Fatal("resolveFiscalYear() error = nil, want missing override value error")
	}
}

func TestCleanPiapProjectionValue(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		valueType string
		want      string
	}{
		{name: "number percentage", value: "50%", valueType: "NUMBER", want: "50"},
		{name: "number percentage with space", value: "50 % ", valueType: " number ", want: "50"},
		{name: "number without percentage", value: "50", valueType: "NUMBER", want: "50"},
		{name: "number strips leading zero", value: "01", valueType: "NUMBER", want: "1"},
		{name: "number strips leading zeros from zero", value: "00", valueType: "NUMBER", want: "0"},
		{name: "number strips multiple leading zeros", value: "007", valueType: "NUMBER", want: "7"},
		{name: "number preserves decimal digits", value: "005.50", valueType: "NUMBER", want: "5.50"},
		{name: "number preserves sign and exponent", value: "-0007e+2", valueType: "NUMBER", want: "-7e+2"},
		{name: "number percentage strips leading zeros", value: "007%", valueType: "NUMBER", want: "7"},
		{name: "number malformed value unchanged", value: "01abc", valueType: "NUMBER", want: "01abc"},
		{name: "non number preserves percentage", value: "50%", valueType: "TEXT", want: "50%"},
		{name: "non number preserves leading zero", value: "007", valueType: "TEXT", want: "007"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanPiapProjectionValue(tt.value, tt.valueType); got != tt.want {
				t.Fatalf("cleanPiapProjectionValue(%q, %q) = %q, want %q", tt.value, tt.valueType, got, tt.want)
			}
		})
	}
}

func TestAppendDVAttachesCommentToValue(t *testing.T) {
	deMapping := &models.Dhis2Mapping{DataElement: "de-uid"}
	combo := config.DHIS2CategoryOptionCombo{
		UID:    "coc-uid",
		Option: "co-uid",
		Combo:  "cc-uid",
	}
	getComboUID := func(string) config.DHIS2CategoryOptionCombo { return combo }

	var dataValues []ExtendedDataValue
	appendDV("actual", "2025Q3", "12", "Reason for variation", &dataValues, deMapping, "ou-uid", getComboUID)

	if len(dataValues) != 1 {
		t.Fatalf("got %d data values, want 1", len(dataValues))
	}
	dataValue := dataValues[0]
	if dataValue.Value == nil || *dataValue.Value != "12" {
		t.Fatalf("value = %v, want 12", dataValue.Value)
	}
	if dataValue.Comment == nil || *dataValue.Comment != "Reason for variation" {
		t.Fatalf("comment = %v, want reason for variation", dataValue.Comment)
	}
}
