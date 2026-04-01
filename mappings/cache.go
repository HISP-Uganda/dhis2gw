package mappings

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"

	"dhis2gw/models"
)

type MappingCache struct {
	db           *sqlx.DB
	instanceName string

	mu    sync.RWMutex
	items map[string]*models.Dhis2Mapping
}

func NewMappingCache(db *sqlx.DB, instanceName string) *MappingCache {
	return &MappingCache{
		db:           db,
		instanceName: instanceName,
		items:        make(map[string]*models.Dhis2Mapping),
	}
}

func normalize(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

// Load loads mappings from DB and replaces the cache atomically.
func (c *MappingCache) Load(ctx context.Context) error {

	const q = `
		SELECT *
		FROM dhis2_mappings
		WHERE source_name = $1
		AND what = 'de'
	`

	var rows []models.Dhis2Mapping

	if err := c.db.SelectContext(ctx, &rows, q, c.instanceName); err != nil {
		return fmt.Errorf("load dhis2 mappings: %w", err)
	}

	next := make(map[string]*models.Dhis2Mapping, len(rows))

	for i := range rows {

		key := normalize(rows[i].Code)

		if _, exists := next[key]; exists {
			return fmt.Errorf(
				"duplicate mapping detected code=%s instance=%s",
				rows[i].Code,
				rows[i].InstanceName,
			)
		}

		next[key] = &rows[i]
	}

	c.mu.Lock()
	c.items = next
	c.mu.Unlock()

	return nil
}

func (c *MappingCache) Reload(ctx context.Context) error {
	return c.Load(ctx)
}

func (c *MappingCache) Get(code string) (*models.Dhis2Mapping, bool) {

	key := normalize(code)

	c.mu.RLock()
	m, ok := c.items[key]
	c.mu.RUnlock()

	return m, ok
}

func (c *MappingCache) MustGet(code string) (*models.Dhis2Mapping, error) {

	m, ok := c.Get(code)

	if !ok {
		return nil, fmt.Errorf(
			"mapping not found code=%s instance=%s",
			code,
			c.instanceName,
		)
	}

	return m, nil
}

func (c *MappingCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

func (c *MappingCache) StartAutoReload(ctx context.Context, interval time.Duration) {

	ticker := time.NewTicker(interval)

	go func() {

		defer ticker.Stop()

		for {

			select {

			case <-ctx.Done():
				return

			case <-ticker.C:

				if err := c.Reload(ctx); err != nil {
					log.Warnf("mapping cache reload failed: %v", err)
				}
			}
		}
	}()
}
