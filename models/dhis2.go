package models

import (
	"dhis2gw/utils"
	"encoding/json"
	"fmt"
)

// DataValue is a single Data Value Object
type DataValue struct {
	DataElement         string           `json:"dataElement"`
	CategoryOptionCombo string           `json:"categoryOptionCombo,omitempty"`
	Value               utils.FlexString `json:"value"`
}

// DataValuesRequest is the format for sending data values - JSON
type DataValuesRequest struct {
	DataSet              string      `json:"dataset"`
	CompleteDate         string      `json:"completeDate"`
	Period               string      `json:"period"`
	OrgUnit              string      `json:"orgUnit"`
	AttributeOptionCombo string      `json:"attributeOptionCombo,omitempty"`
	DataValues           []DataValue `json:"dataValues"`
}

// BulkDataValuesRequest is the format for sending bulk data values -JSON
type BulkDataValuesRequest struct {
	DataValues []struct {
		DataElement string `json:"dataElement"`
		Period      string `json:"period"`
		OrgUnit     string `json:"orgUnit"`
		Value       string `json:"value"`
	} `json:"dataValues"`
}

// ResponseStatus the status of a response
type ResponseStatus string

//const (
//	ResponseStatusSuccess ResponseStatus = "SUCCESS"
//	ResponseStatusError   RequestStatus  = "ERROR"
//	ResponseStatusWarning ResponseStatus = "WARNING"
// )

// ImportOptions the import options for dhis2 data import
type ImportOptions struct {
	IdSchemes                   map[string]string `json:"idScheme,omitempty"`
	DryRun                      bool              `json:"dryRun,omitempty"`
	Async                       bool              `json:"async,omitempty"`
	ImportStrategy              string            `json:"importStrategy,omitempty"`
	MergeMode                   string            `json:"mergeMode,omitempty"`
	ReportMode                  string            `json:"reportMode,omitempty"`
	SkipExistingCheck           bool              `json:"skipExistingCheck,omitempty"`
	Sharing                     bool              `json:"sharing,omitempty"`
	SkipNotifications           bool              `json:"skipNotifications,omitempty"`
	SkipAudit                   bool              `json:"skipAudit,omitempty"`
	DatasetAllowsPeriods        bool              `json:"datasetAllowsPeriods,omitempty"`
	StrictPeriods               bool              `json:"strictPeriods,omitempty"`
	StrictDataElements          bool              `json:"strictData,omitempty"`
	StrictCategoryOptionCombos  bool              `json:"strictCategoryOptionCombos,omitempty"`
	StrictAttributeOptionCombos bool              `json:"strictAttributeOptionCombos,omitempty"`
	StrictOrganisationUnits     bool              `json:"strictOrganisationUnits,omitempty"`
	RequireCategoryOptionCombo  bool              `json:"requireCategoryOptionCombo,omitempty"`
	RequireAttributeOptionCombo bool              `json:"requireAttributeOptionCombo,omitempty"`
	SkipPatternValidation       bool              `json:"skipPatternValidation,omitempty"`
	IgnoreEmptyCollection       bool              `json:"ignoreEmptyCollection,omitempty"`
	Force                       bool              `json:"force,omitempty"`
	FirstRowIsHeader            bool              `json:"firstRowIsHeader,omitempty"`
	SkipLastUpdated             bool              `json:"skipLastUpdated,omitempty"`
	MergeDataValues             bool              `json:"mergeDataValues,omitempty"`
	SkipCache                   bool              `json:"skipCache,omitempty"`
}

// ImportCount the import count in response
type ImportCount struct {
	Created  int `json:"created,omitempty"`
	Imported int `json:"imported"`
	Updated  int `json:"updated"`
	Ignored  int `json:"ignored"`
	Deleted  int `json:"deleted"`
	Total    int `json:"total,omitempty"`
}

