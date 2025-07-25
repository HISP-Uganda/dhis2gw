package models

import (
	"database/sql"
	"database/sql/driver"
	"dhis2gw/db"
	"dhis2gw/utils"
	"dhis2gw/utils/dbutils"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Geometry struct {
	Type        string       `json:"type"`
	Coordinates dbutils.JSON `json:"coordinates"`
}

// Scan implements the driver.Valuer interface
func (a *Geometry) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		switch value.(type) {
		case string:
			b = []byte(value.(string))
		default:
			return errors.New(fmt.Sprintf("type assertion to []byte failed type: %T", value))
		}
	}

	return json.Unmarshal(b, &a)
}

func (a Geometry) Value() (driver.Value, error) {
	return json.Marshal(a)
}

type OrgUnitLevel struct {
	ID      string `db:"id" json:"id"`
	UID     string `db:"uid" json:"uid"`
	Code    string `db:"code" json:"code,omitempty"`
	Name    string `db:"name" json:"name"`
	Level   int    `db:"level" json:"level"`
	Created string `db:"created" json:"created,omitempty"`
	Updated string `db:"updated" json:"updated,omitempty"`
}

const insertOrgunitLevelSQL = `
INSERT INTO orgunitlevel(uid, name, level, created, updated)
VALUES(:uid, :name, :level, NOW(), NOW())
`

func (ol *OrgUnitLevel) NewOrgUnitLevel() {
	dbConn := db.GetDB()
	_, err := dbConn.NamedExec(insertOrgunitLevelSQL, ol)
	if err != nil {
		log.WithError(err).Info("ERROR INSERTING OrgUnit Level")
	}
}

type OrgUnitGroup struct {
	ID        string `db:"id" json:"id"`
	UID       string `db:"uid" json:"uid"`
	Code      string `db:"code" json:"code,omitempty"`
	Name      string `db:"name" json:"name"`
	ShortName string `db:"shortname" json:"shortName,omitempty"`
	Created   string `db:"created" json:"created,omitempty"`
	Updated   string `db:"updated" json:"updated,omitempty"`
}

func (og *OrgUnitGroup) DBID() int64 {
	dbConn := db.GetDB()
	var id sql.NullInt64
	err := dbConn.Get(&id, `SELECT id FROM orgunitgroup WHERE uid = $1`, og.UID)
	if err != nil {
		log.WithError(err).WithField("GroupUID", og.UID).Info("Failed to get orgunit group DBID")
		return 0
	}
	return id.Int64
}
func GetOuGroupUIDByName(name string) string {
	dbConn := db.GetDB()
	var uid string
	err := dbConn.Get(&uid, `SELECT uid FROM orgunitgroup WHERE name = $1`, name)
	if err != nil {
		log.WithError(err).Info("Failed to get orgunit group DBUID")
		return ""
	}
	return uid
}

//func GetOrgUnitGroupByName(name string) *OrgUnitGroup {
//	var og OrgUnitGroup
//	dbConn := db.GetDB()
//	err := dbConn.Get(&og, "SELECT id, name, uid FROM orgunitgroup WHERE name=$1", name)
//	if err != nil {
//		return nil
//	}
//	return &og
//}

const insertOrgunitGroupSQL = `
INSERT INTO orgunitgroup(uid, name, shortname, created, updated)
VALUES(:uid, :name, :shortname, NOW(), NOW())`

func (og *OrgUnitGroup) NewOrgUnitGroup() {
	dbConn := db.GetDB()
	_, err := dbConn.NamedExec(insertOrgunitGroupSQL, og)
	if err != nil {
		log.WithError(err).Info("ERROR INSERTING OrgUnit Level")
	}
}

