package main

import (
	"dhis2gw/models"
	"dhis2gw/utils"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	log "github.com/sirupsen/logrus"
)

// --- FetchProjects using existing Resty client and token store ---

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

func SyncProjects(client *resty.Client, baseURL string) error {
	projects, err := FetchProjects(client, baseURL)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{"projects_count": len(projects)}).Info("Syncing projects")
	for _, project := range projects[:3] { // XXX remove limit
		jsonStr, err := json.Marshal(project)
		if err != nil {
			log.WithFields(log.Fields{"project": project}).Errorf("failed to marshal project: %v", err)
			continue
		}
		ou_code := project.ProjectOrganization.Parent.Code
		if ou_code == "" {
			log.WithFields(log.Fields{"project": project}).Errorf("failed to extract orgunit code")
			continue
		}
		ouUID, err2 := models.GetOrgUnitMapping(ou_code, cfg.InstanceName)
		if err2 != nil {
			// log no match for ou
			log.WithFields(log.Fields{
				"ouCODE": ou_code, "instanceName": cfg.InstanceName, "sourceName": cfg.SourceName}).
				Errorf("failed find match for orgunit: %v", err2)
			continue
		}
		payload, err3 := BuildTrackerPayload(string(jsonStr), cfg, ouUID)
		if err3 != nil {
			log.Infof("failed to build tracker payload: %v", err3)
			continue
		}
		log.Infof("Tracker payload: %s", utils.ToPrettyJSON(payload))
	}
	return nil
}
