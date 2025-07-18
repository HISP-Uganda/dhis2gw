# DHIS2 Gateway Service (dhis2gw)

## Overview

**dhis2gw** is a robust gateway application for sending aggregate and tracker data to DHIS2 from third-party systems. Built with Go and Gin, it leverages asynchronous processing (Asynq) and PostgreSQL for task management and logging. It also provides a KivyMD desktop UI for administrators to monitor, requeue, and manage tasks.

---

## Table of Contents

- [Tech Stack](#tech-stack)
- [Features](#features)
- [API Endpoints](#api-endpoints)
- [Logging & Monitoring](#logging--monitoring)
- [KivyMD Interface](#kivymd-interface)
- [Architecture Diagram](#architecture-diagram)
- [Installation Guide](#installation-guide)
- [Configuration Reference](#configuration-reference)
- [Support](#support)

---

## Tech Stack

- **Backend:** Go, Gin
- **Async Task Processing:** Asynq
- **Database:** PostgreSQL
- **API Documentation:** gin-swagger
- **Desktop UI:** KivyMD (Python)
- **Other:** JWT for authentication, custom logging to PostgreSQL

---

## Features

- **Secure User Management:** Token-based authentication with JWT, user CRUD.
- **Asynchronous Data Submission:** Handles DHIS2 aggregate/tracker data via background jobs.
- **Comprehensive Logging:** Every API request and background task is logged in PostgreSQL.
- **Admin Interface:** KivyMD app for monitoring, filtering, requeuing, and deleting jobs.
- **Easy DHIS2 Mapping Import:** Import mappings via CSV/Excel.
- **Interactive API Docs:** Live documentation via Swagger.

---

## API Endpoints

### User Management

| Endpoint         | Method | Description           |
|------------------|--------|-----------------------|
| `/user`          | POST   | Create a new user     |
| `/users`         | GET    | List users (with filters) |
| `/users/:uid`    | GET    | Get user by UID       |
| `/users/:uid`    | PUT    | Update user by UID    |

### Authentication

| Endpoint              | Method | Description                  |
|-----------------------|--------|------------------------------|
| `/users/getToken`     | POST   | Obtain JWT token (login)     |
| `/users/refreshToken` | POST   | Refresh JWT token            |

### Aggregate Data Submission

| Endpoint      | Method | Description                   |
|---------------|--------|-------------------------------|
| `/aggregate`  | POST   | Submit aggregate values to DHIS2 |

### Logs Management

| Endpoint     | Method | Description                                |
|--------------|--------|--------------------------------------------|
| `/logs`      | GET    | List logs, filterable by date/status       |
| `/logs/:id`  | GET    | Get details of a log/task by ID            |

### DHIS2 Mappings

| Endpoint                  | Method | Description                        |
|---------------------------|--------|------------------------------------|
| `/mappings`               | GET    | List all DHIS2 data mappings       |
| `/mappings/import/csv`    | POST   | Import DHIS2 mappings via CSV      |
| `/mappings/import/excel`  | POST   | Import DHIS2 mappings via Excel    |

### Swagger Documentation

| Endpoint         | Method | Description                     |
|------------------|--------|---------------------------------|
| `/swagger/*any`  | GET    | Access Swagger UI/API docs      |

---

## Logging & Monitoring

- **All** incoming requests and background jobs are logged to a custom PostgreSQL table.
- **Fields tracked:** ID, Time, User, Endpoint, Payload, Status, Error (if any), Retry Count, etc.
- **Integration:** Both synchronous (API) and asynchronous (Asynq tasks) events are recorded for full traceability.

---

## KivyMD Interface

- **Purpose:** Provides an admin panel for system operators.
- **Features:**
  - Filter logs by status, date, user, endpoint, etc.
  - View detailed payload and error information.
  - Requeue failed jobs with one click.
  - Delete or purge logs/tasks as needed.

---

## Architecture Diagram

```plaintext
+------------------+           +--------------------------+         +-------------------+
|                  |  HTTP API |                          |  Asynq  |                   |
|   Third-party    +----------->     dhis2gw (Gin)        +--------->   Asynq Workers    |
|   Systems        |           |                          |         |                   |
+------------------+           +------+----+----+---------+         +-------------------+
                                       |    |    |
          +-------------------+        |    |    |
          |                   |        |    |    |      +--------------------+
          |    KivyMD         |        |    |    +----->|  DHIS2 API         |
          |  Admin App        +<-------+    |           +--------------------+
          |  (Python)         |     REST    |
          +-------------------+             |
                             +--------------v--------------+
                             |   PostgreSQL (DB)            |
                             |  (users, logs, mappings)     |
                             +------------------------------+
```

---

## 📦 Installation Guide

Refer to the full [installation guide with steps and prerequisites](#installation-guide) and [configuration reference](#configuration-reference) below.

---

### 🛠️ Prerequisites

- **PostgreSQL** >= 13
- **Redis** >= 5
- **wget** or **curl** (for downloading packages)
- **dpkg** (for Debian packages)
- **systemd** (for service management)

Create the dhis2gw database if not present:

```bash
 sudo -u postgres psql -c "CREATE DATABASE dhis2gw;"
```

---

### 📥 Step 1: Download the Debian Package

```bash
 wget https://github.com/HISP-Uganda/dhis2gw/releases/download/vX.Y.Z/dhis2gw_X.Y.Z_amd64.deb
```

Replace `X.Y.Z` with the latest version.

---

### 📦 Step 2: Install the Package

```bash
sudo dpkg -i dhis2gw_X.Y.Z_amd64.deb
sudo apt-get install -f   # resolve deps if needed
```

---

### ⚙️ Step 3: Configuration

Default config file:

```
/etc/dhis2gw/dhis2gw.yml
```

You can override any settings via environment variables. See full config table in `dhis2gw_config.md`.

---

### 🧪 Step 4: Test Installation

```bash
/usr/bin/dhis2gw --config /etc/dhis2gw/dhis2gw.yml
```

---

### 🚀 Step 5: Run as a Service

```bash
sudo systemctl enable dhis2gw
sudo systemctl start dhis2gw
sudo systemctl status dhis2gw
journalctl -u dhis2gw -f
```

---

### 🔄 Upgrade

```bash
sudo dpkg -i dhis2gw_X.Y.Z_amd64.deb
```

---

### 🧼 Uninstall

```bash
sudo apt remove dhis2gw
```

---

### Current Stable versions
👉 [Debian Package](https://github.com/HISP-Uganda/dhis2gw/releases)

👉 [Linux Binary v1.0.1](https://github.com/HISP-Uganda/dhis2gw/releases)

👉 [MacOs Binary](https://github.com/HISP-Uganda/dhis2gw/releases)

---

## ⚙️ Configuration Reference

This section outlines all configuration options for the `dhis2gw` service. These can be provided via environment variables, a configuration file (YAML), or defaults.

---

## 📦 `database`

| Field | Environment Variable | Description | Default |
|-------|----------------------|-------------|---------|
| `uri` | `DHIS2GW_DB` | Database connection URI | `postgres://username:password@localhost/dhis2gw?sslmode=disable` |

---

## 🌐 `server`

| Field | Environment Variable | Description | Default |
|-------|----------------------|-------------|---------|
| `host` | `DHIS2GW_HOST` | Server host | `localhost` |
| `port` | `DHIS2GW_SERVER_PORT` | HTTP server port | `9090` |
| `redis_address` | `DHIS2GW_REDIS` | Redis server address | `127.0.0.1:6379` |
| `max_retries` | `DHIS2GW_MAX_RETRIES` | Number of retry attempts | `3` |
| `max_concurrent` | `DHIS2GW_MAX_CONCURRENT` | Maximum concurrent submissions | `5` |
| `request_process_interval` | `DHIS2GW_REQUEST_PROCESS_INTERVAL` | Seconds between processing requests | `4` |
| `templates_directory` | `DHIS2GW_TEMPLATES_DIR` | Path to templates directory | `./templates` |
| `static_directory` | `DHIS2GW_STATIC_DIR` | Path to static assets directory | `./static` |
| `logdir` | `DHIS2GW_LOGDIR` | Log file directory | `/var/log/dhis2gw` |
| `docs_directory` | `RTC_DOCS_DIR` | Path to markdown documentation directory | `./docs/md_docs` |
| `migrations_dir` | `DHIS2GW_MIGRATTIONS_DIR` | Database migrations path | `file:///usr/share/dhis2gw/db/migrations` |
| `timezone` | `DISPATCHER2_TIMEZONE` | Deployment time zone | `Africa/Kampala` |

---

## 🔌 `api`

| Field | Environment Variable | Description | Default |
|-------|----------------------|-------------|---------|
| `dhis2_country` | `dhis2_country` | Name of DHIS2 Country | *(none)* |
| `dhis2_base_url` | `dhis2_base_url` | Base API URL of DHIS2 | *(none)* |
| `dhis2_user` | `dhis2_user` | DHIS2 username | *(none)* |
| `dhis2_password` | `dhis2_password` | DHIS2 password | *(none)* |
| `dhis2_pat` | `dhis2_pat` | DHIS2 personal access token | *(none)* |
| `save_response` | `save_response` | Save response from DHIS2 in DB | `true` |
| `mapping_scheme` | `mapping_scheme` | Aggregate mapping scheme | `CODE` |
| `dhis2_data_set` | `dhis2_data_set` | DataSet ID for DHIS2 | *(none)* |
| `dhis2_attribute_option_combo` | `dhis_2_attribute_option_combo` | Attribute option combo | *(none)* |
| `dhis2_auth_method` | `dhis2_auth_method` | Authentication method (`basic`, `pat`, etc.) | *(none)* |
| `dhis2_tree_i_ds` | `dhis2_tree_i_ds` | Top-level OU IDs (comma-separated) | *(none)* |
| `dhis2_facility_level` | `dhis2_facility_level` | OU level number for facilities | `5` |
| `dhis2_district_oulevel_name` | `DHIS2GW_DHIS2_DISTRICT_OULEVEL_NAME` | OU level name for districts | `District/City` |
| `cc_dhis2_servers` | `cc_dhis2_servers` | Comma-separated list of CC DHIS2 servers for facilities | *(none)* |
| `cc_dhis2_hierarchy_servers` | `cc_dhis2_hierarchy_servers` | Comma-separated list of CC DHIS2 servers for OU hierarchy | *(none)* |
| `cc_dhis2_create_servers` | `cc_dhis2_create_servers` | CC DHIS2 servers to receive new orgunits | *(none)* |
| `cc_dhis2_update_servers` | `cc_dhis2_update_servers` | CC DHIS2 servers to receive orgunit updates | *(none)* |
| `cc_dhis2_ou_group_add_servers` | `cc_dhis2_ou_group_add_servers` | CC DHIS2 servers to assign orgunits to groups | *(none)* |
| `sync_cron_expression` | `sync_cron_expression` | Cron expression for sync job | `0 0-23/6 * * *` |
| `retry_cron_expression` | `retry_cron_expression` | Cron expression for retry job | `*/5 * * * *` |



---

## 🛟 Support

For issues, feature requests, or contributions, visit the [GitHub repository](https://github.com/HISP-Uganda/dhis2gw)