type Attribute struct {
	ID                                string `db:"id" json:"id"`
	UID                               string `db:"uid" json:"uid,omitempty"`
	Code                              string `db:"code" json:"code,omitempty"`
	Name                              string `db:"name" json:"name,omitempty"`
	ShortName                         string `db:"shortname" json:"shortName,omitempty"`
	ValueType                         string `db:"valuetype" json:"valueType,omitempty"`
	Unique                            bool   `db:"isunique" json:"unique,omitempty"`
	Mandatory                         bool   `db:"mandatory" json:"mandatory"`
	OrganisationUnitAttribute         bool   `db:"organisationunitattribute" json:"organisationUnitAttribute,omitempty"`
	OrganisationUnitGroupAttribute    bool   `db:"organisationunitgroupattribute" json:"organisationUnitGroupAttribute,omitempty"`
	OrganisationUnitGroupSetAttribute bool   `db:"organisationunitgroupsetattribute" json:"organisationUnitGroupSetAttribute,omitempty"`
	Created                           string `db:"created" json:"created,omitempty"`
	Updated                           string `db:"updated" json:"updated,omitempty"`
}

type AttributeValue struct {
	Value     string    `json:"value"`
	Attribute Attribute `json:"attribute"`
}

const insertAttributeSQL = `
INSERT INTO attribute(uid, name, shortname, valuetype, mandatory, isunique, organisationunitattribute, created, updated)
VALUES(:uid, :name, :shortname, :valuetype, :mandatory, :isunique, :organisationunitattribute, NOW(), NOW())
`

func (at *Attribute) NewAttribute() {
	dbConn := db.GetDB()
	_, err := dbConn.NamedExec(insertAttributeSQL, at)
	if err != nil {
		log.WithError(err).Error("Failed to insert Attribute")
	}
}

func (at *Attribute) ExistsInDB() bool {
	dbConn := db.GetDB()
	var count int
	err := dbConn.Get(&count, "SELECT count(*) FROM attribute WHERE uid = $1", at.UID)
	if err != nil {
		log.WithError(err).Info("Error reading organisation unit attribute:")
		return false
	}
	return count > 0
}

func (at *Attribute) UpdateCode(code string) {
	dbConn := db.GetDB()
	_, err := dbConn.NamedExec(`UPDATE attribute SET code = :code WHERE uid = :uid`, at)
	if err != nil {
		log.WithError(err).Error("Error updating attrinute code")
	}
}

type OrganisationUnit struct {
	ID               string              `db:"id" json:"id"`
	UID              string              `db:"uid" json:"uid"`
	Code             string              `db:"code" json:"code,omitempty"`
	Name             string              `db:"name" json:"name"`
	ShortName        string              `db:"shortname" json:"shortName,omitempty"`
	Email            string              `db:"email" json:"email,omitempty"`
	URL              string              `db:"url" json:"url,omitempty"`
	Address          string              `db:"address" json:"address,omitempty"`
	DisplayName      string              `db:"-" json:"displayName,omitempty"`
	Description      string              `db:"description" json:"description,omitempty"`
	PhoneNumber      string              `db:"phonenumber" json:"phoneNumber,omitempty"`
	Level            int                 `db:"hierarchylevel" json:"level"`
	ParentID         dbutils.Int         `db:"parentid" json:"parentid,omitempty"`
	Path             string              `db:"path" json:"path,omitempty"`
	MFLID            string              `db:"mflid" json:"mflId,omitempty"`
	MFLUID           string              `db:"mfluid" json:"mflUID,omitempty"`
	MFLParent        sql.NullString      `db:"mflparent" json:"mflParent,omitempty"`
	OpeningDate      string              `db:"openingdate" json:"openingDate"`
	ClosedDate       string              `db:"closeddate" json:"closedDate,omitempty"`
	Deleted          bool                `db:"deleted" json:"deleted,omitempty"`
	Extras           dbutils.MapAnything `db:"extras" json:"extras,omitempty"`
	AttributeValues  dbutils.JSON        `db:"attributevalues" json:"attributeValues,omitempty"`
	LastSyncDate     string              `db:"lastsyncdate" json:"lastSyncDate,omitempty"`
	Geometry         Geometry            `db:"geometry" json:"geometry,omitempty"`
	Created          string              `db:"created" json:"created,omitempty"`
	Updated          string              `db:"updated" json:"updated,omitempty"`
	OrgUnitGroups    []OrgUnitGroup      `json:"organisationUnitGroups,omitempty"`
	OrgUnitRevisions []OrgUnitRevision   `json:"organisationUnitRevisions,omitempty"`
}

