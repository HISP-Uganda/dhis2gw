package utils

import (
	"embed"
	"encoding/json"
	"github.com/xeipuuv/gojsonschema"
)

func ValidateJSONAgainstSchemaString(schema string, document interface{}) (bool, []string, error) {
	schemaLoader := gojsonschema.NewStringLoader(schema)
	docLoader := gojsonschema.NewGoLoader(document)
	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return false, nil, err
	}
	if result.Valid() {
		return true, nil, nil
	}
	var errors []string
	for _, desc := range result.Errors() {
		errors = append(errors, desc.String())
	}
	return false, errors, nil
}

func ValidateJSONAgainstSchema(schemaBytes []byte, document interface{}) (bool, []string, error) {
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	docLoader := gojsonschema.NewGoLoader(document)
	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return false, nil, err
	}
	if result.Valid() {
		return true, nil, nil
	}
	var errors []string
	for _, desc := range result.Errors() {
		errors = append(errors, desc.String())
	}
	return false, errors, nil
}

func UnmarshalToStruct(raw map[string]interface{}, out interface{}) error {
	jsonBytes, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBytes, out)
}

func LoadSchemaFromEmbed(fs embed.FS, path string) ([]byte, error) {
	return fs.ReadFile(path)
}
