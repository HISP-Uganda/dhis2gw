package pbs

// NOTE: Adjust field names to match your PBS schema.
// Common scalars seen in GraphQL servers: DateTime, UUID, ID, String, Int, etc.

// Minimal fields every change should return.
// Extend this fragment with whatever you need to map into DHIS2.
const fragmentChangeMin = `
fragment ChangeMin on Change {
  id
  kind
  externalRef
  updatedAt
  # Add domain fields needed by your mapping, e.g.:
  # orgUnitCode
  # dataElementCode
  # period
  # value
}
`

// Windowed changes with Relay-style pagination.
// Variables:
//
//	$since: inclusive lower bound (RFC3339 DateTime)
//	$until: exclusive/ inclusive per your API contract (RFC3339 DateTime)
//	$first: page size
//	$after: cursor for pagination
const queryChanges = fragmentChangeMin + `
query Changes($since: DateTime!, $until: DateTime!, $first: Int!, $after: String) {
  changes(since: $since, until: $until, first: $first, after: $after) {
    edges {
      cursor
      node {
        ...ChangeMin
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}
`

// Fetch specific changes by IDs (useful for backfills / replays).
// Variables:
//
//	$ids: list of IDs (adjust type if your server uses UUID!)
const queryChangesByIDs = fragmentChangeMin + `
query ChangesByIDs($ids: [ID!]!) {
  changesByIds(ids: $ids) {
    ...ChangeMin
  }
}
`

// Lightweight connectivity/auth check.
// If your schema doesn’t have `ping`, replace with a trivial query you know exists.
const queryPing = `
query Ping {
  ping
}
`
