// cmd/pbs-sync/main.go
package main

import (
	"context"
	"dhis2gw/clients"
	"dhis2gw/clients/pbs"
	"dhis2gw/config"
	"dhis2gw/db"
	"dhis2gw/mappings"
	"dhis2gw/models"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/HISP-Uganda/go-dhis2-sdk/dhis2/schema"
	log "github.com/sirupsen/logrus"
)

var splash = `
┏━┓┏┓ ┏━┓         ┏┓╻╺┳┓┏━┓╻ ╻   ┏━┓╻ ╻┏┓╻┏━╸
┣━┛┣┻┓┗━┓   ╺━╸   ┃┗┫ ┃┃┣━┛┗━┫   ┗━┓┗┳┛┃┗┫┃
╹  ┗━┛┗━┛         ╹ ╹╺┻┛╹    ╹   ┗━┛ ╹ ╹ ╹┗━╸
`

const dvBatchSize = 50

func main() {
	fmt.Print(splash)
	runtimeCfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	config.Set(runtimeCfg)
	cfg := runtimeCfg.Config
	if _, err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	if err := clients.Init(); err != nil {
		log.Fatalf("Failed to initialize DHIS2 client: %v", err)
	}
	if err := models.InitLocation(); err != nil {
		log.Fatalf("Failed to initialize schedules location: %v", err)
	}
	if _, err := config.Watch(func(_, _ *config.RuntimeConfig) {
		if _, err := db.Init(); err != nil {
			log.WithError(err).Error("Failed to reload database")
		}
		if err := clients.Init(); err != nil {
			log.WithError(err).Error("Failed to reload DHIS2 client")
		}
		if err := models.InitLocation(); err != nil {
			log.WithError(err).Error("Failed to reload schedules location")
		}
	}); err != nil {
		log.WithError(err).Warn("Failed to start config watcher")
	}

	baseURL := cfg.PBS.PBSURL
	fy := cfg.PBS.FiscalYear
	interval := cfg.PBS.Sync.Interval
	once := cfg.PBS.Sync.Once

	cacheDir, err := resolveCacheDir(cfg.PBS.Cache.CacheDir, "pbs-sync")
	if err != nil {
		log.Fatalf("failed to resolve cache dir(%s): %v", cacheDir, err)
	}
	log.Infof("Cache directory: %s", cacheDir)

	dbConn := db.GetDB()
	ctx := context.Background()

	mappingCache := mappings.NewMappingCache(dbConn, "pbs")

	if err := mappingCache.Load(ctx); err != nil {
		log.Fatalf("failed to load mapping cache: %v", err)
	}

	log.Printf("Loaded %d mappings", mappingCache.Size())

	// ---- Build token source ----
	var ts pbs.JWTTokenSource
	if cfg.PBS.User != "" && cfg.PBS.Password != "" {
		ts = pbs.NewPBSTokenSource(
			baseURL,
			cfg.PBS.User,
			cfg.PBS.Password,
			cfg.PBS.IPAddress,
		)
	} else if cfg.PBS.JWT != "" {
		ts = pbs.NewStaticJWTSource(cfg.PBS.JWT)
	} else {
		log.Fatal("pbs-sync: no authentication config provided")
	}

	// ---- PBS client ----
	client := pbs.NewClient(baseURL, ts)

	// ---- Graceful shutdown context ----
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ---- Validate token early ----
	tokenCtx, cancelToken := context.WithTimeout(rootCtx, 10*time.Minute)
	defer cancelToken()

	_, err = ts.Token(tokenCtx)
	if err != nil {
		if !cfg.PBS.Cache.UseCacheOnly {
			log.Fatalf("pbs-sync: token error: %v", err)

		}
		// log.Fatalf("pbs-sync: token error: %v", err)
	}
	// log.Infof("pbs-sync: token: %v", token)

	if once {
		log.Info("pbs-sync: running once")
		//if err := fetchOutturns(ctx, client, fy); err != nil {
		//	log.Fatalf("pbs-sync: fetch error: %v", err)
		//}
		// Give long-running GraphQL enough time (adjust as needed)
		runCtx, cancelRun := context.WithTimeout(rootCtx, 10*time.Minute)
		defer cancelRun()

		if err := fetchPiapIndicatorProjectionsByFiscalYear(runCtx, cfg, mappingCache, client, fy); err != nil {
			log.Fatalf("pbs-sync: fetch error: %v", err)
		}
		log.Println("pbs-sync: single run completed (Sync.Once=true)")

		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		//if err := fetchOutturns(ctx, client, fy); err != nil {
		//	log.Printf("pbs-sync: fetch error: %v", err)
		//}
		runCtx, cancelRun := context.WithTimeout(rootCtx, 10*time.Minute)
		if err := fetchPiapIndicatorProjectionsByFiscalYear(runCtx, cfg, mappingCache, client, fy); err != nil {
			log.Printf("pbs-sync: fetch error: %v", err)
		}

		cancelRun()

		select {
		case <-rootCtx.Done():
			log.Println("pbs-sync: shutting down")
			return
		case <-ticker.C:
		}
	}
}

