package main

import (
	"testing"

	"dhis2gw/internal/config"
	"dhis2gw/internal/models"
)

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
