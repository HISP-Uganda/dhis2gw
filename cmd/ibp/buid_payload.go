package main

import (
	"dhis2gw/models"
	"dhis2gw/utils"
	"errors"
	"fmt"
	"regexp"
	"strconv"
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

func ExtractValues(jsonStr string, mapping map[string]string, conf *Config) map[string]string {
	results := make(map[string]string)

	for uid, path := range mapping {
		v := gjson.Get(jsonStr, path)
		if !v.Exists() {
			log.Warnf("Path %s not found in JSON: %s", path, jsonStr)
			if _, ok := conf.Defaults[uid]; ok {
				switch conf.Defaults[uid].(type) {
				case string:
					results[uid] = conf.Defaults[uid].(string)
				default:
					results[uid] = fmt.Sprintf("%v", conf.Defaults[uid])
				}
			}
			continue
		}

		if v.IsArray() {
			arr := v.Array()

			if strings.Contains(path, "#") {
				var items []string
				for _, item := range arr {
					items = append(items, item.String())
				}
				results[uid] = strings.Join(items, ", ")
				continue
			}

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

//func BuildTrackerPayload(
//	jsonStr string,
//	cfg *Config,
//	orgUnit string,
//) (tracker.LegacyNestedPayload, error) {
//	// now := time.Now().Format("2006-01-02")
//	now := tracker.DHIS2Time{
//		Time: time.Now(),
//	}
//	// teUID := utils.GenerateUID()
//	// enrollmentUID := utils.GenerateUID()
//
//	// --- Extract attributes ---
//	validAttributes := utils.FilterValidUIDs(cfg.Program.TrackedEntityAttributes)
//	// log.Infof("YYYY Valid Attributes Slice: %v", validAttributes)
//	validAttributesSlice := utils.FilterValidUIDsSlice(validAttributes)
//	// log.Infof("XXXXXX Valid Attributes Slice: %v", validAttributesSlice)
//	missingMandatoryAttrs := utils.SetDifference(validAttributesSlice, cfg.MandatoryTrackedEntityAttributes)
//	// log.Infof("Missing mandatory attributes: %v", missingMandatoryAttrs)
//	var (
//		missingWithDefaults    []string
//		missingWithoutDefaults []string
//	)
//	if len(missingMandatoryAttrs) > 0 {
//
//		for _, attr := range missingMandatoryAttrs {
//			if _, hasDefault := cfg.Defaults[attr]; hasDefault {
//				missingWithDefaults = append(missingWithDefaults, attr)
//			} else {
//				missingWithoutDefaults = append(missingWithoutDefaults, attr)
//			}
//		}
//		log.Warnf(
//			"Missing mandatory attributes without defaults: %v; with defaults (will be auto-filled): %v",
//			missingWithoutDefaults,
//			missingWithDefaults,
//		)
//		// Only block if there are mandatory attributes with no defaults
//		if len(missingWithoutDefaults) > 0 {
//			return tracker.LegacyNestedPayload{}, errors.New("missing mandatory attributes without defaults")
//		}
//	}
//	attrs := ExtractValues(jsonStr, validAttributes, cfg)
//	log.Debugf("Valid attributes ====>: %v", attrs)
//	var attrList []tracker.TrackedEntityAttribute
//	for uid, val := range attrs {
//		attrVal := NormalizeValue(uid, val)
//		attrList = append(attrList, tracker.TrackedEntityAttribute{Attribute: &uid, Value: &attrVal})
//	}
//	for _, k := range missingWithDefaults {
//		v := cfg.Defaults[k].(string)
//		attrList = append(attrList, tracker.TrackedEntityAttribute{Attribute: &k, Value: &v})
//	}
//
//	// --- Build Events from stages ---
//	var events []tracker.LegacyNestedEvent
//	for _, stageCfg := range cfg.Program.Stages {
//		validDataValues := utils.FilterValidUIDs(stageCfg.DataValues)
//		validDataValuesSlice := utils.FilterValidUIDsSlice(validDataValues)
//		missingMandatoryDataValues := utils.MissingStrings(validDataValuesSlice, cfg.MandatoryTrackedEntityAttributes)
//		if len(missingMandatoryDataValues) > 0 {
//			log.Warnf("Missing mandatory attributes: %v", missingMandatoryDataValues)
//		}
//		dataVals := ExtractValues(jsonStr, validDataValues, cfg)
//		var dataValueList []schema.DataValue
//		for deUID, val := range dataVals {
//			dataValueList = append(dataValueList, schema.DataValue{DataElement: &deUID, Value: &val})
//		}
//
//		if len(dataValueList) > 0 {
//			events = append(events, tracker.LegacyNestedEvent{
//				Program:      cfg.Program.ProgramID,
//				ProgramStage: stageCfg.ProgramStage,
//				OrgUnit:      orgUnit,
//				OccurredAt:   now,
//				DataValues:   dataValueList,
//				Status:       "ACTIVE",
//				// TrackedEntityInstance: teUID,
//			})
//		}
//
//	}
//
//	tei := tracker.LegacyNestedTrackedEntity{
//		TrackedEntityType: cfg.Program.TrackedEntityType,
//		// TrackedEntityInstance: teUID,
//		OrgUnit:    orgUnit,
//		Attributes: attrList,
//		Enrollments: []tracker.LegacyNestedEnrollment{
//			{
//				Program:    cfg.Program.ProgramID,
//				OrgUnit:    orgUnit,
//				EnrolledAt: now,
//				OccurredAt: now,
//				Events:     events,
//				Status:     "ACTIVE",
//				// TrackedEntityInstance: teUID,
//			},
//		},
//	}
//
//	return tracker.LegacyNestedPayload{TrackedEntities: []tracker.LegacyNestedTrackedEntity{tei}}, nil
//}

func BuildTrackerPayloadV2(
	jsonStr string,
	cfg *Config,
	orgUnit string,
) (tracker.NestedPayload, error) {
	// now := time.Now().Format("2006-01-02")
	now := tracker.DHIS2Time{
		Time: time.Now(),
	}
	// teUID := utils.GenerateUID()
	// enrollmentUID := utils.GenerateUID()

	// --- Extract attributes ---
	validAttributes := utils.FilterValidUIDs(cfg.Program.TrackedEntityAttributes)
	// log.Infof("YYYY Valid Attributes Slice: %v", validAttributes)
	validAttributesSlice := utils.FilterValidUIDsSlice(validAttributes)
	// log.Infof("XXXXXX Valid Attributes Slice: %v", validAttributesSlice)
	missingMandatoryAttrs := utils.SetDifference(validAttributesSlice, cfg.MandatoryTrackedEntityAttributes)
	// log.Infof("Missing mandatory attributes: %v", missingMandatoryAttrs)
	var (
		missingWithDefaults    []string
		missingWithoutDefaults []string
	)
	if len(missingMandatoryAttrs) > 0 {

		for _, attr := range missingMandatoryAttrs {
			if _, hasDefault := cfg.Defaults[attr]; hasDefault {
				missingWithDefaults = append(missingWithDefaults, attr)
			} else {
				missingWithoutDefaults = append(missingWithoutDefaults, attr)
			}
		}
		log.Warnf(
			"Missing mandatory attributes without defaults: %v; with defaults (will be auto-filled): %v",
			missingWithoutDefaults,
			missingWithDefaults,
		)
		// Only block if there are mandatory attributes with no defaults
		if len(missingWithoutDefaults) > 0 {
			return tracker.NestedPayload{}, errors.New("missing mandatory attributes without defaults")
		}
	}
	attrs := ExtractValues(jsonStr, validAttributes, cfg)
	log.Debugf("Valid attributes ====>: %v", attrs)
	var attrList []tracker.TrackedEntityAttribute
	for uid, val := range attrs {
		attrVal := NormalizeValue(uid, val)
		attrList = append(attrList, tracker.TrackedEntityAttribute{Attribute: &uid, Value: &attrVal})
	}
	for _, k := range missingWithDefaults {
		v := cfg.Defaults[k].(string)
		attrList = append(attrList, tracker.TrackedEntityAttribute{Attribute: &k, Value: &v})
	}

	// --- Build Events from stages ---
	var events []tracker.NestedEvent
	for _, stageCfg := range cfg.Program.Stages {
		validDataValues := utils.FilterValidUIDs(stageCfg.DataValues)
		validDataValuesSlice := utils.FilterValidUIDsSlice(validDataValues)
		missingMandatoryDataValues := utils.MissingStrings(validDataValuesSlice, cfg.MandatoryTrackedEntityAttributes)
		if len(missingMandatoryDataValues) > 0 {
			log.Warnf("Missing mandatory attributes: %v", missingMandatoryDataValues)
		}
		dataVals := ExtractValues(jsonStr, validDataValues, cfg)
		var dataValueList []schema.TrackerDataValue
		for deUID, val := range dataVals {
			dataValueList = append(dataValueList, schema.TrackerDataValue{DataElement: &deUID, Value: &val})
		}

		if len(dataValueList) > 0 {
			events = append(events, tracker.NestedEvent{
				Program:      cfg.Program.ProgramID,
				ProgramStage: stageCfg.ProgramStage,
				OrgUnit:      orgUnit,
				OccurredAt:   now,
				DataValues:   dataValueList,
				Status:       "ACTIVE",
				// TrackedEntityInstance: teUID,
			})
		}

	}

	tei := tracker.NestedTrackedEntity{
		TrackedEntityType: cfg.Program.TrackedEntityType,
		// TrackedEntityInstance: teUID,
		OrgUnit:    orgUnit,
		Attributes: attrList,
		Enrollments: []tracker.NestedEnrollment{
			{
				Program:    cfg.Program.ProgramID,
				OrgUnit:    orgUnit,
				EnrolledAt: now,
				OccurredAt: now,
				Events:     events,
				Status:     "ACTIVE",
				// TrackedEntityInstance: teUID,
			},
		},
	}

	return tracker.NestedPayload{TrackedEntities: []tracker.NestedTrackedEntity{tei}}, nil
}

//// ValidateTrackerPayload validates tracker.NestedPayload based on the models.Program set in cfg.ProgramConf
//func ValidateTrackerPayload(payload tracker.NestedPayload, config models.Program) (bool, map[string]string) {
//
//	return false, nil
//}

// ValidateTrackerPayload validates tracker.NestedPayload based on the models.Program set in cfg.ProgramConf
//func ValidateTrackerPayload(payload tracker.NestedPayload, config models.Program) (bool, map[string]string) {
//	// Basic shape checks
//	payloadErrors := map[string]string{}
//
//	// Collect mandatory TEAs and DEs from config
//	mandatoryTEAs := map[string]struct{}{}
//	for _, ptea := range config.ProgramTrackedEntityAttributes {
//		if ptea.Mandatory && ptea.TrackedEntityAttribute.ID != "" {
//			mandatoryTEAs[ptea.TrackedEntityAttribute.ID] = struct{}{}
//		}
//	}
//	// Map stageID -> set of mandatory DE IDs
//	mandatoryStageDEs := map[string]map[string]struct{}{}
//	for _, stage := range config.ProgramStages {
//		for _, psde := range stage.ProgramStageDataElements {
//			if psde.Compulsory && psde.DataElement.ID != "" {
//				if _, ok := mandatoryStageDEs[stage.ID]; !ok {
//					mandatoryStageDEs[stage.ID] = map[string]struct{}{}
//				}
//				mandatoryStageDEs[stage.ID][psde.DataElement.ID] = struct{}{}
//			}
//		}
//	}
//
//	// Validate attributes on tracked entity
//	// Expecting payload.TrackedEntityInstances[0].Attributes with attribute/value pairs
//	if len(payload.TrackedEntities) == 0 {
//		payloadErrors["teis"] = "missing tracked entities"
//	} else {
//		attrSeen := map[string]struct{}{}
//		if len(payload.TrackedEntities) > 0 {
//			for _, a := range payload.TrackedEntities[0].Attributes {
//				if *a.Attribute != "" {
//					attrSeen[*a.Attribute] = struct{}{}
//				}
//			}
//			// Find missing mandatory TEAs
//			for teaID := range mandatoryTEAs {
//				if _, ok := attrSeen[teaID]; !ok {
//					payloadErrors["tea."+teaID] = "mandatory attribute missing"
//				}
//			}
//		}
//
//	}
//
//	// Validate events data values per stage
//	// Expecting payload.Events with ProgramStage and DataValues [{dataElement,value}]
//	stageToElements := map[string]map[string]struct{}{}
//	for _, en := range payload.TrackedEntities {
//		if len(en.Enrollments) > 0 {
//			for _, m := range en.Enrollments {
//				for _, ev := range m.Events {
//					if ev.ProgramStage == "" {
//						continue
//					}
//					if _, ok := stageToElements[ev.ProgramStage]; !ok {
//						stageToElements[ev.ProgramStage] = map[string]struct{}{}
//					}
//					for _, dv := range ev.DataValues {
//						if *dv.DataElement != "" {
//							stageToElements[ev.ProgramStage][*dv.DataElement] = struct{}{}
//						}
//					}
//				}
//			}
//
//		}
//
//	}
//
//	for stageID, req := range mandatoryStageDEs {
//		seen := stageToElements[stageID]
//		for deID := range req {
//			if seen == nil {
//				payloadErrors["stage."+stageID] = "mandatory data elements missing for stage"
//				// still check specifics to list all DEs
//			}
//			if _, ok := seen[deID]; !ok {
//				payloadErrors["de."+stageID+"."+deID] = "mandatory data element missing"
//			}
//		}
//	}
//
//	return len(payloadErrors) == 0, payloadErrors
//}

// ValidateTrackerPayload2 validates tracker.NestedPayload based on the models.Program set in cfg.ProgramConf
func ValidateTrackerPayload2(payload tracker.LegacyNestedPayload, config models.Program) (bool, map[string]string) {
	// Basic presence and rules validation
	errs := map[string]string{}

	// --- Build lookup maps from config for quick validation ---
	// TEA rules: id -> {valueType, optionSet options(set)}
	type teaRule struct {
		valueType    string
		optionSetIDs map[string]struct{}
		hasOptionSet bool
	}
	teaRules := map[string]teaRule{}
	for _, ptea := range config.ProgramTrackedEntityAttributes {
		r := teaRule{
			valueType:    ptea.ValueType,
			optionSetIDs: map[string]struct{}{},
			hasOptionSet: ptea.TrackedEntityAttribute.OptionSetValue || len(ptea.TrackedEntityAttribute.OptionSet.Options) > 0 || ptea.TrackedEntityAttribute.OptionSet.ID != "",
		}
		for _, opt := range ptea.TrackedEntityAttribute.OptionSet.Options {
			// Use both id and name as acceptable (IDs are canonical; names commonly used in payloads)
			if opt.ID != "" {
				r.optionSetIDs[opt.ID] = struct{}{}
			}
			if opt.Code != "" {
				r.optionSetIDs[opt.Code] = struct{}{}
			}
			if opt.Name != "" && opt.ID != "" {
				// allow "name (id)" style is not necessary; keeping minimal
			}
		}
		teaRules[ptea.TrackedEntityAttribute.ID] = r
	}

	// Stage DE rules: stageID -> deID -> {valueType, optionSet options(set)}
	type deRule struct {
		valueType    string
		optionSetIDs map[string]struct{}
		hasOptionSet bool
		mandatory    bool
	}
	stageRules := map[string]map[string]deRule{}
	for _, st := range config.ProgramStages {
		if _, ok := stageRules[st.ID]; !ok {
			stageRules[st.ID] = map[string]deRule{}
		}
		for _, psde := range st.ProgramStageDataElements {
			r := deRule{
				valueType:    psde.DataElement.ValueType,
				optionSetIDs: map[string]struct{}{},
				hasOptionSet: psde.DataElement.OptionSetValue || len(psde.DataElement.OptionSet.Options) > 0 || psde.DataElement.OptionSet.ID != "",
				mandatory:    psde.Compulsory,
			}
			for _, opt := range psde.DataElement.OptionSet.Options {
				if opt.ID != "" {
					r.optionSetIDs[opt.ID] = struct{}{}
				}
				if opt.Name != "" {
					r.optionSetIDs[opt.Name] = struct{}{}
				}
			}
			stageRules[st.ID][psde.DataElement.ID] = r
		}
	}

	// --- Presence checks already implemented in your version (kept minimal here) ---
	// Check TEA presence for the first tracked entity (extend if multiple needed)
	if len(payload.TrackedEntities) == 0 {
		errs["teis"] = "missing tracked entities"
	} else {
		// presence
		seen := map[string]struct{}{}
		for _, a := range payload.TrackedEntities[0].Attributes {
			if a.Attribute != nil && *a.Attribute != "" {
				seen[*a.Attribute] = struct{}{}
			}
		}
		// If you have a list of mandatory TEAs (e.g., cfg.MandatoryTrackedEntityAttributes),
		// verify presence here. Otherwise, skip presence-only checks.

		// Example presence validation using rules derived from config (when mandatory flag is required):
		for _, ptea := range config.ProgramTrackedEntityAttributes {
			if ptea.Mandatory {
				id := ptea.TrackedEntityAttribute.ID
				if _, ok := seen[id]; !ok {
					errs["tea."+id] = "mandatory attribute missing"
				}
			}
		}
	}

	// --- Value validation helpers ---
	isEmpty := func(s string) bool { return len(strings.TrimSpace(s)) == 0 }

	validateValueType := func(valueType string, val string) bool {
		if isEmpty(val) {
			return false
		}
		switch strings.ToLower(valueType) {
		case "text", "long_text", "letter", "email", "url", "username", "file_resource", "image":
			return true
		case "number", "integer", "integer_positive", "integer_zero_or_positive", "percent":
			_, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return false
			}
			if strings.Contains(valueType, "integer") {
				if !regexp.MustCompile(`^-?\d+$`).MatchString(val) {
					return false
				}
				n, _ := strconv.ParseInt(val, 10, 64)
				switch strings.ToLower(valueType) {
				case "integer_positive":
					return n > 0
				case "integer_zero_or_positive":
					return n >= 0
				default:
					return true
				}
			}
			return true
		case "boolean", "true_only":
			v := strings.ToLower(val)
			if v == "true" || v == "false" || v == "1" || v == "0" || v == "yes" || v == "no" {
				if valueType == "true_only" {
					// true_only accepts true/1/yes only
					return v == "true" || v == "1" || v == "yes"
				}
				return true
			}
			return false
		case "date":
			_, err := time.Parse("2006-01-02", val)
			return err == nil
		case "datetime":
			// basic ISO-like acceptance
			_, err := time.Parse(time.RFC3339, val)
			return err == nil
		case "time":
			_, err := time.Parse("15:04", val)
			if err == nil {
				return true
			}
			_, err = time.Parse("15:04:05", val)
			return err == nil
		case "coordinate":
			// Expect "[lon,lat]" or "lon lat"
			v := strings.TrimSpace(val)
			if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
				v = strings.Trim(v, "[]")
			}
			parts := strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' })
			if len(parts) != 2 {
				return false
			}
			lon, err1 := strconv.ParseFloat(parts[0], 64)
			lat, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 != nil || err2 != nil {
				return false
			}
			return lon >= -180 && lon <= 180 && lat >= -90 && lat <= 90
		default:
			// Unknown types treated as text
			return true
		}
	}

	// --- TEA value validation ---
	if len(payload.TrackedEntities) > 0 {
		for _, a := range payload.TrackedEntities[0].Attributes {
			if a.Attribute == nil {
				continue
			}
			id := *a.Attribute
			r, ok := teaRules[id]
			if !ok {
				continue
			}
			val := ""
			if a.Value != nil {
				val = fmt.Sprintf("%v", *a.Value)
			}
			// OptionSet validation
			if r.hasOptionSet {
				if _, ok := r.optionSetIDs[val]; !ok {
					errs["tea.value."+id] = fmt.Sprintf("invalid option value: %v", val)
					log.Infof("OptionSetIDs is : %v", r.optionSetIDs)
					continue
				}
			}
			// ValueType validation
			if r.valueType != "" && !validateValueType(r.valueType, val) {
				errs["tea.type."+id] = fmt.Sprintf("invalid value: %v for type "+r.valueType, val)
			}
		}
	}

	// --- Event DE presence and value validation per stage ---
	for _, tei := range payload.TrackedEntities {
		for _, enr := range tei.Enrollments {
			for _, ev := range enr.Events {
				stageID := ev.ProgramStage
				if stageID == "" {
					continue
				}
				deMap := stageRules[stageID]
				if len(deMap) == 0 {
					continue
				}
				// seen for presence
				seen := map[string]struct{}{}
				for _, dv := range ev.DataValues {
					if dv.DataElement == nil {
						continue
					}
					deID := *dv.DataElement
					seen[deID] = struct{}{}
					r, ok := deMap[deID]
					if !ok {
						continue
					}
					val := ""
					if dv.Value != nil {
						val = fmt.Sprintf("%v", *dv.Value)
					}
					// OptionSet validation
					if r.hasOptionSet {
						if _, ok := r.optionSetIDs[val]; !ok {
							errs["de.value."+stageID+"."+deID] = "invalid option value"
							continue
						}
					}
					// ValueType validation
					if r.valueType != "" && !validateValueType(r.valueType, val) {
						errs["de.type."+stageID+"."+deID] = "invalid value for type " + r.valueType
					}
				}
				// Mandatory presence for this stage
				for deID, r := range deMap {
					if r.mandatory {
						if _, ok := seen[deID]; !ok {
							errs["de.missing."+stageID+"."+deID] = "mandatory data element missing"
						}
					}
				}
			}
		}
	}

	return len(errs) == 0, errs
}

func ValidateTrackerPayload(payload tracker.NestedPayload, config models.Program) (bool, map[string]string) {
	// Basic presence and rules validation
	errs := map[string]string{}

	// --- Build lookup maps from config for quick validation ---
	// TEA rules: id -> {valueType, optionSet options(set)}
	type teaRule struct {
		valueType    string
		optionSetIDs map[string]struct{}
		hasOptionSet bool
	}
	teaRules := map[string]teaRule{}
	for _, ptea := range config.ProgramTrackedEntityAttributes {
		r := teaRule{
			valueType:    ptea.ValueType,
			optionSetIDs: map[string]struct{}{},
			hasOptionSet: ptea.TrackedEntityAttribute.OptionSetValue || len(ptea.TrackedEntityAttribute.OptionSet.Options) > 0 || ptea.TrackedEntityAttribute.OptionSet.ID != "",
		}
		for _, opt := range ptea.TrackedEntityAttribute.OptionSet.Options {
			// Use both id and name as acceptable (IDs are canonical; names commonly used in payloads)
			if opt.ID != "" {
				r.optionSetIDs[opt.ID] = struct{}{}
			}
			if opt.Code != "" {
				r.optionSetIDs[opt.Code] = struct{}{}
			}
			if opt.Name != "" && opt.ID != "" {
				// allow "name (id)" style is not necessary; keeping minimal
			}
		}
		teaRules[ptea.TrackedEntityAttribute.ID] = r
	}

	// Stage DE rules: stageID -> deID -> {valueType, optionSet options(set)}
	type deRule struct {
		valueType    string
		optionSetIDs map[string]struct{}
		hasOptionSet bool
		mandatory    bool
	}
	stageRules := map[string]map[string]deRule{}
	for _, st := range config.ProgramStages {
		if _, ok := stageRules[st.ID]; !ok {
			stageRules[st.ID] = map[string]deRule{}
		}
		for _, psde := range st.ProgramStageDataElements {
			r := deRule{
				valueType:    psde.DataElement.ValueType,
				optionSetIDs: map[string]struct{}{},
				hasOptionSet: psde.DataElement.OptionSetValue || len(psde.DataElement.OptionSet.Options) > 0 || psde.DataElement.OptionSet.ID != "",
				mandatory:    psde.Compulsory,
			}
			for _, opt := range psde.DataElement.OptionSet.Options {
				if opt.ID != "" {
					r.optionSetIDs[opt.ID] = struct{}{}
				}
				if opt.Name != "" {
					r.optionSetIDs[opt.Name] = struct{}{}
				}
			}
			stageRules[st.ID][psde.DataElement.ID] = r
		}
	}

	// --- Presence checks already implemented in your version (kept minimal here) ---
	// Check TEA presence for the first tracked entity (extend if multiple needed)
	if len(payload.TrackedEntities) == 0 {
		errs["teis"] = "missing tracked entities"
	} else {
		// presence
		seen := map[string]struct{}{}
		for _, a := range payload.TrackedEntities[0].Attributes {
			if a.Attribute != nil && *a.Attribute != "" {
				seen[*a.Attribute] = struct{}{}
			}
		}
		// If you have a list of mandatory TEAs (e.g., cfg.MandatoryTrackedEntityAttributes),
		// verify presence here. Otherwise, skip presence-only checks.

		// Example presence validation using rules derived from config (when mandatory flag is required):
		for _, ptea := range config.ProgramTrackedEntityAttributes {
			if ptea.Mandatory {
				id := ptea.TrackedEntityAttribute.ID
				if _, ok := seen[id]; !ok {
					errs["tea."+id] = "mandatory attribute missing"
				}
			}
		}
	}

	// --- Value validation helpers ---
	isEmpty := func(s string) bool { return len(strings.TrimSpace(s)) == 0 }

	validateValueType := func(valueType string, val string) bool {
		if isEmpty(val) {
			return false
		}
		switch strings.ToLower(valueType) {
		case "text", "long_text", "letter", "email", "url", "username", "file_resource", "image":
			return true
		case "number", "integer", "integer_positive", "integer_zero_or_positive", "percent":
			_, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return false
			}
			if strings.Contains(valueType, "integer") {
				if !regexp.MustCompile(`^-?\d+$`).MatchString(val) {
					return false
				}
				n, _ := strconv.ParseInt(val, 10, 64)
				switch strings.ToLower(valueType) {
				case "integer_positive":
					return n > 0
				case "integer_zero_or_positive":
					return n >= 0
				default:
					return true
				}
			}
			return true
		case "boolean", "true_only":
			v := strings.ToLower(val)
			if v == "true" || v == "false" || v == "1" || v == "0" || v == "yes" || v == "no" {
				if valueType == "true_only" {
					// true_only accepts true/1/yes only
					return v == "true" || v == "1" || v == "yes"
				}
				return true
			}
			return false
		case "date":
			_, err := time.Parse("2006-01-02", val)
			return err == nil
		case "datetime":
			// basic ISO-like acceptance
			_, err := time.Parse(time.RFC3339, val)
			return err == nil
		case "time":
			_, err := time.Parse("15:04", val)
			if err == nil {
				return true
			}
			_, err = time.Parse("15:04:05", val)
			return err == nil
		case "coordinate":
			// Expect "[lon,lat]" or "lon lat"
			v := strings.TrimSpace(val)
			if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
				v = strings.Trim(v, "[]")
			}
			parts := strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' })
			if len(parts) != 2 {
				return false
			}
			lon, err1 := strconv.ParseFloat(parts[0], 64)
			lat, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 != nil || err2 != nil {
				return false
			}
			return lon >= -180 && lon <= 180 && lat >= -90 && lat <= 90
		default:
			// Unknown types treated as text
			return true
		}
	}

	// --- TEA value validation ---
	if len(payload.TrackedEntities) > 0 {
		for _, a := range payload.TrackedEntities[0].Attributes {
			if a.Attribute == nil {
				continue
			}
			id := *a.Attribute
			r, ok := teaRules[id]
			if !ok {
				continue
			}
			val := ""
			if a.Value != nil {
				val = fmt.Sprintf("%v", *a.Value)
			}
			// OptionSet validation
			if r.hasOptionSet {
				if _, ok := r.optionSetIDs[val]; !ok {
					errs["tea.value."+id] = fmt.Sprintf("invalid option value: %v", val)
					log.Infof("OptionSetIDs is : %v", r.optionSetIDs)
					continue
				}
			}
			// ValueType validation
			if r.valueType != "" && !validateValueType(r.valueType, val) {
				errs["tea.type."+id] = fmt.Sprintf("invalid value: %v for type "+r.valueType, val)
			}
		}
	}

	// --- Event DE presence and value validation per stage ---
	for _, tei := range payload.TrackedEntities {
		for _, enr := range tei.Enrollments {
			for _, ev := range enr.Events {
				stageID := ev.ProgramStage
				if stageID == "" {
					continue
				}
				deMap := stageRules[stageID]
				if len(deMap) == 0 {
					continue
				}
				// seen for presence
				seen := map[string]struct{}{}
				for _, dv := range ev.DataValues {
					if dv.DataElement == nil {
						continue
					}
					deID := *dv.DataElement
					seen[deID] = struct{}{}
					r, ok := deMap[deID]
					if !ok {
						continue
					}
					val := ""
					if dv.Value != nil {
						val = fmt.Sprintf("%v", *dv.Value)
					}
					// OptionSet validation
					if r.hasOptionSet {
						if _, ok := r.optionSetIDs[val]; !ok {
							errs["de.value."+stageID+"."+deID] = "invalid option value"
							continue
						}
					}
					// ValueType validation
					if r.valueType != "" && !validateValueType(r.valueType, val) {
						errs["de.type."+stageID+"."+deID] = "invalid value for type " + r.valueType
					}
				}
				// Mandatory presence for this stage
				for deID, r := range deMap {
					if r.mandatory {
						if _, ok := seen[deID]; !ok {
							errs["de.missing."+stageID+"."+deID] = "mandatory data element missing"
						}
					}
				}
			}
		}
	}

	return len(errs) == 0, errs
}
