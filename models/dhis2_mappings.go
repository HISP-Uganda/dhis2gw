package models

import (
	"context"
	"database/sql"
	"dhis2gw/db"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
)

type Dhis2Mapping struct {
	ID                  int64     `json:"id,omitempty" db:"id"`
	UID                 string    `json:"uid" db:"uid"`
	Code                string    `json:"code" db:"code"`
	What                string    `json:"what" db:"what"`
	Name                string    `json:"name,omitempty" db:"name"`
	Description         string    `json:"description,omitempty" db:"description"`
	DataSet             string    `json:"dataSet,omitempty" db:"dataset"`
	DataElement         string    `json:"dataElement,omitempty" db:"dataelement"`
	CategoryOptionCombo *string   `json:"categoryOptionCombo,omitempty" db:"category_option_combo"`
	CategoryOption      string    `json:"categoryOption,omitempty" db:"category_option"`
	CategoryCombo       string    `json:"categoryCombo,omitempty" db:"category_combo"`
	InstanceName        string    `json:"instanceName,omitempty" db:"instance_name"`
	SourceName          string    `json:"sourceName,omitempty" db:"source_name"`
	SourceOrgUnit       string    `json:"sourceOrgUnit,omitempty" db:"source_orgunit"`
	DestinationOrgUnit  string    `json:"destinationOrgUnit,omitempty" db:"dest_orgunit"`
	DimensionType       string    `json:"dimensionType,omitempty" db:"dimension_type"`
	Created             time.Time `json:"created,omitempty" db:"created"`
	Updated             time.Time `json:"updated,omitempty" db:"updated"`
}

type Dhis2MappingDimension struct {
	ID                  int64     `db:"id" json:"id"`
	MappingID           int64     `db:"mapping_id" json:"mappingId"`
	SourceField         string    `db:"source_field" json:"sourceField"`
	SourceLabel         string    `db:"source_label" json:"sourceLabel,omitempty"`
	CategoryOption      string    `db:"category_option" json:"categoryOption"`            // DHIS2 UID
	CategoryOptionCombo string    `db:"category_option_combo" json:"categoryOptionCombo"` // DHIS2 UID
	Type                string    `db:"type" json:"type,omitempty"`                       // attribution, disaggregation, etc.
	DimensionGroup      string    `db:"dimension_group" json:"dimensionGroup,omitempty"`
	Created             time.Time `db:"created" json:"created"`
	Updated             time.Time `db:"updated" json:"updated"`
}

type MappingsFilter struct {
	Code                *string   `json:"code,omitempty"`
	What                *string   `json:"what,omitempty"`
	Name                *string   `json:"name,omitempty"`
	DataSet             *string   `json:"dataSet,omitempty"`
	DataElement         *string   `json:"dataElement,omitempty"`
	CategoryOptionCombo *string   `json:"categoryOptionCombo,omitempty"`
	CategoryOption      *string   `json:"categoryOption,omitempty"`
	CategoryCombo       *string   `json:"categoryCombo,omitempty"`
	InstanceName        *string   `json:"instanceName,omitempty"`
	SourceName          *string   `json:"sourceName,omitempty"`
	SourceOrgUnit       *string   `json:"sourceOrgUnit,omitempty"`
	DestinationOrgUnit  *string   `json:"destinationOrgUnit,omitempty"`
	UID                 *string   `json:"uid,omitempty"`
	Page                int       `json:"page,omitempty"`
	PageSize            int       `json:"pageSize,omitempty"`
	Created             time.Time `json:"created,omitempty"`
	Updated             time.Time `json:"updated,omitempty"`
}

const insertDhis2MappingSQL = `
INSERT INTO dhis2_mappings(code, what, name, description, dataset, dataelement, 
    category_option_combo, category_option, category_combo, instance_name, source_name, source_orgunit, 
	dest_orgunit, created, updated)
VALUES(:code, :what, :name, :description, :dataset, :dataelement, 
    :category_option_combo, :category_option, :category_combo, :instance_name, :source_name, :source_orgunit, 
	:dest_orgunit, NOW(), NOW()) 
ON CONFLICT ON CONSTRAINT unique_code_source_instance_what
DO NOTHING 
RETURNING id`

// Insert adds a new Dhis2Mapping
func (d *Dhis2Mapping) Insert() (int64, error) {
	dbConn := db.GetDB()
	var id int64
	err := dbConn.QueryRowx(insertDhis2MappingSQL, d).Scan(&id)
	if err != nil {
		log.WithError(err).Error("Failed to insert Dhis2Mapping")
		return 0, err
	}
	return id, nil
}

