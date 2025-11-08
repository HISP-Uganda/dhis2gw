// cmd/pbs-sync/main.go
package main

import (
	"context"
	"dhis2gw/clients"
	"dhis2gw/clients/pbs"
	"dhis2gw/config"
	"dhis2gw/models"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/HISP-Uganda/go-dhis2-sdk/dhis2/schema"
	log "github.com/sirupsen/logrus"
)

var splash = `
┏━┓┏┓ ┏━┓         ┏┓╻╺┳┓┏━┓╻ ╻   ┏━┓╻ ╻┏┓╻┏━╸
┣━┛┣┻┓┗━┓   ╺━╸   ┃┗┫ ┃┃┣━┛┗━┫   ┗━┓┗┳┛┃┗┫┃
╹  ┗━┛┗━┛         ╹ ╹╺┻┛╹    ╹   ┗━┛ ╹ ╹ ╹┗━╸
`

func main() {
	fmt.Print(splash)

	baseURL := config.DHIS2GWConf.PBS.PBSURL
	// vote := config.DHIS2GWConf.PBS.VoteCode
	fy := config.DHIS2GWConf.PBS.FiscalYear
	interval := config.DHIS2GWConf.PBS.Sync.Interval
	once := config.DHIS2GWConf.PBS.Sync.Once

	// ---- Build token source ----
	var ts pbs.JWTTokenSource
	if config.DHIS2GWConf.PBS.User != "" && config.DHIS2GWConf.PBS.Password != "" {
		ts = pbs.NewPBSTokenSource(
			baseURL,
			config.DHIS2GWConf.PBS.User,
			config.DHIS2GWConf.PBS.Password,
			config.DHIS2GWConf.PBS.IPAddress,
		)
	} else if config.DHIS2GWConf.PBS.JWT != "" {
		ts = pbs.NewStaticJWTSource(config.DHIS2GWConf.PBS.JWT)
	} else {
		log.Fatal("pbs-sync: no authentication config provided")
	}

	// ---- PBS client ----
	client := pbs.NewClient(baseURL, ts)

	// ---- Context ----
	ctx := context.Background()

	token, err := ts.Token(ctx)
	if err != nil {
		log.Fatalf("pbs-sync: token error: %v", err)
	}
	log.Infof("pbs-sync: token: %v", token)

	if once {
		if err := fetchOutturns(ctx, client, fy); err != nil {
			log.Fatalf("pbs-sync: fetch error: %v", err)
		}
		log.Println("pbs-sync: single run completed (Sync.Once=true)")

		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := fetchOutturns(ctx, client, fy); err != nil {
			log.Printf("pbs-sync: fetch error: %v", err)
		}

		select {
		case <-ctx.Done():
			log.Println("pbs-sync: shutting down")
			return
		case <-ticker.C:
		}
	}
}

func fetchOutturns(ctx context.Context, client *pbs.Client, fy string) error {
	log.Infof("pbs-sync: fetching outturns")
	resp, err := pbs.CgBudgetOutturnsByFiscalYear(ctx, client.Gql(), fy)
	if err != nil {
		return err
	}
	log.Infof("pbs-sync: got %d outturn rows", len(resp.CgBudgetOutturnByFiscalYear))
	for _, row := range resp.CgBudgetOutturnByFiscalYear[:10] {
		log.Infof("VoteCode=%s Vote=%s FY=%s Prog=%s Approved=%.2f Q1Exp=%.2f",
			row.Vote_Code, row.Vote_Name, row.Fiscal_Year, row.Programme_Name, row.ApprovedBudget, row.Q1Expenditure)
		dvs, dvsError := BuildPBSDataValues2(row, &config.DHIS2GWConf)
		if dvsError != nil {
			// log.Warnf("pbs-sync: WARN 001: build data values error: %v", dvsError)
			continue
		}
		if len(dvs) == 0 {
			// log.Warnf("pbs-sync: WARN 002: No data values built for vote %s item %s", row.Vote_Code, row.Item_Code)
			continue
		}
		// log.Infof("pbs-sync: built %d data values for vote %s item %s", len(dvs), row.Vote_Code, row.Item_Code)
		// print JSON formatted data values as is for debugging
		for _, dv := range dvs {
			log.WithFields(log.Fields{
				"dataElement":          *dv.DataElement,
				"attributeOptionCombo": *dv.AttributeOptionCombo,
				"period":               *dv.Period,
				"orgUnit":              *dv.OrgUnit,
				"value":                *dv.Value,
			}).Info("pbs-sync: data value")
		}
		// Push to DHIS2 in batches of 200
		batchSize := 200
		for i := 0; i < len(dvs); i += batchSize {
			end := i + batchSize
			if end > len(dvs) {
				end = len(dvs)
			}
			batch := dvs[i:end]

			if err := clients.PushDataValues(batch); err != nil {
				log.Errorf("pbs-sync: push data values error: %v", err)
				continue
			}
			log.Infof("pbs-sync: pushed %d data values to DHIS2", len(batch))
		}

		//if err := models.PushDataValues(ctx, dvs); err != nil {
		//	log.Errorf("pbs-sync: push data values error: %v", err)
		//	continue
		//}
	}
	return nil
}

