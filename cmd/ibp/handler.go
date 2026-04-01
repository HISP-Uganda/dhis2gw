package main

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	log "github.com/sirupsen/logrus"
)

func GetIBPData(c *gin.Context) {
	vote := c.Query("vote")
	year := c.Query("year")
	dataSet := c.Query("dataSet")
	period := c.Query("period")
	// data := ""

	// "https://train.ndpme.go.ug/ndpdb/api/40/dataValueSets?orgUnit=loDwQx7yYgv&dataSet=h4fLQM9G8vr&period=2025July"
	// r, err := dhis2Client.GetResource("system/info", nil)
	if vote == "" || year == "" || period == "" {

	}

	params := url.Values{}
	params.Add("dataSet", "h4fLQM9G8vr")
	params.Add("period", "2025July")
	params.Add("orgUnit", "loDwQx7yYgv")
	r, err := dhis2Client.GetResourceValues("dataValueSets", params)
	if err != nil {
		log.Errorf("Failed to fetch data: %v", err)
	}
	if r.StatusCode() != http.StatusOK {
		log.Errorf("Failed to fetch data: %v", r.StatusCode())
		// return
	} else {
		log.Infof("Successfully fetched data !!")
		var data map[string]interface{}
		err2 := json.Unmarshal(r.Body(), &data)
		if err2 != nil {
			log.Errorf("Failed to fetch data: %v", err2)
		}
		c.JSON(200, gin.H{
			// "DHIS Version": data["version"].(string),
			"vote":    vote,
			"year":    year,
			"dataSet": dataSet,
			"period":  period,
			"data":    data,
		})
		return
	}

	c.JSON(200, gin.H{"data": "Coming soon......."})

	return
}
