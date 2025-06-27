package models

import (
	"database/sql"
	"dhis2gw/db"
	"encoding/csv"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
	"io"
	"mime/multipart"
	"strconv"
	"strings"
	"time"
)

type Dhis2Mapping struct {
	ID                  int64     `json:"id,omitempty" db:"id"`
	UID                 string    `json:"uid" db:"uid"`
	Code                string    `json:"code" db:"code"`
	Name                string    `json:"name,omitempty" db:"name"`
	Description         string    `json:"description,omitempty" db:"description"`
	DataSet             string    `json:"dataSet,omitempty" db:"dataset"`
	DataElement         string    `json:"dataElement,omitempty" db:"dataelement"`
	Dhis2Name           string    `json:"dhis2Name,omitempty" db:"dhis2_name"`
	CategoryOptionCombo string    `json:"categoryOptionCombo,omitempty" db:"category_option_combo"`
	Created             time.Time `json:"created,omitempty" db:"created"`
	Updated             time.Time `json:"updated,omitempty" db:"updated"`
}

type MappingsFilter struct {
	Code                *string   `json:"code,omitempty"`
	Name                *string   `json:"name,omitempty"`
	DataSet             *string   `json:"dataSet,omitempty"`
	DataElement         *string   `json:"dataElement,omitempty"`
	Dhis2Name           *string   `json:"dhis2Name,omitempty"`
	CategoryOptionCombo *string   `json:"categoryOptionCombo,omitempty"`
	UID                 *string   `json:"uid,omitempty"`
	Page                int       `json:"page,omitempty"`
	PageSize            int       `json:"pageSize,omitempty"`
	Created             time.Time `json:"created,omitempty"`
	Updated             time.Time `json:"updated,omitempty"`
}

const insertDhis2MappingSQL = `
INSERT INTO dhis2_mappings(name, description, dataset, dataelement, dhis2_name, 
    category_option_combo, created, updated)
VALUES(:name, :description, :data_set, :data_element, :dhis2_name, 
    :category_option_combo, NOW(), NOW()) RETURNING id`

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
    UPDATE dhis2_mappings SET name = :name, description = :description, dataset = :data_set, 
    dataelement = :data_element, dhis2_name = :dhis2_name, category_option_combo = :category_option_combo, 
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
func GetDhis2MappingsByCode(scheme string) (map[string]*Dhis2Mapping, error) {
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
	if filter.DataSet != nil {
		where = append(where, "dataset = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.DataSet)
	}
	if filter.DataElement != nil {
		where = append(where, "dataelement = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.DataElement)
	}
	if filter.Dhis2Name != nil {
		where = append(where, "dhis2_name = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.Dhis2Name)
	}
	if filter.CategoryOptionCombo != nil {
		where = append(where, "category_option_combo = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.CategoryOptionCombo)
	}
	if filter.UID != nil {
		where = append(where, "uid = $"+strconv.Itoa(len(args)+1))
		args = append(args, *filter.UID)
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
		if idx, ok := headerMap["dhis2_name"]; ok {
			m.Dhis2Name = record[idx]
		}
		if idx, ok := headerMap["category_option_combo"]; ok {
			m.CategoryOptionCombo = record[idx]
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
		if idx, ok := headerMap["dhis2_name"]; ok && idx < len(row) {
			m.Dhis2Name = row[idx]
		}
		if idx, ok := headerMap["category_option_combo"]; ok && idx < len(row) {
			m.CategoryOptionCombo = row[idx]
		}
		mappings = append(mappings, m)
	}
	return mappings, nil
}
