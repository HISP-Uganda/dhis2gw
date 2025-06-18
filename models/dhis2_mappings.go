package models

import (
	"database/sql"
	"dhis2gw/db"
	log "github.com/sirupsen/logrus"
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
func GetDhis2MappingsByCode() (map[string]*Dhis2Mapping, error) {
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
		mappings[m.Code] = &m
	}
	return mappings, nil
}