func BuildPBSDataValues(
	fy string,
	records []map[string]any,
	cfg *config.Config, // pass your loaded config here
) ([]schema.DataValue, error) {

	var dvs []schema.DataValue
	var quarterPattern = regexp.MustCompile(`(?i)^(Q[1-4])[_\-]?(.*)$`)

	for _, record := range records {
		voteCode, _ := record["Vote_Code"].(string)
		ouMapping, err := models.GetOrgUnitMapping(voteCode, "pbs")
		if err != nil {
			log.Warnf("No org unit mapping for vote code %s, skipping", voteCode)
			continue // skip unmapped
		}
		itemCode, _ := record["Item_Code"].(string)
		deMapping, err := models.GetDataElementMapping(itemCode, "pbs")
		if err != nil {
			continue // skip unmapped
		}

		for field, rawVal := range record {
			matches := quarterPattern.FindStringSubmatch(field)
			if len(matches) == 3 {
				quarter := matches[1]
				baseField := strings.TrimSpace(matches[2]) // e.g. "Release", "Spent"
				baseKey := strings.ToLower(baseField)      // match YAML keys

				combo, exists := cfg.PBS.CategoryOptionCombos[baseKey]
				if !exists {
					// fallback to defaultcombo if unknown field
					combo.UID = cfg.PBS.DefaultCategoryOptionCombo
				}
				period := deriveQuarterPeriod(fy, quarter)
				val := fmt.Sprintf("%.0f", rawVal) // convert numeric to string
				dvs = append(dvs, schema.DataValue{
					DataElement:          &deMapping.DataElement,
					AttributeOptionCombo: &combo.UID, // e.g. "HllvX50cXC0"
					Period:               &period,
					OrgUnit:              &ouMapping,
					Value:                &val,
				})
				continue
			}

			if strings.EqualFold(field, "ApprovedBudget") {
				baseKey := "approved" // corresponds to approved in config

				combo, exists := cfg.PBS.CategoryOptionCombos[baseKey]
				if !exists {
					combo.UID = cfg.PBS.DefaultCategoryOptionCombo
				}

				// derive full-year period (e.g. 2025 for FY 2025-2026)
				year := strings.Split(fy, "-")[0]
				period := strings.TrimSpace(fmt.Sprintf("%sJuly", year))

				val := fmt.Sprintf("%.0f", rawVal)

				dvs = append(dvs, schema.DataValue{
					DataElement:          &deMapping.DataElement,
					AttributeOptionCombo: &combo.UID,
					Period:               &period,
					OrgUnit:              &ouMapping,
					Value:                &val,
				})
			}
		}

	}

	return dvs, nil
}

