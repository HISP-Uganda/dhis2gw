package main

import (
	"dhis2gw/utils"
	"errors"
	"strings"
	"time"

	"github.com/HISP-Uganda/go-dhis2-sdk/dhis2/schema"
	"github.com/HISP-Uganda/go-dhis2-sdk/tracker"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// ExtractValue retrieves a value from a JSON string using a GJSON path.
//
// Example:
//
//	val := extractValue(jsonStr, "project.details.name")
//	fmt.Println(val) // "Water Improvement"
func ExtractValue(jsonStr, path string) string {
	result := gjson.Get(jsonStr, path)
	if !result.Exists() {
		return ""
	}

	if result.IsArray() {
		// If it’s an array, join all elements with a comma
		arr := result.Array()
		values := make([]string, 0, len(arr))
		for _, v := range arr {
			values = append(values, v.String())
		}
		return strings.Join(values, ", ")
	}

	return result.String()
}

func ExtractValues(jsonStr string, mapping map[string]string) map[string]string {
	results := make(map[string]string)

	for uid, path := range mapping {
		v := gjson.Get(jsonStr, path)
		if !v.Exists() {
			log.Warnf("Path %s not found in JSON", path)
			continue
		}

		if v.IsArray() {
			arr := v.Array()
			if len(arr) == 0 {
				log.Warnf("Path %s is an empty array", path)
				continue
			}
			results[uid] = arr[0].String()
		} else {
			results[uid] = v.String()
		}
	}

	return results
}

func BuildTrackerPayload(
	jsonStr string,
	cfg *Config,
	orgUnit string,
) (tracker.NestedPayload, error) {
	// now := time.Now().Format("2006-01-02")
	now := time.Now()

	// --- Extract attributes ---
	validAttributes := utils.FilterValidUIDs(cfg.Program.TrackedEntityAttributes)
	validAttributesSlice := utils.FilterValidUIDsSlice(validAttributes)
	missingMandatoryAttrs := utils.MissingStrings(validAttributesSlice, cfg.MandatoryTrackedEntityAttributes)
	if len(missingMandatoryAttrs) > 0 {
		log.Warnf("Missing mandatory attributes: %v", missingMandatoryAttrs)
		return tracker.NestedPayload{}, errors.New("missing mandatory attributes")
	}
	attrs := ExtractValues(jsonStr, validAttributes)
	log.Debugf("Valid attributes ====>: %v", attrs)
	var attrList []tracker.TrackedEntityAttribute
	for uid, val := range attrs {
		attrList = append(attrList, tracker.TrackedEntityAttribute{Attribute: &uid, Value: &val})
	}

	// --- Build Events from stages ---
	var events []tracker.NestedEvent
	for _, stageCfg := range cfg.Program.Stages {
		validDataValues := utils.FilterValidUIDs(stageCfg.DataValues)
		dataVals := ExtractValues(jsonStr, validDataValues)
		var dataValueList []schema.TrackerDataValue
		for deUID, val := range dataVals {
			dataValueList = append(dataValueList, schema.TrackerDataValue{DataElement: &deUID, Value: &val})
		}

		events = append(events, tracker.NestedEvent{
			Program:      cfg.Program.ProgramID,
			ProgramStage: stageCfg.ProgramStage,
			OrgUnit:      orgUnit,
			OccurredAt:   now,
			DataValues:   dataValueList,
			Status:       "ACTIVE",
		})
	}

	tei := tracker.NestedTrackedEntity{
		TrackedEntityType: cfg.Program.TrackedEntityType,
		OrgUnit:           orgUnit,
		Attributes:        attrList,
		Enrollments: []tracker.NestedEnrollment{
			{
				Program:    cfg.Program.ProgramID,
				OrgUnit:    orgUnit,
				EnrolledAt: now,
				OccurredAt: now,
				Events:     events,
			},
		},
	}

	return tracker.NestedPayload{TrackedEntities: []tracker.NestedTrackedEntity{tei}}, nil
}
