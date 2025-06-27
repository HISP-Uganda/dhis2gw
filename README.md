# dhis2gw: DHIS2 Gateway Service

## Overview

**dhis2gw** is a robust gateway application for sending aggregate and tracker data to DHIS2 from third-party systems. Built with Go and Gin, it leverages asynchronous processing (Asynq) and PostgreSQL for task management and logging. It also provides a KivyMD desktop UI for administrators to monitor, requeue, and manage tasks.

---

## Table of Contents

- [Tech Stack](#tech-stack)
- [Features](#features)
- [API Endpoints](#api-endpoints)
    - [User Management](#user-management)
    - [Authentication](#authentication)
    - [Aggregate Data Submission](#aggregate-data-submission)
    - [Logs Management](#logs-management)
    - [DHIS2 Mappings](#dhis2-mappings)
    - [Swagger Documentation](#swagger-documentation)
- [Logging & Monitoring](#logging--monitoring)
- [KivyMD Interface](#kivymd-interface)
- [Architecture Diagram](#architecture-diagram)
- [Future Improvements](#future-improvements)

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
          |                   |        |    |    |
          |    KivyMD         |        |    |    |      +--------------------+
          |  Admin App        +<-------+    |    +----->|  DHIS2 API         |
          |  (Python)         |     REST    |          +--------------------+
          +-------------------+        |    |
                                       |    |
                            +----------v----v----------+
                            |   PostgreSQL (DB)        |
                            |  (users, logs, mappings) |
                            +--------------------------+
