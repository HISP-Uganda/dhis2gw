# DHIS2 Gateway (dhis2gw)

This project serves as a middleware gateway to synchronize data between external systems and DHIS2.

## Components

### PBS Sync (`cmd/pbs_sync`)

`pbs-sync` is a specialized command-line tool designed to fetch budget outturn data from the **Program Budgeting System (PBS)** and push it into **DHIS2**.

#### Features
- **Authentication**: Supports both Username/Password and Static JWT authentication with the PBS API.
- **Data Mapping**:
  - Maps PBS `Vote_Code` to DHIS2 Organisation Units.
  - Maps PBS `Item_Code` to DHIS2 Data Elements.
- **Period Conversion**: Automatically converts Fiscal Years (e.g., "2025-2026") and Quarters (Q1-Q4) into standard DHIS2 quarters based on a July-start fiscal year.
- **Execution Modes**: Can run as a one-off task or a persistent daemon with configurable intervals.

#### Key Technologies & Libraries

The project leverages several key Go libraries to handle configuration, API communication, and logging:

- **GraphQL Client**: [Khan/genqlient](https://github.com/Khan/genqlient)
  - Used for type-safe interaction with the PBS GraphQL API. It generates Go code from your GraphQL queries, ensuring compile-time safety for API requests.
- **DHIS2 SDK**: `github.com/HISP-Uganda/go-dhis2-sdk`
  - Provides the data structures (`schema.DataValue`) and client logic necessary to communicate with the DHIS2 Web API.
- **Logging**: [sirupsen/logrus](https://github.com/sirupsen/logrus)
  - Used for structured logging throughout the application, providing clear info, warning, and error levels.
- **Configuration**: [spf13/viper](https://github.com/spf13/viper)
  - Manages application configuration, allowing settings to be read from JSON/YAML files (like `config.yaml`) or environment variables.

#### Configuration

The tool relies on the global application configuration. Key PBS settings include:

| Key | Environment Variable | Description |
| :--- | :--- | :--- |
| `pbs.pbs_url` | `PBS_URL` | The base URL for the PBS GraphQL API. |
| `pbs.user` | `PBS_USER` | Username for PBS authentication. |
| `pbs.password` | `PBS_PASSWORD` | Password for PBS authentication. |
| `pbs.fiscal_year` | `PBS_FISCAL_YEAR` | The target fiscal year to fetch (e.g., "2025-2026"). |
| `pbs.sync.once` | `PBSSYNC_ONCE` | If `true`, runs once and exits. If `false`, runs periodically. |
| `pbs.sync.interval` | `PBSSYNC_INTERVAL` | Duration string (e.g., "1h", "24h") for sync frequency. |

#### Running the Command

You can run the tool using a configuration file or by setting environment variables.

**Method 1: Using a Config File**

1.  Create a `dhis2gw.yml` file (or use the default in `/etc/dhis2gw/` on Linux).
2.  Run pointing to that file:

```bash
go run cmd/pbs_sync/main.go --config-file /path/to/your/dhis2gw.yml
```

**Method 2: Using Environment Variables (One-off Run)**

This example runs a single synchronisation for the fiscal year 2024-2025 and then exits.

```bash
export PBS_URL="https://api.pbs.go.ug/graphql"
export PBS_USER="admin"
export PBS_PASSWORD="secure_password"
export PBS_FISCAL_YEAR="2024-2025"
export PBSSYNC_ONCE="true"

go run cmd/pbs_sync/main.go
```

**Method 3: Running as a Daemon**

This example runs the synchronizer continuously, fetching data every 6 hours.

```bash
export PBS_URL="https://api.pbs.go.ug/graphql"
export PBS_USER="admin"
export PBS_PASSWORD="secure_password"
export PBSSYNC_ONCE="false"
export PBSSYNC_INTERVAL="6h"

# Build first for production use
go build -o dist/pbs-sync cmd/pbs_sync/main.go

# Run the binary
./dist/pbs-sync
```

#### Deployment (Debian/Systemd)

This project is packaged as a Debian package that installs the `pbs-sync` binary and configures it to run as a systemd service.

**Installation**

After installing the `.deb` package, the binary is placed in `/usr/bin/pbs-sync` and a default configuration file is created at `/etc/dhis2gw/dhis2gw.yml`.

**Service Management**

The package installs a systemd service named `pbs-sync.service`.

**Start the service**
```bash
sudo systemctl start pbs-sync
```
**Stop the service**
```bash
sudo systemctl stop pbs-sync
```
**Restart the service**
```bash
sudo systemctl restart pbs-sync
```


**View service status**
```bash
sudo systemctl status pbs-sync
```

**View logs**
```bash
journalctl -u pbs-sync -f
```