func resolveCacheDir(dir string, appName string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	// Default: ~/.cache/<appName>
	if dir == "" {
		return filepath.Join(base, appName), nil
	}

	// Absolute path provided
	if filepath.IsAbs(dir) {
		return dir, nil
	}

	// Relative path -> place under user cache dir
	return filepath.Join(base, dir), nil
}

func fetchOutturns(ctx context.Context, cfg config.Config, client *pbs.Client, fy string) error {
	log.Infof("pbs-sync: fetching outturns")
	resp, err := pbs.CgBudgetOutturnsByFiscalYear(ctx, client.Gql(), fy)
	if err != nil {
		return err
	}
	log.Infof("pbs-sync: got %d outturn rows", len(resp.CgBudgetOutturnByFiscalYear))
	for _, row := range resp.CgBudgetOutturnByFiscalYear {
		log.Infof("VoteCode=%s Vote=%s FY=%s Prog=%s Approved=%.2f Q1Exp=%.2f",
			row.Vote_Code, row.Vote_Name, row.Fiscal_Year, row.Programme_Name, row.ApprovedBudget, row.Q1Expenditure)
		dvs, dvsError := BuildPBSDataValues2(row, &cfg)
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
		batchSize := 5
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

	}
	return nil
}

func ForEachBatch[T any](slice []T, batchSize int, callback func(batch []T) error) error {
	for i := 0; i < len(slice); i += batchSize {
		end := i + batchSize
		if end > len(slice) {
			end = len(slice)
		}
		batch := slice[i:end]
		err := callback(batch)
		if err != nil {
			return err
		}
	}
	return nil
}