func (o *OrganisationUnit) DBID() int64 {
	dbConn := db.GetDB()
	var id sql.NullInt64
	err := dbConn.Get(&id, `SELECT id FROM organisationunit WHERE uid = $1`, o.ID)
	if err != nil {
		log.WithError(err).Info("Failed to get organisation unit id")
	}
	return id.Int64
}

func (o *OrganisationUnit) ValidateUID() bool {
	uidPattern := `^[a-zA-Z0-9]{11}$`
	re := regexp.MustCompile(uidPattern)
	return re.MatchString(o.UID)
}

func (o *OrganisationUnit) Parent() map[string]string {
	dbConn := db.GetDB()
	parentUID := ""
	parentMap := make(map[string]string)
	err := dbConn.Get(&parentUID, `SELECT uid FROM organisationunit WHERE mflid = $1`, o.MFLParent)
	if err != nil {
		log.WithField("UID", o.UID).WithError(err).Error("Could not get parent ID")
		return nil
	}
	parentMap["id"] = parentUID

	return parentMap

}
func (o *OrganisationUnit) ParentByParentId() map[string]string {
	dbConn := db.GetDB()
	parentUID := ""
	parentMap := make(map[string]string)
	if parentId, err := o.ParentID.Value(); err == nil {
		err := dbConn.Get(&parentUID, `SELECT uid FROM organisationunit WHERE id = $1`, parentId)
		if err != nil {
			log.WithField("UID", o.UID).WithError(err).Error("Could not get parent ID")
			return nil
		}
		parentMap["id"] = parentUID
	} else {
		return nil
	}

	return parentMap

}

func (o *OrganisationUnit) ParentByUID() dbutils.Map {
	dbConn := db.GetDB()
	parentUID := ""
	var parentMap dbutils.Map
	if o.ValidateUID() {
		err := dbConn.Get(&parentUID, `SELECT uid FROM organisationunit WHERE id = 
            (SELECT parentid FROM organisationunit WHERE uid = $1)`, o.UID)
		if err != nil {
			log.WithField("UID", o.UID).WithError(err).Error("Could not get parent ID")
			return parentMap
		}
		parentMap.Map()["id"] = parentUID
	}
	return parentMap

}

func (o *OrganisationUnit) GetGroups() []OrgUnitGroup {
	dbConn := db.GetDB()
	ouGroups := []OrgUnitGroup{}
	err := dbConn.Select(&ouGroups, `SELECT * FROM orgunitgroup WHERE id IN 
                (SELECT orgunitgroupid FROM orgunitgroupmembers WHERE organisationunitid = $1)`, o.DBID())
	if err != nil {
		log.WithError(err).Error("Failed to get organisation unit groups")
		return nil
	}
	return ouGroups
}

func (o *OrganisationUnit) DHIS2Payload() dbutils.MapAnything {
	o.ID = o.UID
	if !o.ValidateUID() {
		return nil
	}
	var facilityJSON dbutils.MapAnything
	payload := make(dbutils.MapAnything)
	fj, _ := json.Marshal(o)
	_ = json.Unmarshal(fj, &facilityJSON)

	for k := range facilityJSON {
		switch k {
		case "extras", "url", "uid", "mflId", "mflParent", "mflUID", "parentid", "level", "path":
		default:
			payload[k] = facilityJSON[k]
		}
	}
	parent, _ := o.ParentID.Value()
	if parent != nil {
		payload["parent"] = o.ParentByParentId()
	}
	if len(o.Code) == 0 {
		delete(payload, "code")
	}
	return payload
}

