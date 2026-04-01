package main

import (
	"context"
	"crypto/sha256"
	"dhis2gw/config"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"dhis2gw/clients/pbs"
)

type CacheMeta struct {
	Operation string          `json:"operation"`
	Variables json.RawMessage `json:"variables"`
	Endpoint  string          `json:"endpoint"`
	SchemaTag string          `json:"schemaTag,omitempty"`
	// AppTag    string          `json:"appTag,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type CacheFile struct {
	Meta CacheMeta       `json:"meta"`
	Data json.RawMessage `json:"data"`
}

// Canonical JSON for variables: stable key ordering.
func canonicalVars(vars map[string]any) ([]byte, error) {
	// Encode with stable ordering by sorting keys and building a new map in order.
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make(map[string]any, len(vars))
	for _, k := range keys {
		ordered[k] = vars[k]
	}
	return json.Marshal(ordered)
}

func cacheKey(cfg config.Config, operation string, canonicalVarsJSON []byte) []byte {
	// Keep it simple and stable.
	// If you want, you can also add a "query signature" / hash of the actual GraphQL query text.
	return []byte(fmt.Sprintf(
		"op=%s|vars=%s|endpoint=%s|schema=%s",
		operation, canonicalVarsJSON, cfg.PBS.Cache.Endpoint, cfg.PBS.Cache.SchemaTag,
	))
}

func keyHash12(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:12]
}

func cachePaths(cfg config.Config, operation string, vars map[string]any) (dir, dataPath string, metaPath string, err error) {
	cvars, err := canonicalVars(vars)
	if err != nil {
		return "", "", "", err
	}
	h := keyHash12(cacheKey(cfg, operation, cvars))

	// human part: if fiscalYear exists, put it in the filename
	human := "vars"
	if fy, ok := vars["fiscalYear"]; ok {
		human = fmt.Sprintf("fy-%v", fy)
	}

	dir = filepath.Join(cfg.PBS.Cache.CacheDir, "graphql", operation)
	base := fmt.Sprintf("%s__%s", human, h)

	dataPath = filepath.Join(dir, base+".json")      // full envelope (meta + data)
	metaPath = filepath.Join(dir, base+".meta.json") // optional separate meta (handy for grepping)
	return dir, dataPath, metaPath, nil
}

func cacheFile(cfg config.Config, operation string, fiscalYear string) string {
	dir := fmt.Sprintf("%s/graphql/%s", cfg.PBS.Cache.CacheDir, operation)
	os.MkdirAll(dir, 0755)
	return fmt.Sprintf("%s/fy_%s.json", dir, fiscalYear)
}

func isFresh(path string, ttl time.Duration) bool {
	if ttl <= 0 {
		return true
	}
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(st.ModTime()) <= ttl
}

func readCache(path string) (CacheFile, error) {
	var cf CacheFile
	b, err := os.ReadFile(path)
	if err != nil {
		return cf, err
	}
	if err := json.Unmarshal(b, &cf); err != nil {
		return cf, err
	}
	return cf, nil
}

func writeCache(dir, dataPath, metaPath string, cf CacheFile) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp := dataPath + ".tmp"
	b, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, dataPath); err != nil {
		return err
	}

	// optional meta file (nice for inspection)
	mb, _ := json.MarshalIndent(cf.Meta, "", "  ")
	_ = os.WriteFile(metaPath, mb, 0o644)
	return nil
}

// Fetcher is your actual GraphQL call. It should return the raw JSON response payload for the operation.
// If you're using genqlient, you can marshal the typed response to JSON instead.
type Fetcher func(ctx context.Context) (json.RawMessage, error)

// GetOrFetch wraps: read cache (if allowed + fresh) else fetch and cache.
func GetOrFetch(ctx context.Context, cfg config.Config, operation string, vars map[string]any, fetch Fetcher) (json.RawMessage, bool, error) {
	if !cfg.PBS.Cache.Enabled {
		data, err := fetch(ctx)
		return data, false, err
	}

	dir, dataPath, metaPath, err := cachePaths(cfg, operation, vars)
	if err != nil {
		return nil, false, err
	}

	// 1) Use cache if exists and fresh
	if isFresh(dataPath, cfg.PBS.Cache.TTL) {
		cf, err := readCache(dataPath)
		if err == nil && len(cf.Data) > 0 {
			return cf.Data, true, nil
		}
	}

	// 2) If cache-only mode, fail here
	if cfg.PBS.Cache.UseCacheOnly {
		return nil, false, fmt.Errorf("cache miss or stale: %s", dataPath)
	}

	// 3) Fetch from network
	data, err := fetch(ctx)
	if err != nil {
		// Optional: if fetch fails but cache exists (even stale), fall back for dev.
		if cf, rerr := readCache(dataPath); rerr == nil && len(cf.Data) > 0 {
			return cf.Data, true, nil
		}
		return nil, false, err
	}

	cvars, _ := canonicalVars(vars)
	cf := CacheFile{
		Meta: CacheMeta{
			Operation: operation,
			Variables: cvars,
			Endpoint:  cfg.PBS.Cache.Endpoint,
			SchemaTag: cfg.PBS.Cache.SchemaTag,
			// AppTag:    cfg.AppTag,
			CreatedAt: time.Now(),
		},
		Data: data,
	}
	if werr := writeCache(dir, dataPath, metaPath, cf); werr != nil {
		// don’t fail the request just because caching failed
	}
	return data, false, nil
}

// ErrCacheDisabled Helper if you want strict error when caching is enabled but cache dir missing/misconfigured:
var ErrCacheDisabled = errors.New("cache disabled")

func MustCacheDir(cfg config.Config) error {
	if !cfg.PBS.Cache.Enabled {
		return ErrCacheDisabled
	}
	if cfg.PBS.Cache.CacheDir == "" {
		return fmt.Errorf("cache dir not set")
	}
	return nil
}

func GetIndicatorProjections(
	ctx context.Context,
	cfg config.Config,
	client *pbs.Client,
	fiscalYear string,
) ([]ProjectionsDTO, error) {

	file := cacheFile(cfg, "CgPiapIndicatorProjectionsByFiscalYear", fiscalYear)

	// 1️⃣ Load cache if available
	if cfg.PBS.Cache.UseCacheOnly {

		if _, err := os.Stat(file); err == nil {

			// var cached pbs.CgPiapIndicatorProjectionsByFiscalYearResponse
			var cached struct {
				Data struct {
					CgPiapIndicatorProjectionsByFiscalYear []ProjectionsDTO `json:"cgPiapIndicatorProjectionsByFiscalYear"`
				} `json:"data"`
			}
			data, err := os.ReadFile(file)
			if err != nil {
				return nil, err
			}

			if err := json.Unmarshal(data, &cached); err != nil {
				return nil, err
			}

			fmt.Println("Loaded cached data:", file)

			return cached.Data.CgPiapIndicatorProjectionsByFiscalYear, nil
		}
	}

	// 2️⃣ Fetch from GraphQL
	fmt.Println("Fetching from GraphQL...")

	resp, err := pbs.CgPiapIndicatorProjectionsByFiscalYear(ctx, client.Gql(), fiscalYear)
	if err != nil {
		return nil, err
	}

	// 3️⃣ Save to cache
	if cfg.PBS.Cache.UseCacheOnly {

		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return nil, err
		}

		err = os.WriteFile(file, data, 0644)
		if err != nil {
			fmt.Println("Warning: failed to write cache:", err)
		} else {
			fmt.Println("Cached result:", file)
		}
	}

	return resp.CgPiapIndicatorProjectionsByFiscalYear, nil
}