// Update updates a Dhis2Mapping
func (d *Dhis2Mapping) Update() error {
	dbConn := db.GetDB()
	_, err := dbConn.NamedExec(`
    UPDATE dhis2_mappings SET name = :name, what = :what, description = :description, dataset = :dataset, 
    dataelement = :dataelement, category_option_combo = :category_option_combo,
	category_option = :category_option, category_combo = :category_combo,
	instance_name = :instance_name, source_name = :source_name,
	source_orgunit = :source_orgunit, dest_orgunit = :dest_orgunit,
    updated = NOW() WHERE uid = :uid`, d)
	if err != nil {
		log.WithError(err).Error("Failed to update Dhis2Mapping")
		return err
	}
	return nil
}

// Delete a Dhis2Mapping
func (d *Dhis2Mapping) Delete() error {
	dbConn := db.GetDB()
	_, err := dbConn.Exec("DELETE FROM dhis2_mappings WHERE id = $1", d.ID)
	if err != nil {
		log.WithError(err).Error("Failed to delete Dhis2Mapping")
		return err
	}
	return nil
}

func (d *Dhis2Mapping) DbID() int64 {
	dbConn := db.GetDB()
	var id sql.NullInt64
	err := dbConn.Get(&id, `SELECT id FROM dhis2_mappings WHERE uid = $1`, d.UID)
	if err != nil {
		log.WithError(err).Info("Failed to get device id")
	}
	return id.Int64
}

// InsertOrUpdate a Dhis2Mapping
func (d *Dhis2Mapping) InsertOrUpdate() error {
	if d.ID == 0 {
		_, err := d.Insert()
		return err
	}
	d.ID = d.DbID()
	return d.Update()
}

// GetDhis2Mappings returns a map[string]*Dhis2Mapping where the key is the name of the mapping
func GetDhis2Mappings() (map[string]*Dhis2Mapping, error) {
	dbConn := db.GetDB()
	rows, err := dbConn.Queryx("SELECT * FROM dhis2_mappings")
	if err != nil {
		log.WithError(err).Error("Failed to get Dhis2Mappings")
		return nil, err
	}
	defer rows.Close()

	mappings := make(map[string]*Dhis2Mapping)
	for rows.Next() {
		var m Dhis2Mapping
		err := rows.StructScan(&m)
		if err != nil {
			log.WithError(err).Error("Failed to scan Dhis2Mapping row")
			continue
		}
		mappings[m.Name] = &m
	}
	return mappings, nil
}

// GetDhis2MappingsByCode a map[string]*Dhis2Mapping where the key is the code of the mapping
func GetDhis2MappingsByCode(scheme, source, instance string) (map[string]*Dhis2Mapping, error) {
	dbConn := db.GetDB()
	rows, err := dbConn.Queryx("SELECT * FROM dhis2_mappings WHERE source_name = $1 AND instance_name = $2", source, instance)
	if err != nil {
		log.WithError(err).Error("Failed to get Dhis2Mappings")
		return nil, err
	}
	defer rows.Close()
	mappings := make(map[string]*Dhis2Mapping)
	for rows.Next() {
		var m Dhis2Mapping
		err := rows.StructScan(&m)
		if err != nil {
			log.WithError(err).Error("Failed to scan Dhis2Mapping row")
			continue
		}
		if scheme != "" && scheme == "UID" {

			mappings[m.DataElement] = &m
		} else {
			mappings[m.Code] = &m
		}
	}
	return mappings, nil
}

