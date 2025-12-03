package main

import (
	"context"
	"dhis2gw/clients"
	"dhis2gw/models"
	"dhis2gw/utils"
	"fmt"
	"net/http"

	sdk "github.com/HISP-Uganda/go-dhis2-sdk"
	"github.com/HISP-Uganda/go-dhis2-sdk/tracker"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	log "github.com/sirupsen/logrus"
)

// --- FetchProjects using the existing Resty client and token store ---

func FetchProjects(client *resty.Client, baseURL string) ([]Project, error) {
	var projects []Project

	resp, err := client.R().
		SetResult(&projects).
		Get(fmt.Sprintf("%s/integrations/ndp-mne/get-projects", baseURL))

	if err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %v", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode(), resp.String())
	}

	log.Printf("✅ Retrieved %d projects from NDP-MNE", len(projects))
	return projects, nil
}

func SyncProjects(client *resty.Client, baseURL string, dhis2Client *sdk.Client) error {
	projects, err := FetchProjects(client, baseURL)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{"projects_count": len(projects)}).Info("Syncing projects")
	for _, project := range projects[:5] { // XXX remove limit
		jsonStr, err := json.Marshal(project)
		if err != nil {
			log.WithFields(log.Fields{"project": project}).Errorf("failed to marshal project: %v", err)
			continue
		}
		ouCode := project.ProjectOrganization.Parent.Code
		if ouCode == "" {
			log.WithFields(log.Fields{"project": project}).Errorf("failed to extract orgunit code")
			continue
		}
		ouUID, err2 := models.GetOrgUnitMapping(ouCode, cfg.InstanceName)
		if err2 != nil {
			// log no match for ou
			log.WithFields(log.Fields{
				"ouCODE": ouCode, "instanceName": cfg.InstanceName, "sourceName": cfg.SourceName}).
				Errorf("failed find match for orgunit: %v", err2)
			continue
		}
		// search for the existence of TE based on the search attributes
		searchAttr := cfg.Program.SearchAttribute
		if attrPath, searchAttrExists := cfg.Program.TrackedEntityAttributes[searchAttr]; searchAttrExists {
			searchValue := ExtractValue(string(jsonStr), attrPath)
			if searchValue != "" {
				params := map[string]string{
					searchAttr: searchValue,
				}
				// payload, err3 := BuildTrackerPayload(string(jsonStr), cfg, ouUID)
				payload, err3 := BuildTrackerPayloadV2(string(jsonStr), cfg, ouUID)
				if err3 != nil {
					log.Infof("failed to build tracker payload: %v", err3)
					continue
				}

				teExists, tes := clients.SearchTrackedEntity(dhis2Client, ouUID, cfg.Program.ProgramID, params, "EQ")
				if teExists {
					log.Infof("Tracked Entity Exists: %v", utils.ToPrettyJSON(tes))
					if len(tes) > 0 {

						tei := tes[0]
						teUpdatePayload := tracker.TrackedEntityUpdatePayload{
							TrackedEntityType: tei.TrackedEntityType,
							TrackedEntity:     &tei.TrackedEntity,
							OrgUnit:           tei.OrgUnit,
							Attributes:        payload.TrackedEntities[0].Attributes,
						}
						putURL := fmt.Sprintf("trackedEntityInstances/%s?program=%s", tei.TrackedEntity, cfg.Program.ProgramID)
						resp, err := dhis2Client.PutResource(putURL, teUpdatePayload)
						if err != nil || !resp.IsSuccess() {

							log.Infof("Error updating trackedEntity attributes in DHIS2: %v: %v", err, string(resp.Body()))
						}

						// Update data values
						//for _, en := range tei.Enrollments {
						//	for _, ev := range en.Events {
						//		eventUpdatePayload := tracker.EventUpdatePayload{
						//			Event:   ev.Event,
						//			Program: cfg.Program.ProgramID,
						//			OrgUnit: tei.OrgUnit,
						//			Status:  "ACTIVE",
						//		}
						//	}
						//
						//}
						//	dataValuePutURL := fmt.Sprintf("events/%s/%s/", syncLog.EventID, v.DataElement)
						//	resp, err := client.PutResource(dataValuePutURL, ep)
						//	if err != nil || !resp.IsSuccess() {
						//		log.Infof("Error sending result to DHIS2: %v: %v", err, string(resp.Body()))
						//		continue
						//	}

					}

				} else {

					// valid, validationErrors := ValidateTrackerPayload2(payload, *cfg.ProgramConfig)
					valid, validationErrors := ValidateTrackerPayload(payload, *cfg.ProgramConfig)
					if valid || validationErrors == nil {
						log.Infof("Tracker payload: %s", utils.ToPrettyJSON(payload))
						cxt := context.Background()
						// _, restyResponse, err4 := dhis2Client.SendLegacyTrackerPayload(cxt, &payload, nil)
						qprams := map[string]string{"async": "false"}
						_, restyResponse, err4 := dhis2Client.SendTrackerPayload(cxt, &payload, qprams)
						if err4 != nil {
							log.Infof("failed to send tracker payload: %v", err4)
						}
						if restyResponse.StatusCode() == http.StatusOK {
							log.Infof("Tracker payload: %s", utils.ToPrettyJSON(string(restyResponse.Body())))
						} else {
							log.Infof("Failed to submit tracker payload: %s", utils.ToPrettyJSON(string(restyResponse.Body())))
						}

					} else {
						// log invalid tracker payload
						log.Infof("Invalid tracker payload: %s", utils.ToPrettyJSON(validationErrors))
					}
				}
			} else {
				log.Infof("Tracked Entity Search attribute (%v) value is empty:", searchAttr)
			}
		}

	}
	return nil
}
