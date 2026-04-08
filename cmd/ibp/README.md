# IBP Sync (`cmd/ibp`)

The `ibp` command handles the synchronization of project data from the **Integrated Bank of Projects (IBP)** to DHIS2. Unlike the PBS sync which handles aggregate budget data, this component deals with tracker data (Projects).

## Features
- **Source**: Integrated Bank of Projects (IBP) API.
- **Destination**: DHIS2 Tracker Program.
- **Functionality**:
  - Fetches project lists and details from IBP.
  - Maps IBP project fields to DHIS2 Tracked Entity Attributes.
  - Creates or updates Tracked Entity Instances (TEIs) in DHIS2.
  - Supports periodic synchronization via cron expressions.

## Configuration

The IBP sync loads its configuration from:

```text
/etc/dhis2gw/conf.d/ibp.yaml
```

The command also merges environment overrides at runtime.

### Supported Configuration

```yaml
Debug: false

Server:
  BaseURL: https://example.ibp.local
  Username: ibp-user
  Password: ibp-password
  JWTSecret: change-me
  TimeoutSeconds: 10
  DataSyncCronExpression: "0 */12 * * *"

program:
  OrganisationUnitPath: projectOrganization.parent.code
  SearchAttribute: SOME_DHIS2_ATTRIBUTE_UID
  ProgramID: DHIS2_PROGRAM_UID
  TrackedEntityType: DHIS2_TRACKED_ENTITY_TYPE_UID
  TrackedEntityAttributes:
    ATTRIBUTE_UID: projectTitle
    ANOTHER_ATTRIBUTE_UID: projectCode
  Stages:
    registration:
      programStage: DHIS2_PROGRAM_STAGE_UID
      dataValues:
        DATA_ELEMENT_UID: startDate

Defaults:
  xR2SRSZxDSl: "true"
```

### Configuration Notes

- `Server.BaseURL`, `Server.Username`, and `Server.Password` are used to authenticate against the IBP API.
- `Server.TimeoutSeconds` controls the HTTP client timeout. If omitted, it defaults to `10`.
- `Server.DataSyncCronExpression` defines when the scheduled sync runs.
- `program.SearchAttribute` must match a key in `program.TrackedEntityAttributes`. It is used to search for an existing tracked entity before create or update.
- `program.TrackedEntityAttributes` maps DHIS2 tracked entity attribute UIDs to GJSON paths in the IBP project payload.
- `program.Stages` maps logical stage aliases to DHIS2 program stage UIDs and their data element mappings.
- `Defaults` is used to fill missing mandatory DHIS2 values when they are not present in the IBP payload.

### DHIS2 Connection

Although the config struct includes `dhis2_url`, `dhis2_user`, and `dhis2_password`, this command currently derives the DHIS2 connection from the main application runtime configuration. In practice, the DHIS2 credentials are inherited from the base `dhis2gw` config rather than being controlled by `ibp.yaml`.

| Key | Description |
| :--- | :--- |
| `Debug` | Enables debug-oriented configuration values when needed. |
| `Server.BaseURL` | IBP API base URL. |
| `Server.Username` | IBP username. |
| `Server.Password` | IBP password. |
| `Server.JWTSecret` | JWT secret value from the IBP service configuration. |
| `Server.TimeoutSeconds` | HTTP timeout in seconds. Defaults to `10`. |
| `Server.DataSyncCronExpression` | Cron expression for sync schedule. |
| `program.OrganisationUnitPath` | Source path used to determine the IBP organisation unit value. |
| `program.SearchAttribute` | DHIS2 attribute UID used to search for existing tracked entities. |
| `program.ProgramID` | Target DHIS2 tracker program UID. |
| `program.TrackedEntityType` | Target DHIS2 tracked entity type UID. |
| `program.TrackedEntityAttributes` | Map of DHIS2 attribute UIDs to IBP payload paths. |
| `program.Stages` | Map of stage aliases to DHIS2 program stage definitions and data element mappings. |
| `Defaults` | Fallback values for required DHIS2 attributes or elements when source data is missing. |

## Running the IBP Sync

To run the IBP synchronization service:

```bash
# Run directly via Go
go run cmd/ibp/main.go

# Or build and run
go build -o dist/ibp cmd/ibp/main.go
./dist/ibp
```

The service will:

- initialize shared runtime dependencies such as the database and cached server metadata
- authenticate with the IBP API
- authenticate with DHIS2 using the base application configuration
- load the DHIS2 program and tracked entity type metadata used for validation
- schedule project synchronization using the configured cron expression
- start an HTTP server on port `8484`

### HTTP Endpoints

- `GET /api/test`
- `GET /api/data`

Both endpoints are exposed under basic authentication.