func (o *OrganisationUnit) OrgUnitDHIS2Payload() []byte {
	payload := make(dbutils.MapAnything)
	var facilityMap dbutils.MapAnything
	_ = facilityMap.Scan(o)
	// fj, _ := json.Marshal(o)
	for k := range facilityMap {
		switch k {
		case "extras", "url", "uid", "mflId", "mflParent", "mflUID":
		default:
			payload[k] = facilityMap[k]

		}
	}
	ret, err := json.Marshal(payload)
	if err != nil {
		log.WithError(err).Error("Failed to generate DHIS2 Payload for new facility")
		return nil
	}
	return ret
}

func (o *OrganisationUnit) OrganisationUnitDBFields() []string {
	e := reflect.ValueOf(o).Elem()
	var ret []string
	for i := 0; i < e.NumField(); i++ {
		t := e.Type().Field(i).Tag.Get("db")
		if len(t) > 0 && t != "-" {
			ret = append(ret, t)
		}
	}
	ret = append(ret, "*")
	return ret
}

const insertOrgUnitSQL = `
INSERT INTO organisationunit (uid,name, shortname,path, parentid, hierarchylevel,address,
        email,phonenumber,url,mflid,extras,attributevalues, openingdate, created, updated)
VALUES (:uid, :name,  :shortname, :path, ou_paraent_from_path(:path, :hierarchylevel), 
        :hierarchylevel, :address, :email, :phonenumber, :url, :mflid, :extras, :attributevalues, :openingdate, now(), now())
RETURNING id
`

func (o *OrganisationUnit) ExistsInDB() bool {
	dbConn := db.GetDB()
	var count int
	err := dbConn.Get(&count, "SELECT count(*) FROM organisationunit WHERE uid = $1", o.UID)
	if err != nil {
		log.WithError(err).Info("Error reading organisation unit:")
		return false
	}
	return count > 0
}

func (o *OrganisationUnit) NewOrgUnit() {
	dbConn := db.GetDB()
	rows, err := dbConn.NamedQuery(insertOrgUnitSQL, o)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"UID": o.UID}).Info("Failed to insert Organisation Unit")
		var facilityJSON dbutils.MapAnything
		fj, _ := json.Marshal(o)
		_ = json.Unmarshal(fj, &facilityJSON)
		ouFailure := OrgUnitFailure{UID: utils.GetUID(),
			FacilityUID: o.UID, MFLUID: o.MFLUID, Reason: err.Error(), Object: facilityJSON, Action: "Add"}
		ouFailure.NewOrgUnitFailure()
		return
	}
	for rows.Next() {
		var id sql.NullInt64
		_ = rows.Scan(&id)
		log.WithFields(log.Fields{"ID": id.Int64, "UID": o.UID, "OuByID": o.ID}).Info("Created New OrgUnit")
		o.UpdateGeometry()
		if len(o.OrgUnitGroups) > 0 {
			log.WithField("Groups", o.OrgUnitGroups).Info("Groups on Ou:", o.ID)
			for _, ouGroup := range o.OrgUnitGroups {
				o.AddToGroup(ouGroup)
			}
		}
	}
	_ = rows.Close()
}

// Children ...
func (o *OrganisationUnit) Children() []*OrganisationUnit {
	ouID := o.DBID()
	ids, _ := OrgUnitChildren(ouID)
	dbConn := db.GetDB()
	children := make([]*OrganisationUnit, len(ids))
	for i, id := range ids {
		child := &OrganisationUnit{}
		err := dbConn.Get(child, "SELECT id, uid, name FROM organisationunit WHERE id = $1", id)
		if err != nil {
			// log.WithError(err).WithField("ID", id).Error("Failed to get Organisation Unit")
			continue
		}
		children[i] = child
	}
	return children
}

