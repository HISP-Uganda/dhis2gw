package controllers

import (
	"bytes"
	"dhis2gw/db"
	"dhis2gw/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type MappingController struct{}

// GetMappingsHandler godoc
// @Summary Get mappings
// @Description Returns a list of DHIS2 mappings
// @Tags mappings
// @Produce json
// @Security BasicAuth
// @Security TokenAuth
// @Param code query string false "Filter by code"
// @Param name query string false "Filter by name"
// @Param dataSet query string false "Filter by data set"
// @Param dataElement query string false "Filter by data element"
// @Param dhis2Name query string false "Filter by DHIS2 name"
// @Param categoryOptionCombo query string false "Filter by category option combo"
// @Param uid query string false "Filter by UID"
// @Param created query string false "Filter by created date (RFC3339 format)"
// @Param updated query string false "Filter by updated date (RFC3339 format)"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 10)"
// @Success 200 {array} models.Dhis2Mapping
// @Failure 500 {object} models.ErrorResponse "Server-side error"
// @Router /mappings [get]
func (m *MappingController) GetMappingsHandler() gin.HandlerFunc {
	// Parse filters from query params (code, name, dataSet, dataElement, dhis2Name, categoryOptionCombo)

	return func(c *gin.Context) {
		var filter models.MappingsFilter
		if code := c.Query("code"); code != "" {
			filter.Code = &code
		}
		if what := c.Query("what"); what != "" {
			filter.What = &what
		}
		if name := c.Query("name"); name != "" {
			filter.Name = &name
		}
		if dataSet := c.Query("dataSet"); dataSet != "" {
			filter.DataSet = &dataSet
		}
		if dataElement := c.Query("dataElement"); dataElement != "" {
			filter.DataElement = &dataElement
		}

		if categoryOptionCombo := c.Query("categoryOptionCombo"); categoryOptionCombo != "" {
			filter.CategoryOptionCombo = &categoryOptionCombo
		}
		if categoryOption := c.Query("categoryOption"); categoryOption != "" {
			filter.CategoryOptionCombo = &categoryOption
		}
		if categoryCombo := c.Query("categoryCombo"); categoryCombo != "" {
			filter.CategoryOptionCombo = &categoryCombo
		}
		if source_name := c.Query("source"); source_name != "" {
			filter.SourceName = &source_name
		}
		if instance_name := c.Query("instance_name"); instance_name != "" {
			filter.InstanceName = &instance_name
		}
		if source_ou := c.Query("source_orgunit"); source_ou != "" {
			filter.SourceOrgUnit = &source_ou
		}
		if destination_ou := c.Query("destination_orgunit"); destination_ou != "" {
			filter.DestinationOrgUnit = &destination_ou
		}
		if uid := c.Query("uid"); uid != "" {
			filter.UID = &uid
		}
		if created := c.Query("created"); created != "" {
			if t, err := time.Parse(time.RFC3339, created); err == nil {
				filter.Created = t
			} else {
				log.WithError(err).Error("Invalid created date format")
				c.JSON(400, gin.H{"error": "Invalid created date format"})
				return
			}
		}
		if updated := c.Query("updated"); updated != "" {
			if t, err := time.Parse(time.RFC3339, updated); err == nil {
				filter.Updated = t
			} else {
				log.WithError(err).Error("Invalid updated date format")
				c.JSON(400, gin.H{"error": "Invalid updated date format"})
				return
			}
		}
		// Pagination params
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
		filter.Page = page
		filter.PageSize = pageSize

		dbConn := db.GetDB()
		var mappings []models.Dhis2Mapping
		err := dbConn.Select(&mappings, "SELECT * FROM dhis2_mappings")
		if err != nil {
			log.WithError(err).Error("Failed to fetch DHIS2 mappings")
			c.JSON(500, gin.H{"error": "Internal server error"})
			return
		}
		c.JSON(200, mappings)
	}
}

type CSVMappingsResponse = models.ImportResponse[models.Dhis2Mapping]

// ImportExcelHandler godoc
// @Summary Import DHIS2 mappings from Excel
// @Description Imports DHIS2 mappings from an Excel file
// @Tags mappings
// @Accept multipart/form-data
// @Produce json
// @Security BasicAuth
// @Security TokenAuth
// @Param file formData file true "Excel file containing DHIS2 mappings"
// @Success 200 {object} CSVMappingsResponse
// @Failure 400 {object} models.ErrorResponse "Invalid file format"
// @Failure 500 {object} models.ErrorResponse "Server-side error"
// @Router /mappings/import/excel [post]
func (m *MappingController) ImportExcelHandler(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file is received"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot open uploaded file"})
		return
	}
	defer file.Close()

	records, err := models.ParseDhis2MappingExcel(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse Excel", "details": err.Error()})
		return
	}
	importError := models.BulkInsertMappings(records)
	if importError != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import mappings", "details": importError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": len(records), "records": records})
}

type ExcelImportResponse = models.ImportResponse[models.Dhis2Mapping]

// ImportCSVHandler godoc
// @Summary Import DHIS2 mappings from CSV
// @Description Imports DHIS2 mappings from a CSV file
// @Tags mappings
// @Accept multipart/form-data
// @Produce json
// @Security BasicAuth
// @Security TokenAuth
// @Param file formData file true "CSV file containing DHIS2 mappings"
// @Success 200 {object} ExcelImportResponse
// @Failure 400 {object} models.ErrorResponse "Invalid file format"
// @Failure 500 {object} models.ErrorResponse "Server-side error"
// @Router /mappings/import/csv [post]
func (m *MappingController) ImportCSVHandler(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file is received"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot open uploaded file"})
		return
	}
	defer file.Close()

	records, err := models.ParseDhis2MappingCSV(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse CSV", "details": err.Error()})
		return
	}
	importError := models.BulkInsertMappings(records)
	if importError != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import mappings", "details": importError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": len(records), "records": records})
}

// ExportExcelTemplateHandler godoc
// @Summary Export Excel template for DHIS2 mappings
// @Description Provides an Excel template for DHIS2 mappings
// @Tags mappings
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BasicAuth
// @Security TokenAuth
// @Success 200 {file} file "Excel template file"
// @Failure 500 {object} models.ErrorResponse "Server-side error"
// @Router /mappings/export/excel-template [get]
func (m *MappingController) ExportExcelTemplateHandler(c *gin.Context) {
	fileBytes, err := models.GenerateDhis2MappingExcelTemplate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate Excel template"})
		return
	}
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", `attachment; filename="dhis2_mappings_template.xlsx"`)
	var buf bytes.Buffer
	if err := fileBytes.Write(&buf); err != nil {
		c.String(http.StatusInternalServerError, "failed to generate Excel: %v", err)
		return
	}
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// ExportExcelMappingsHandler godoc
// @Summary Export DHIS2 mappings to Excel
// @Description Exports DHIS2 mappings to an Excel file
// @Tags mappings
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BasicAuth
// @Security TokenAuth
// @Success 200 {file} file "Excel file containing DHIS2 mappings"
// @Failure 500 {object} models.ErrorResponse "Server-side error"
// @Router /mappings/export/excel [get]
func (m *MappingController) ExportExcelMappingsHandler(c *gin.Context) {
	mappings, err := models.GetAllMappings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch mappings"})
		return
	}
	fileBytes, err := models.GenerateDhis2MappingExcel(mappings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate Excel"})
		return
	}
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", `attachment; filename="dhis2_mappings.xlsx"`)
	var buf bytes.Buffer
	if err := fileBytes.Write(&buf); err != nil {
		c.String(http.StatusInternalServerError, "failed to generate Excel: %v", err)
		return
	}
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}