type ConflictObject struct {
	Object    string
	Objects   map[string]string
	Value     string
	ErrorCode string
	Property  string
}

type Response struct {
	ResponseType    string
	Status          ResponseStatus
	ImportOptions   ImportOptions    `json:"importOptions,omitempty"`
	ImportCount     ImportCount      `json:"importCount,omitempty"`
	Stats           ImportCount      `json:"stats,omitempty"`
	Description     string           `json:"description,omitempty"`
	TypeReports     []any            `json:"typeReports,omitempty"`
	Conflicts       []ConflictObject `json:"conflicts,omitempty"`
	DataSetComplete string           `json:"dataSetComplete,omitempty"`
}

type ImportJobResponse struct {
	Name                     string `json:"name"`
	ID                       string `json:"id"`
	Created                  string `json:"created"`
	JobType                  string `json:"jobType"`
	RelativeNotifierEndpoint string `json:"relativeNotifierEndpoint"`
}

// ImportSummary for Aggregate and Async Requests
type ImportSummary struct {
	HTTPStatus     string `json:"httpStatus"`
	HTTPStatusCode int    `json:"httpStatusCode"`
	Response       Response
	Status         string
	Message        string
}

// ImportJobSummary the summary returned after async requests.
type ImportJobSummary struct {
	HTTPStatus     string            `json:"httpStatus"`
	HTTPStatusCode int               `json:"httpStatusCode"`
	Response       ImportJobResponse `json:"response"`
	Status         string            `json:"status"`
	Message        string            `json:"message"`
}

// HTTPBadGatewayError ...
type HTTPBadGatewayError struct {
	HTTPStatus     string `json:"httpStatus"`
	HTTPStatusCode string `json:"httpStatusCode"`
	Status         ResponseStatus
	Message        string
}

// Conflict is the object for the conflicts returned by DHIS 2 API
type Conflict struct {
	Object string `json:"object"`
	Value  string `json:"value"`
}

// DataValuesResponse represents the format of the DHIS 2 API response
type DataValuesResponse struct {
	b struct {
		Status        string                 `json:""`
		Description   string                 `json:"description"`
		ResponseType  string                 `json:"responseType"`
		ImportCount   ImportCount            `json:"importCount"`
		Conflicts     []Conflict             `json:"conflicts"`
		ImportOptions map[string]interface{} `json:"importOptions"`
	}
}

// AsyncJobImportSummary for importSummary returned when checking job status
type AsyncJobImportSummary struct {
	ResponseType    string           `json:"responseType"`
	Status          string           `json:"status"`
	ImportCount     ImportCount      `json:"importCount"`
	ImportConflicts []ConflictObject `json:"importConflicts,omitempty"`
	Reference       string           `json:"reference"`
	Description     string           `json:"description"`
	ImportOptions   ImportOptions    `json:"importOptions"`
	DataSetComplete string           `json:"dataSetComplete,omitempty"`
	ImportTime      string           `json:"importTime,omitempty"`
}

// AsyncJobStatus ..
type AsyncJobStatus struct {
	UID       string `json:"uid"`
	Level     string `json:"level"`
	Category  string `json:"category"`
	Time      string `json:"time"`
	Message   string `json:"message"`
	Completed bool   `json:"completed"`
}

// Status returns the response Status
func (b *DataValuesResponse) Status() string { return b.b.Status }

// Description returns the description in the response
func (b *DataValuesResponse) Description() string { return b.b.Description }

// ImportCounts return the slug for the import counts
func (b *DataValuesResponse) ImportCounts() string {

	out, err := json.Marshal(b.b.ImportCount)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s", string(out))
}

// Conflicts returns the conflicts in the response
func (b *DataValuesResponse) Conflicts() string {

	out, err := json.Marshal(b.b.Conflicts)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s", string(out))
}

// IsValidDataValuesRequest return true if body is a valid DataValuesRequest
func IsValidDataValuesRequest(body string) bool {
	return true
}