// GetOrganisationUnitByID ...
func GetOrganisationUnitByID(id int64) (*OrganisationUnit, error) {
	dbConn := db.GetDB()
	ou := &OrganisationUnit{}
	err := dbConn.Get(ou,
		"SELECT id, uid, name, parentid, hierarchylevel, path FROM organisationunit WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	return ou, nil
}

// GetOrganisationUnitsByNames returns a list of organisationunit id given a slice of names
func GetOrganisationUnitsByNames(names []string) ([]int64, error) {
	dbConn := db.GetDB()
	var ids []int64

	// Create placeholders for each name
	placeholders := make([]string, len(names))
	for i := range names {
		placeholders[i] = "$" + strconv.Itoa(i+1) // PostgreSQL placeholders are 1-based
	}

	// Create the query with the placeholders
	query := `SELECT id FROM organisationunit WHERE hierarchylevel = 3 AND name IN (` +
		strings.Join(placeholders, ", ") + `)`

	// Convert []string to []interface{}
	args := make([]interface{}, len(names))
	for i, v := range names {
		args[i] = v
	}

	// Execute the query, passing the names as arguments
	err := dbConn.Select(&ids, query, args...)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

// OrgUnitChildren returns a slice of ids with organisationunit's children
func OrgUnitChildren(id int64) ([]int64, error) {
	dbConn := db.GetDB()
	var ids []int64
	err := dbConn.Select(&ids, `SELECT id FROM organisationunit WHERE parentid = $1`, id)
	if err != nil {
		log.WithError(err).Error("Failed to get children of Organisation Unit")
		return nil, err
	}
	return ids, nil
}

// IsPointInOrganisationUnit a function that returns true if point in organisationunit geometry
// the function takes the id of the organisationunit and a longitude and latitude
func IsPointInOrganisationUnit(id int64, longitude, latitude float64) (bool, error) {
	dbConn := db.GetDB()
	var isPointInGeometry bool
	err := dbConn.Get(&isPointInGeometry, `
	 SELECT ST_Covers(geometry, ST_SetSRID(ST_MakePoint($1, $2), 4326)) 
		FROM organisationunit WHERE id = $3`,
		longitude, latitude, id)
	if err != nil {
		return false, err
	}
	return isPointInGeometry, nil
}

// GetOrganisationUnitById returns the organisationunit given its id
func GetOrganisationUnitById(id int64) (*OrganisationUnit, error) {
	dbConn := db.GetDB()
	ou := &OrganisationUnit{}
	err := dbConn.Get(ou, "SELECT * FROM organisationunit WHERE id = $1", id)
	if err != nil {
		log.WithError(err).WithField("ID", id).Error("Failed to get Organisation Unit")
		return nil, err
	}
	return ou, nil
}

func CompareDefinition(newDefinition, latestDefinition dbutils.MapAnything) (bool, dbutils.MapAnything, error) {
	dbConn := db.GetDB()
	var matches bool
	var diff dbutils.MapAnything
	currentFacilityJSON, err := json.Marshal(newDefinition)
	if err != nil {
		log.WithError(err).Info("Failed to convert facility object to JSON")
		return false, nil, err
	}
	latestRevisionFacilityJSON, err := json.Marshal(latestDefinition)
	if err != nil {
		log.WithError(err).Info("Failed to convert new facility object to JSON")
		return false, diff, err
	}

	err = dbConn.Get(&diff, `SELECT jsonb_diff_val($1::JSONB, $2::JSONB)`,
		currentFacilityJSON, latestRevisionFacilityJSON)
	if err != nil {
		log.WithError(err).Info("Failed the JSON objects for new and old facility definition")
		return false, diff, err
	}
	matches = len(diff) == 0

	return matches, diff, nil
}

func (o *OrganisationUnit) UpdateMFLID(mflID string) {
	dbConn := db.GetDB()
	o.MFLID = mflID
	_, err := dbConn.NamedExec(`UPDATE organisationunit SET mflid = :mflid WHERE uid = :uid`, o)
	if err != nil {
		log.WithError(err).Error("Error updating organisation MFLID")
	}
}

func (o *OrganisationUnit) UpdateMFLUID(mflUID string) {
	dbConn := db.GetDB()
	o.MFLUID = mflUID
	_, err := dbConn.NamedExec(`UPDATE organisationunit SET mfluid = :mfluid WHERE uid = :uid`, o)
	if err != nil {
		log.WithError(err).Error("Error updating organisation MFLUID")
	}
}

func GetOrgUnitByMFLID(mflid string) OrganisationUnit {
	dbConn := db.GetDB()
	var ou OrganisationUnit
	rows, err := dbConn.Queryx(`SELECT id,hierarchylevel,path, uid FROM organisationunit WHERE mflid = $1`, mflid)
	if err != nil {
		log.WithError(err).WithField("MFLID", mflid).Info("Failed to get orgunit DBUID")
	}
	for rows.Next() {
		var id, path, uid string
		var lvl int
		err := rows.Scan(&id, &lvl, &path, &uid)
		if err != nil {
			// log.Fatalln("==>", err)
			log.WithError(err).Error("Error reading request from queue:")
		}
		ou.ID = id
		ou.Level = lvl
		ou.Path = path
		ou.UID = uid
	}
	_ = rows.Close()
	return ou
}

func (o *OrganisationUnit) AddToGroup(ouGroup OrgUnitGroup) {
	dbConn := db.GetDB()
	_, err := dbConn.Exec(`INSERT INTO orgunitgroupmembers (organisationunitid, orgunitgroupid, created, updated)
    			VALUES($1, $2, NOW(), NOW())`, o.DBID(), ouGroup.DBID())
	if err != nil {
		log.WithError(err).WithFields(log.Fields{"OuUID": o.UID, "oUGroup": ouGroup, "oUGroups": o.OrgUnitGroups}).Info(
			"Failed to add orgunit to group")
	}
}

func (o *OrganisationUnit) UpdateMFLParent(mflParent string) {
	dbConn := db.GetDB()
	o.MFLParent = sql.NullString{String: mflParent, Valid: true}
	_, err := dbConn.NamedExec(`UPDATE organisationunit SET mflparent = :mflparent WHERE uid = :uid`, o)
	if err != nil {
		log.WithError(err).Error("Error updating organisation MFLID")
	}
}

func (o *OrganisationUnit) UpdateGeometry() {
	dbConn := db.GetDB()
	if len(o.Geometry.Type) == 0 {
		return
	}
	log.WithField("Geometry", o.Geometry.Type).Info("Going to update Location Geometry")

	var geomObj geom.T
	switch o.Geometry.Type {
	case "Point":
		var coordinates []float64
		if err := json.Unmarshal(o.Geometry.Coordinates, &coordinates); err != nil {
			log.WithError(err).Error("Failed to unmarshal Point coordinates")
			return
		}
		pointGeom := geom.NewPoint(geom.XY).MustSetCoords([]float64{coordinates[0], coordinates[1]})
		geomObj = pointGeom
	case "Polygon":
		var coordinates [][][]float64
		if err := json.Unmarshal(o.Geometry.Coordinates, &coordinates); err != nil {
			log.WithError(err).Error("Failed to unmarshal Polygon coordinates")
			return
		}
		geomObj = getPloygon(coordinates)
	case "MultiPolygon":
		var coordinates [][][][]float64
		if err := json.Unmarshal(o.Geometry.Coordinates, &coordinates); err != nil {
			log.WithError(err).Error("Failed to unmarshal MultiPolygon coordinates")
			return
		}
		geomObj = getMultiPloygon(coordinates)
	default:
		log.WithField("Type", o.Geometry.Type).Error("Unsupported geometry type:")
		return
	}
	geoJSONBytes, err := geojson.Marshal(geomObj)
	if err != nil {
		log.WithError(err).Error("Failed to Marshal Geometry Object")
		return
	}
	geoJSONString := string(geoJSONBytes)
	// log.WithField("geoJSONString", geoJSONString).Info("XXXXX Geo")
	args := dbutils.MapAnything{"geometry": geoJSONString, "uid": o.UID}
	_, _ = dbConn.NamedExec(`UPDATE organisationunit SET geometry = :geometry  WHERE uid = :uid`, args)
}

func getPloygon(coordinates [][][]float64) *geom.Polygon {
	ring := geom.NewLinearRing(geom.XY)
	var coords []geom.Coord
	for _, c := range coordinates[0] {
		coords = append(coords, geom.Coord{c[0], c[1]})

	}
	ring.MustSetCoords(coords)
	polygonGeom := geom.NewPolygon(geom.XY).MustSetCoords([][]geom.Coord{ring.Coords()})
	return polygonGeom
}

func getMultiPloygon(coordinates [][][][]float64) *geom.MultiPolygon {
	polygonGeoms := make([]*geom.Polygon, len(coordinates))
	multiPolygonGeom := geom.NewMultiPolygon(geom.XY)
	for i, c := range coordinates {
		polygonGeoms[i] = getPloygon(c)
		err := multiPolygonGeom.Push(polygonGeoms[i])
		if err != nil {
			log.WithError(err).Info("Failed to push polygon")
			continue
		}
	}
	return multiPolygonGeom
}

type OrgUnitRevision struct {
	ID                  string              `db:"id" json:"id"`
	UID                 string              `db:"uid" json:"uid"`
	IsActive            bool                `db:"is_active" json:"isActive"`
	OrganisationUnitUID dbutils.Int         `db:"organisationunit_id" json:"organisationUnitUID"`
	Revision            int64               `db:"revision" json:"revision"`
	Definition          dbutils.MapAnything `db:"definition" json:"definition"`
	District            string              `db:"district" json:"district,omitempty"`
	Created             string              `db:"created" json:"created,omitempty"`
	Updated             string              `db:"updated" json:"updated,omitempty"`
}

type OrgUnitFailure struct {
	ID          string              `db:"id" json:"id"`
	UID         string              `db:"uid" json:"uid"`
	FacilityUID string              `db:"facility_uid" json:"facilityUID"`
	MFLUID      string              `db:"mfluid" json:"MFLUID"`
	Action      string              `db:"action" json:"action"` // create, update, delete
	Reason      string              `db:"reason" json:"reason"` // error message
	Object      dbutils.MapAnything `db:"object" json:"object"`
	Created     string              `db:"created" json:"created,omitempty"`
	Updated     string              `db:"updated" json:"updated,omitempty"`
}

func (f *OrgUnitFailure) NewOrgUnitFailure() {
	dbConn := db.GetDB()
	_, err := dbConn.NamedExec(`INSERT INTO orgunitfailure(uid, facility_uid, mfluid, 
            action, reason, object) VALUES (:uid, :facility_uid, :mfluid, :action, :reason, :object)`, f)
	if err != nil {
		log.WithError(err).Info("Failed to Log Failure")
	}
	// _ = rows.Close()
}

type MetadataObject struct {
	Operation string `json:"op"`
	Path      string `json:"path"`
	Value     any    `json:"value"`
}

func GenerateMetadataPayload(newFacility dbutils.MapAnything) []MetadataObject {
	// metaDataSlice := make([]MetadataObject, len(diffMap))
	var metaDataSlice []MetadataObject
	for k := range newFacility {
		switch k {
		case "extras", "url", "uid", "mflId", "mflParent", "mflUID", "id", "organisationUnitGroups":
		case "geometry":
			geoType, ok := newFacility[k].(map[string]interface{})["type"]
			if ok {
				if geoType != "" {
					m := MetadataObject{Operation: "add", Path: fmt.Sprintf("/%s", k), Value: newFacility[k]}
					metaDataSlice = append(metaDataSlice, m)
				}

			}

		default:
			m := MetadataObject{Operation: "add", Path: fmt.Sprintf("/%s", k), Value: newFacility[k]}
			metaDataSlice = append(metaDataSlice, m)
		}
	}
	return metaDataSlice
}
