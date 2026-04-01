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

The IBP sync uses its own configuration structure (typically defined in `cmd/ibp/config.go` and loaded from a JSON/YAML file or environment).

| Key | Description |
| :--- | :--- |
| `server.base_url` | IBP API Base URL. |
| `server.username` | IBP Username. |
| `server.password` | IBP Password. |
| `dhis2_url` | Target DHIS2 instance URL. |
| `dhis2_user` | DHIS2 Username. |
| `dhis2_password` | DHIS2 Password. |
| `server.data_sync_cron_expression` | Cron expression for sync schedule (e.g., `0 */12 * * *`). |

## Running the IBP Sync

To run the IBP synchronization service:

```bash
# Run directly via Go
go run cmd/ibp/main.go

# Or build and run
go build -o dist/ibp cmd/ibp/main.go
./dist/ibp
```

The service will start, authenticate with both systems, verify the DHIS2 program configuration, and then enter a loop waiting for the scheduled cron time to trigger the synchronization.