// BuildPBSDataValues2 is an alternative version that uses the generated struct directly
func BuildPBSDataValues2(
	row pbs.CgBudgetOutturnsByFiscalYearCgBudgetOutturnByFiscalYearOpmCgBudgetOutturnDto,
	cfg *config.Config,
) ([]schema.DataValue, error) {
	var dvs []schema.DataValue

	ouMapping, err := models.GetOrgUnitMapping(row.Vote_Code, cfg.PBS.InstanceName)
	if err != nil {
		return nil, fmt.Errorf(
			"no org unit mapping for Vote Code: %s, Vote Name: %s", row.Vote_Code, row.Vote_Name)
	}

	deMapping, err := models.GetDataElementMapping(row.Item_Code, cfg.PBS.InstanceName)
	if err != nil {
		return nil, fmt.Errorf(
			"no data element mapping for Item Code: %s, Item Name: %s", row.Item_Code, row.Item_Description)
	}

	// Helper to get combo UID with default fallback
	getComboUID := func(baseKey string) string {
		if combo, ok := cfg.PBS.CategoryOptionCombos[baseKey]; ok && combo.UID != "" {
			return combo.UID
		}
		return cfg.PBS.DefaultCategoryOptionCombo
	}

	// Helper to append a single data value if value != 0
	appendDV := func(baseKey, period string, v float64) {
		if v == 0 {
			return
		}
		val := fmt.Sprintf("%.0f", v)
		comboUID := getComboUID(baseKey)
		dvs = append(dvs, schema.DataValue{
			DataElement:          &deMapping.DataElement,
			AttributeOptionCombo: &comboUID,
			Period:               &period,
			OrgUnit:              &ouMapping,
			Value:                &val,
		})
	}

	// Table-driven for quarters
	type qrow struct {
		qName       string
		release     float64
		expenditure float64
	}
	quarters := []qrow{
		{"Q1", row.Q1Release, row.Q1Expenditure},
		{"Q2", row.Q2Release, row.Q2Expenditure},
		{"Q3", row.Q3Release, row.Q3Expenditure},
		{"Q4", row.Q4Release, row.Q4Expenditure},
	}

	for _, q := range quarters {
		period := deriveQuarterPeriod(row.Fiscal_Year, q.qName)
		appendDV("expenditure", period, q.expenditure)
		appendDV("release", period, q.release)
	}

	// Approved Budget (annual, July period of first FY year)
	if row.ApprovedBudget != 0 {
		year := strings.Split(row.Fiscal_Year, "-")[0]
		period := strings.TrimSpace(fmt.Sprintf("%sJuly", year))
		appendDV("approved", period, row.ApprovedBudget)
	}

	return dvs, nil
}

// deriveQuarterPeriod converts a fiscal year like "2025-2026" and "Q1"
// into the correct DHIS2 period, assuming fiscal year starts in July.
//
// Quarter mapping (Uganda):
//
//	Q1 → July–Sep  (Year = first part of FY)
//	Q2 → Oct–Dec   (Year = first part of FY)
//	Q3 → Jan–Mar   (Year = second part of FY)
//	Q4 → Apr–Jun   (Year = second part of FY)
//
// Examples:
//
//	deriveQuarterPeriod("2025-2026", "Q1") → "2025Q3"
//	deriveQuarterPeriod("2025-2026", "Q2") → "2025Q4"
//	deriveQuarterPeriod("2025-2026", "Q3") → "2026Q1"
//	deriveQuarterPeriod("2025-2026", "Q4") → "2026Q2"
func deriveQuarterPeriod(fiscalYear string, quarter string) string {
	// Split the fiscal year (e.g., "2025-2026")
	years := strings.FieldsFunc(fiscalYear, func(r rune) bool { return r == '-' || r == '/' })
	startYear := strings.TrimSpace(years[0])
	endYear := startYear
	if len(years) > 1 {
		endYear = strings.TrimSpace(years[1])
	}

	if startYear == "" {
		startYear = fmt.Sprintf("%d", time.Now().Year())
		endYear = startYear
	}

	q := strings.ToUpper(strings.TrimSpace(quarter))
	var year string
	var dhisQuarter string

	switch q {
	case "Q1":
		// Jul–Sep of start year → Calendar Q3
		year, dhisQuarter = startYear, "Q3"
	case "Q2":
		// Oct–Dec of start year → Calendar Q4
		year, dhisQuarter = startYear, "Q4"
	case "Q3":
		// Jan–Mar of next year → Calendar Q1
		year, dhisQuarter = endYear, "Q1"
	case "Q4":
		// Apr–Jun of next year → Calendar Q2
		year, dhisQuarter = endYear, "Q2"
	default:
		// fallback
		year, dhisQuarter = startYear, q
	}

	return fmt.Sprintf("%s%s", year, dhisQuarter)
}