func fetchPiapIndicatorProjectionsByFiscalYear(
	ctx context.Context, cfg config.Config, mappingsCache *mappings.MappingCache, client *pbs.Client, fy string) error {
	log.Infof("pbs-sync: fetching piap indicator projections for year %s", fy)
	//resp, err := pbs.CgPiapIndicatorProjectionsByFiscalYear(ctx, client.Gql(), fy)
	//if err != nil {
	//	log.Errorf("pbs-sync: CgPiapIndicatorProjectionsByFiscalYear fetch error: %v", err)
	//	return err
	//}
	resp, err := GetIndicatorProjections(ctx, cfg, client, fy) // XXX change this true
	if err != nil {
		log.Errorf("pbs-sync: failed to get indicator projections: %v", err)
	}
	if resp == nil {
		return fmt.Errorf("GetIndicatorPorjections yields a nil response (fy=%s)", fy)
	}
	log.Infof("pbs-sync: fetched %d piap indicator projections.", len(resp))

	for _, row := range resp {
		// log.WithFields(log.Fields{"PiapIndicatortProjection": row}).Info("pbs-sync: piap indicator projection")
		dvs, err2 := BuildPiapIndicatorProjectsDataValues(row, &cfg, mappingsCache)
		if err2 != nil {
			log.Warnf("pbs-sync: failed to build data values for piap indicator projection: %v", err2)
			continue
		}
		if len(dvs) == 0 {
			log.Warnf("pbs-sync: no data values generated for piap indicator projection: %v", row)
			continue
		}
		log.Infof("pbs-sync: %d data values generated for piap indicator projection: %v", len(dvs), dvs)
		for _, dv := range dvs {
			log.WithFields(log.Fields{
				"dataElement":         *dv.DataElement,
				"period":              *dv.Period,
				"orgUnit":             *dv.OrgUnit,
				"categoryOptionCombo": *dv.CategoryOptionCombo,
				"categoryCombo":       *dv.CategoryCombo,
				"categoryOption":      *dv.CategoryOption,
				// "value":               *dv.Value,
			}).Info("pbs-sync: data value")
		}

		_ = ForEachBatch(dvs, dvBatchSize, func(batch []ExtendedDataValue) error {
			if err := PushIndividualDataValues(batch); err != nil {
				log.Errorf("pbs-sync: push data values error: %v", err)
				return nil // swallow error to "continue" like before
			}
			log.Infof("pbs-sync: pushed %d data values to DHIS2", len(batch))
			return nil
		})

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

type ProjectionsDTO = pbs.CgPiapIndicatorProjectionsByFiscalYearCgPiapIndicatorProjectionsByFiscalYearOpmCgPiapIndicatorProjectionsDto

func BuildPiapIndicatorProjectsDataValues(
	row ProjectionsDTO,
	cfg *config.Config,
	mappingsCache *mappings.MappingCache,
) ([]ExtendedDataValue, error) {
	var dvs []ExtendedDataValue
	ouMapping, err := models.GetOrgUnitMapping(row.Vote_Code, cfg.PBS.InstanceName)
	if err != nil {
		return nil, fmt.Errorf(
			"no org unit mapping for Vote Code: %s, Vote Name: %s for source_name: %s",
			row.Vote_Code, row.Vote_Name, cfg.PBS.InstanceName)
	}

	deMapping, ok := mappingsCache.Get(row.PIAP_Output_Indicator_Code)
	if !ok {
		log.Warnf("missing mapping for code %s", row.PIAP_Output_Indicator_Code)
		return nil, fmt.Errorf("missing mapping for code %s", row.PIAP_Output_Indicator_Code)
	}

	// Helper to get combo UID with default fallback
	getComboUID := func(baseKey string) config.DHIS2CategoryOptionCombo {
		conf := config.DHIS2CategoryOptionCombo{}
		if combo, ok := cfg.PBS.CategoryOptionCombos[baseKey]; ok && combo.UID != "" {

			return combo
		}
		return conf
	}

	// Table-driven for quarters
	type qrow[T float64 | string] struct {
		qName              string
		actual             T
		reasonForVariation string
	}
	quarters := []qrow[string]{
		{qName: "Q1", actual: row.Q1_Actual_Target, reasonForVariation: row.Q1_Reason_For_Variation},
		{qName: "Q2", actual: row.Q2_Cum_Performance, reasonForVariation: row.Q2_Reason_For_Variation},
		{qName: "Q3", actual: row.Q3_Cum_Performance, reasonForVariation: row.Q3_Reason_For_Variation},
		{qName: "Q4", actual: row.Q4_Cum_Performance, reasonForVariation: row.Q4_Reason_For_Variation},
	}

	for _, q := range quarters {
		period := deriveQuarterPeriod(row.Fiscal_Year, q.qName)
		appendDV(
			"cg_piap_indicator_projections_actual", period, q.actual, q.reasonForVariation, &dvs, deMapping, ouMapping, getComboUID)
		//if q.reasonForVariation != "" {
		//	// we add the reason for variation
		//}
		log.WithFields(log.Fields{"PBS_QUATER": q.qName, "PERIOD": period, "Year": row.Fiscal_Year}).Info("Period Information")
	}

	// Approved Budget (annual, July period of first FY year)
	if row.Target_Y1 != "" {
		// targetY1, err := strconv.ParseFloat(row.Target_Y1, 64)
		//if err != nil {
		//	return nil, err
		//}
		//if targetY1 != 0 {
		year := strings.Split(row.Fiscal_Year, "-")[0]
		period := strings.TrimSpace(fmt.Sprintf("%sJuly", year))
		appendDV("cg_piap_indicator_projections_target_y1", period, row.Target_Y1, "", &dvs, deMapping, ouMapping, getComboUID)
		//}
	}
	log.WithFields(log.Fields{"DATAVALUES": dvs}).Debug("The data values to push")

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

func appendDV[T float64 | string](
	baseKey string,
	period string,
	v T,
	comment string,
	dvs *[]ExtendedDataValue,
	deMapping *models.Dhis2Mapping,
	ouMapping string,
	getComboUID func(string) config.DHIS2CategoryOptionCombo,
) {
	var val string

	switch x := any(v).(type) {
	case float64:
		if x == 0 {
			return
		}
		val = fmt.Sprintf("%.0f", x)

	case string:
		if x == "" {
			return
		}
		val = x
	}

	coc := getComboUID(baseKey)
	dv := ExtendedDataValue{
		DataValue: schema.DataValue{
			DataElement:          &deMapping.DataElement,
			AttributeOptionCombo: &coc.UID,
			CategoryOptionCombo:  &coc.UID,
			Period:               &period,
			OrgUnit:              &ouMapping,
			Value:                &val,
		},
		CategoryCombo:  &coc.Combo,
		CategoryOption: &coc.Option,
	}

	if comment != "" {
		commentDv := ExtendedDataValue{
			DataValue: schema.DataValue{
				DataElement:          &deMapping.DataElement,
				AttributeOptionCombo: &coc.UID,
				CategoryOptionCombo:  &coc.UID,
				Period:               &period,
				OrgUnit:              &ouMapping,
				Comment:              &comment,
			},
			CategoryCombo:  &coc.Combo,
			CategoryOption: &coc.Option,
		}
		*dvs = append(*dvs, commentDv)
		// dv.Comment = &comment
	}
	*dvs = append(*dvs, dv)
}