// GetMappingsByFilter returns a slice of Dhis2Mapping filtered by the provided MappingsFilter
func GetMappingsByFilter(filter MappingsFilter) ([]Dhis2Mapping, int, error) {
	dbConn := db.GetDB()
	var (
		mappings []Dhis2Mapping
		args     []interface{}
		where    []string
		query    = `SELECT * FROM dhis2_mappings`
		countQ   = `SELECT COUNT(*) FROM dhis2_mappings`
	)

	if filter.Code != nil {
		where = append(where, "code = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.Code)
	}
	if filter.Name != nil {
		where = append(where, "name = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.Name)
	}
	if filter.What != nil {
		where = append(where, "what = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.What)
	}
	if filter.DataSet != nil {
		where = append(where, "dataset = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.DataSet)
	}
	if filter.DataElement != nil {
		where = append(where, "dataelement = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.DataElement)
	}

	if filter.CategoryOptionCombo != nil {
		where = append(where, "category_option_combo = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.CategoryOptionCombo)
	}
	if filter.CategoryOption != nil {
		where = append(where, "category_option = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.CategoryOption)
	}
	if filter.CategoryCombo != nil {
		where = append(where, "category_combo = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.CategoryCombo)
	}
	if filter.CategoryOptionCombo != nil {
		where = append(where, "category_option_combo = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.CategoryOptionCombo)
	}
	if filter.UID != nil {
		where = append(where, "uid = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.UID)
	}
	if filter.InstanceName != nil {
		where = append(where, "instance_name = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.InstanceName)
	}
	if filter.SourceName != nil {
		where = append(where, "source_name = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.SourceName)
	}

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	err := dbConn.Select(&mappings, query, args...)
	if err != nil {
		log.WithError(err).Error("Failed to get Dhis2Mappings by filter")
		return nil, 0, err
	}

	var count int
	err = dbConn.Get(&count, countQ, args...)
	if err != nil {
		log.WithError(err).Error("Failed to count Dhis2Mappings")
		return nil, 0, err
	}

	return mappings, count, nil
}

// BulkInsertMappings inserts multiple Dhis2Mappings in a single transaction
func BulkInsertMappings(mappings []Dhis2Mapping) error {
	dbConn := db.GetDB()
	tx, err := dbConn.Beginx()
	if err != nil {
		log.WithError(err).Error("Failed to begin transaction for bulk insert")
		return err
	}

	stmt, err := tx.PrepareNamed(insertDhis2MappingSQL)
	if err != nil {
		log.WithError(err).Error("Failed to prepare statement for bulk insert")
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, m := range mappings {
		if _, err := stmt.Exec(m); err != nil {
			log.WithError(err).Error("Failed to execute bulk insert for Dhis2Mapping")
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func ParseDhis2MappingCSV(file multipart.File) ([]Dhis2Mapping, error) {
	var mappings []Dhis2Mapping

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}
	headerMap := map[string]int{}
	for i, h := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		var m Dhis2Mapping
		if idx, ok := headerMap["code"]; ok {
			m.Code = record[idx]
		}
		if idx, ok := headerMap["what"]; ok {
			m.What = record[idx]
		}
		if idx, ok := headerMap["name"]; ok {
			m.Name = record[idx]
		}
		if idx, ok := headerMap["description"]; ok {
			m.Description = record[idx]
		}
		if idx, ok := headerMap["dataset"]; ok {
			m.DataSet = record[idx]
		}
		if idx, ok := headerMap["dataelement"]; ok {
			m.DataElement = record[idx]
		}

		if idx, ok := headerMap["category_option_combo"]; ok && idx < len(record) {
			if record[idx] != "" {
				v := record[idx]
				m.CategoryOptionCombo = &v
			}
		}
		if idx, ok := headerMap["category_option"]; ok {
			m.CategoryOption = record[idx]
		}
		if idx, ok := headerMap["category_combo"]; ok {
			m.CategoryCombo = record[idx]
		}
		if idx, ok := headerMap["instance_name"]; ok {
			m.InstanceName = record[idx]
		}
		if idx, ok := headerMap["source_name"]; ok {
			m.SourceName = record[idx]
		}
		mappings = append(mappings, m)
	}
	return mappings, nil
}

func ParseDhis2MappingExcel(file multipart.File) ([]Dhis2Mapping, error) {
	var mappings []Dhis2Mapping

	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, err
	}
	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("no rows found")
	}
	headers := rows[0]
	headerMap := map[string]int{}
	for i, h := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	for _, row := range rows[1:] {
		var m Dhis2Mapping

		if idx, ok := headerMap["code"]; ok && idx < len(row) {
			m.Code = row[idx]
		}
		if idx, ok := headerMap["what"]; ok && idx < len(row) {
			m.What = row[idx]
		}
		if idx, ok := headerMap["name"]; ok && idx < len(row) {
			m.Name = row[idx]
		}
		if idx, ok := headerMap["description"]; ok && idx < len(row) {
			m.Description = row[idx]
		}
		if idx, ok := headerMap["dataset"]; ok && idx < len(row) {
			m.DataSet = row[idx]
		}
		if idx, ok := headerMap["dataelement"]; ok && idx < len(row) {
			m.DataElement = row[idx]
		}

		if idx, ok := headerMap["category_option_combo"]; ok && idx < len(row) {
			if row[idx] != "" {
				v := row[idx]
				m.CategoryOptionCombo = &v
			}
		}
		if idx, ok := headerMap["category_option"]; ok && idx < len(row) {
			m.CategoryOption = row[idx]
		}
		if idx, ok := headerMap["category_combo"]; ok && idx < len(row) {
			m.CategoryCombo = row[idx]
		}
		if idx, ok := headerMap["instance_name"]; ok && idx < len(row) {
			m.InstanceName = row[idx]
		}
		if idx, ok := headerMap["source_name"]; ok && idx < len(row) {
			m.SourceName = row[idx]
		}
		if idx, ok := headerMap["source_orgunit"]; ok && idx < len(row) {
			m.SourceOrgUnit = row[idx]
		}
		if idx, ok := headerMap["destination_orgunit"]; ok && idx < len(row) {
			m.DestinationOrgUnit = row[idx]
		}
		mappings = append(mappings, m)
	}
	return mappings, nil
}

// GetAllMappings returns all Dhis2Mappings
func GetAllMappings() ([]Dhis2Mapping, error) {
	dbConn := db.GetDB()
	var mappings []Dhis2Mapping
	err := dbConn.Select(&mappings, "SELECT * FROM dhis2_mappings")
	if err != nil {
		log.WithError(err).Error("Failed to get all Dhis2Mappings")
		return nil, err
	}
	return mappings, nil
}

// GenerateDhis2MappingExcel generates an Excel file from a slice of Dhis2Mapping
func GenerateDhis2MappingExcel(mappings []Dhis2Mapping) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Mappings"
	f.NewSheet(sheet)

	headers := []string{
		"code", "what", "name", "description", "dataset", "dataelement",
		"category_option_combo", "category_option", "category_combo",
		"instance_name", "source_name", "source_orgunit", "destination_orgunit",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	for r, m := range mappings {
		values := []interface{}{
			m.Code, m.What, m.Name, m.Description, m.DataSet, m.DataElement,
			m.CategoryOptionCombo, m.CategoryOption, m.CategoryCombo,
			m.InstanceName, m.SourceName, m.SourceOrgUnit, m.DestinationOrgUnit,
		}
		for c, v := range values {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			f.SetCellValue(sheet, cell, v)
		}
	}

	// Set active sheet by name
	idx, err := f.GetSheetIndex(sheet)
	if err != nil {
		return nil, fmt.Errorf("get sheet index %q: %w", sheet, err)
	}
	f.SetActiveSheet(idx)

	return f, nil
}

// GenerateDhis2MappingExcelTemplate generates an Excel template for Dhis2Mapping
func GenerateDhis2MappingExcelTemplate() (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Mappings"
	f.NewSheet(sheet)

	headers := []string{
		"code", "what", "name", "description", "dataset", "dataelement",
		"category_option_combo", "category_option", "category_combo",
		"instance_name", "source_name", "source_orgunit", "destination_orgunit",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Set active sheet by name
	idx, err := f.GetSheetIndex(sheet)
	if err != nil {
		return nil, fmt.Errorf("get sheet index %q: %w", sheet, err)
	}
	f.SetActiveSheet(idx)

	return f, nil
}

// GetOrgUnitMapping returns the destination org unit for a given source org unit and instance
func GetOrgUnitMapping(sourceOrgUnit, instanceName string) (string, error) {
	dbConn := db.GetDB()
	var destOrgUnit sql.NullString
	err := dbConn.Get(&destOrgUnit, `SELECT dest_orgunit FROM dhis2_mappings 
		WHERE source_orgunit = $1 AND instance_name = $2 AND what = 'ou' LIMIT 1`, sourceOrgUnit, instanceName)
	if err != nil {
		// log.WithError(err).Warn("Failed to get org unit mapping")
		return "", err
	}
	if destOrgUnit.Valid {
		return destOrgUnit.String, nil
	}
	return "", nil
}

// GetDataElementMapping returns the data element mapping for a given code and instance
func GetDataElementMapping(code, instanceName string) (*Dhis2Mapping, error) {
	dbConn := db.GetDB()
	var mapping Dhis2Mapping
	err := dbConn.Get(&mapping, `SELECT * FROM dhis2_mappings 
		WHERE code = $1 AND instance_name = $2 AND what = 'de' LIMIT 1`, code, instanceName)
	if err != nil {
		// log.WithError(err).Error("Failed to get data element mapping")
		return nil, err
	}
	return &mapping, nil
}

var ErrMappingNotFound = errors.New("data element mapping not found")

func GetDataElementMappingWithContext(ctx context.Context, dbConn sqlx.QueryerContext, code, instanceName string) (*Dhis2Mapping, error) {
	code = strings.TrimSpace(code)
	instanceName = strings.TrimSpace(instanceName)

	const q = `
		SELECT *
		FROM dhis2_mappings
		WHERE code = $1
		  AND instance_name = $2
		  AND what = 'de'
		LIMIT 1
	`

	var mapping Dhis2Mapping
	err := sqlx.GetContext(ctx, dbConn, &mapping, q, code, instanceName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: code=%q instance=%q", ErrMappingNotFound, code, instanceName)
		}
		return nil, fmt.Errorf("get data element mapping failed: code=%q instance=%q: %w", code, instanceName, err)
	}

	return &mapping, nil
}

// GetDataElementMappingByUID returns the data element mapping for a given UID and instance
func GetDataElementMappingByUID(uid, instanceName string) (*Dhis2Mapping, error) {
	dbConn := db.GetDB()
	var mapping Dhis2Mapping
	err := dbConn.Get(&mapping, `SELECT * FROM dhis2_mappings 
		WHERE uid = $1 AND instance_name = $2 AND what = 'de' LIMIT 1`, uid, instanceName)
	if err != nil {
		log.WithError(err).Error("Failed to get data element mapping by UID")
		return nil, err
	}
	return &mapping, nil
}
