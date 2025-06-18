package models

import (
	"fmt"
	"github.com/HISP-Uganda/go-dhis2-sdk/aggregate"
	"github.com/HISP-Uganda/go-dhis2-sdk/dhis2/schema"
	log "github.com/sirupsen/logrus"
	"time"
)

type AggregateRequest struct {
	OrgUnit     string         `json:"orgUnit"`
	OrgUnitName string         `json:"orgUnitName,omitempty"`
	Period      string         `json:"period"`
	DataSet     string         `json:"dataSet"`
	DataValues  map[string]any `json:"dataValues"`
}

func (r *AggregateRequest) ToDHIS2AggregatePayload() aggregate.DataValueSetPayload {
	dataValues := ConvertDataValuesToDHIS2DataValues(r.DataValues)
	dateNow := time.Now().Format("2006-01-02")
	return aggregate.DataValueSetPayload{
		DataSet:      r.DataSet,
		Period:       r.Period,
		OrgUnit:      r.OrgUnit,
		CompleteDate: dateNow,
		DataValues:   dataValues,
	}
}

func ConvertDataValuesToDHIS2DataValues(requestDataValues map[string]any) []schema.DataValue {
	dv := []schema.DataValue{}
	codedMapping, err := GetDhis2MappingsByCode()
	if err != nil {
		log.Debugf("Error getting code dimensions: %v", err)
		return dv
	}
	for k, v := range requestDataValues {
		// if k in codedMapping create schema.DataValue and add to dv
		if value, ok := codedMapping[k]; ok {
			// Convert v to string safely
			var strVal string
			switch vTyped := v.(type) {
			case string:
				strVal = vTyped
			case fmt.Stringer:
				strVal = vTyped.String()
			case int, int32, int64, float32, float64, bool:
				strVal = fmt.Sprintf("%v", vTyped)
			default:
				strVal = ""
			}
			dataValue := schema.DataValue{
				DataElement:         &value.DataElement,
				Value:               &strVal,
				CategoryOptionCombo: &value.CategoryOptionCombo,
			}
			dv = append(dv, dataValue)
		} else {
			// key does not exist
		}
	}
	return dv
}
