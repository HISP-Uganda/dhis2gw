package pbs

import "time"

// ------------------------------
// GraphQL response wire models
// ------------------------------

// ChangesResponse matches `query Changes` (Relay connection).
type ChangesResponse struct {
	Changes struct {
		Edges []struct {
			Cursor string     `json:"cursor"`
			Node   ChangeNode `json:"node"`
		} `json:"edges"`
		PageInfo PageInfo `json:"pageInfo"`
	} `json:"changes"`
}

// ChangesByIDsResponse matches `query ChangesByIDs`.
type ChangesByIDsResponse struct {
	ChangesByIDs []ChangeNode `json:"changesByIds"`
}

// PingResponse matches `query Ping`.
type PingResponse struct {
	Ping string `json:"ping"`
}

// PageInfo is the Relay page info.
type PageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// ------------------------------
// Domain nodes (GraphQL → Go)
// ------------------------------

// ChangeNode mirrors fields returned by the fragment `ChangeMin`.
// Extend with the fields you add in the fragment/query.
type ChangeNode struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	ExternalRef string `json:"externalRef"`
	UpdatedAt   string `json:"updatedAt"` // RFC3339 string from API

	// --- Add domain fields you need for DHIS2 mapping (uncomment/adjust) ---
	// OrgUnitCode     string  `json:"orgUnitCode"`
	// DataElementCode string  `json:"dataElementCode"`
	// Period          string  `json:"period"`
	// Value           string  `json:"value"`
	// NumericValue    float64 `json:"numericValue"`
}

// ToChangeItem converts a ChangeNode into your internal normalized struct.
// Add/adjust mappings as your schema/needs evolve.
func (n ChangeNode) ToChangeItem() ChangeItem {
	return ChangeItem{
		ID:          n.ID,
		Kind:        n.Kind,
		ExternalRef: n.ExternalRef,
		UpdatedAt:   n.UpdatedAt,
		// Map domain fields here as you add them to ChangeNode:
		// OrgUnitCode:     n.OrgUnitCode,
		// DataElementCode: n.DataElementCode,
		// Period:          n.Period,
		// Value:           n.Value,
		// NumericValue:    n.NumericValue,
	}
}

// UpdatedAtTime parses UpdatedAt when you need a time.Time.
func (n ChangeNode) UpdatedAtTime() (time.Time, error) {
	return time.Parse(time.RFC3339, n.UpdatedAt)
}

// ------------------------------
// Internal normalized model
// ------------------------------

// ChangeItem is what your integration/mapping code uses.
// Keep it stable even if the GraphQL schema evolves.
type ChangeItem struct {
	ID          string
	Kind        string
	ExternalRef string
	UpdatedAt   string // keep original string for logging; parse when needed

	// Normalized fields for DHIS2 mapping (extend as required):
	// OrgUnitCode     string
	// DataElementCode string
	// Period          string
	// Value           string
	// NumericValue    float64
}
