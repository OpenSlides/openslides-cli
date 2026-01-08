> **📋 Pre-Release Notice**
> 
> `osmanage` is currently in **alpha/pre-release stage** and is being actively developed for production use at Intevation GmbH. While the tool is functional, it is not yet ready for general community adoption:
> 
> - APIs and command interfaces may change
> - Some features are still being finalized
> - Documentation is work-in-progress
> - Production hardening is ongoing
> 
> We plan to release a stable, community-ready version in the future. For now, use at your own risk or for experimentation only. Contributions and feedback are welcome!

# osmanage

A command-line interface for managing OpenSlides instances. This tool provides direct access to OpenSlides backend actions, datastore queries, database migrations, and deployment setup.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [setup](#setup)
  - [action](#action)
  - [create-user](#create-user)
  - [get](#get)
  - [initial-data](#initial-data)
  - [migrations](#migrations)
  - [set](#set)
  - [set-password](#set-password)
- [Configuration](#configuration)
- [Examples](#examples)
- [Development](#development)
- [Credits](#credits)
- [License](#license)

---

## Overview

`osmanage` is a comprehensive management utility for OpenSlides 4.x instances. It combines deployment automation, database operations, and administrative tasks into a single tool.

**Key Features:**
- **Deployment Setup**: Generate Docker Compose and Kubernetes configurations
- **Secrets Management**: Automatic generation of secure passwords and certificates
- **Datastore Queries**: Direct PostgreSQL access with advanced filtering
- **Database Migrations**: Manage OpenSlides schema migrations with progress tracking
- **User Management**: Create users and manage passwords
- **Backend Actions**: Execute arbitrary OpenSlides actions

---

## Installation

### Binary Release

Download the latest binary from the [releases page](https://github.com/OpenSlides/openslides-cli/releases):

```bash
# Linux AMD64
curl -L https://github.com/OpenSlides/openslides-cli/releases/latest/download/osmanage -o osmanage
chmod +x osmanage
sudo mv osmanage /usr/local/bin/
```

### From Source

**Requirements:**
- Go 1.23 or later

```bash
git clone https://github.com/OpenSlides/openslides-cli.git
cd openslides-cli
CGO_ENABLED=0 go build -a -ldflags="-s -w" ./cmd/osmanage
```

---


## Quick Start

### Standard Setup (Auto-Initialize)

If your config has `OPENSLIDES_BACKEND_CREATE_INITIAL_DATA: 1` set (default):

```bash
# 1. Generate deployment configuration
osmanage setup ./openslides-deployment \
  --config config.yml \
  --template templates/docker-compose.yml

# 2. Start services
cd openslides-deployment
docker compose up -d

# 3. Wait for services to be ready (~30 seconds)
docker compose logs -f backendManage

# 4. Access OpenSlides at http://localhost:8000
# Login as 'superadmin' with password from:
cat secrets/superadmin

# 5. Stop services when done
docker compose down

# Optional: Remove volumes (complete cleanup)
docker compose down -v
```

---

## Commands

### `setup`

Creates deployment configuration files, secrets, and SSL certificates.

**Usage:**
```bash
osmanage setup <directory> [flags]
```

**Flags:**
- `-t, --template <path>`: Custom template file or directory
- `-c, --config <files>`: YAML config file(s) (can be used multiple times)
- `-f, --force`: Overwrite existing files

**Generated Files:**
- `docker-compose.yml` or Kubernetes manifests
- `secrets/` directory with:
  - `auth_token_key`
  - `auth_cookie_key`
  - `internal_auth_password`
  - `postgres_password`
  - `superadmin`
  - `cert_crt` and `cert_key` (if HTTPS enabled)

**Example:**
```bash
osmanage setup ./deployment \
  --config config.yml \
  --template templates/docker-compose.yml
```

### `action`

Execute arbitrary OpenSlides backend actions.

**Usage:**
```bash
osmanage action <action-name> [payload] [flags]
```

**Flags:**
- `-a, --address`: Backend service address (default: `localhost:9002`)
- `--password-file`: Authorization password file (default: `secrets/internal_auth_password`)
- `-f, --file`: JSON payload file or `-` for stdin

**Examples:**
```bash
# Inline JSON
osmanage action meeting.create '[{"name": "Annual Meeting", "committee_id": 1, "language": "de", "admin_ids": [1]}]'

# From file
osmanage action meeting.create --file meeting.json

# From stdin
echo '[{"name": "Test Meeting", "committee_id": 1, "language": "de", "admin_ids": [1]}]' | osmanage action meeting.create --file -
```

---

### `create-user`

Create a new OpenSlides user.

**Usage:**
```bash
osmanage create-user [user-data] [flags]
```

**Flags:**
- `-a, --address`: Backend service address (default: `localhost:9002`)
- `--password-file`: Authorization password file (default: `secrets/internal_auth_password`)
- `-f, --file`: JSON user data file or `-` for stdin

**Required Fields:**
- `username`: User login name
- `default_password`: Initial password

**Example:**
```bash
# From file
osmanage create-user --file user.json

# Inline
osmanage create-user '{"username": "admin", "default_password": "secret123", "first_name": "Admin", "last_name": "User"}'
```

**user.json:**
```json
{
  "username": "mmax",
  "default_password": "changemepwd",
  "first_name": "Max",
  "last_name": "Mustermann",
  "email": "mmax@example.com",
  "is_active": true
}
```

---

### `get`

Query the OpenSlides datastore with advanced filtering.

**Usage:**
```bash
osmanage get <collection> [flags]
```

**Supported Collections:**
- `user`
- `meeting`
- `organization`

**Flags:**
- `--postgres-host`: PostgreSQL host (default: `localhost`)
- `--postgres-port`: PostgreSQL port (default: `5432`)
- `--postgres-user`: PostgreSQL user (default: `instance_user`)
- `--postgres-database`: PostgreSQL database (default: `instance_db`)
- `--postgres-password-file`: Password file (default: `/secrets/postgres-password`)
- `--fields`: Comma-separated field list
- `--filter`: Simple key=value filters (multiple allowed, AND'ed together)
- `--filter-raw`: Complex JSON filter with operators
- `--exists`: Return boolean (requires filter)

**Supported Operators (in `--filter-raw`):**
- `=`: Equal
- `!=`: Not equal
- `>`: Greater than
- `<`: Less than
- `>=`: Greater than or equal
- `<=`: Less than or equal
- `~=`: Regex match

**Examples:**

**Simple queries:**
```bash
# Get all users with specific fields
osmanage get user --fields first_name,last_name,email

# Filter by equality
osmanage get user --filter is_active=true --filter username=admin

# Check existence
osmanage get meeting --filter id=1 --exists
```

**Complex filters:**
```bash
# Numeric comparison
osmanage get user --filter-raw '{"field":"id","operator":">","value":10}'

# Regex matching
osmanage get user --filter-raw '{"field":"username","operator":"~=","value":"^admin"}'

# AND filter
osmanage get user --filter-raw '{
  "and_filter": [
    {"field": "is_active", "operator": "=", "value": true},
    {"field": "first_name", "operator": "~=", "value": "^M"}
  ]
}'

# OR filter
osmanage get meeting --filter-raw '{
  "or_filter": [
    {"field": "name", "operator": "~=", "value": "Annual"},
    {"field": "name", "operator": "~=", "value": "Board"}
  ]
}'

# NOT filter
osmanage get user --filter-raw '{
  "not_filter": {
    "field": "is_active",
    "operator": "=",
    "value": false
  }
}'

# Nested filters
osmanage get user --filter-raw '{
  "and_filter": [
    {"field": "is_active", "operator": "=", "value": true},
    {
      "or_filter": [
        {"field": "username", "operator": "~=", "value": "^admin"},
        {"field": "username", "operator": "~=", "value": "^super"}
      ]
    }
  ]
}'
```

**Output Format:**
```json
{
  "1": {
    "id": 1,
    "username": "admin",
    "first_name": "Admin",
    "last_name": "User",
    "is_active": true
  },
  "2": {
    "id": 2,
    "username": "mmax",
    "first_name": "Max",
    "last_name": "Mustermann",
    "is_active": true
  }
}
```

---

### `initial-data`

Initialize a new OpenSlides datastore.

**Usage:**
```bash
osmanage initial-data [flags]
```

**Flags:**
- `-a, --address`: Backend service address (default: `localhost:9002`)
- `--password-file`: Authorization password file (default: `secrets/internal_auth_password`)
- `--superadmin-password-file`: Superadmin password file (default: `secrets/superadmin`)
- `-f, --file`: JSON initial data file or `-` for stdin

**Behavior:**
- Sets up organization and default data
- Sets superadmin (user ID 1) password
- **Returns error if datastore is not empty** (exit code 2)

**Example:**
```bash
# With default data
osmanage initial-data \
  --address localhost:9002 \
  --password-file secrets/internal_auth_password \
  --superadmin-password-file secrets/superadmin

# With custom initial data
osmanage initial-data \
  --address localhost:9002 \
  --password-file secrets/internal_auth_password \
  --superadmin-password-file secrets/superadmin \
  --file initial-data.json
```

---

### `migrations`

Manage OpenSlides database migrations.

**Subcommands:**
- `migrate`: Prepare migrations (dry-run)
- `finalize`: Prepare and apply migrations
- `reset`: Reset unapplied migrations
- `clear-collectionfield-tables`: Clear auxiliary tables (offline only)
- `stats`: Show migration statistics
- `progress`: Query progress of running migration

**Common Flags:**
- `-a, --address`: Backend service address (default: `localhost:9002`)
- `--password-file`: Authorization password file (default: `secrets/internal_auth_password`)
- `--interval`: Progress polling interval (default: `1s`, use `0` to disable)

**Features:**
- Automatic retry with exponential backoff
- Real-time progress tracking
- Context-aware timeouts
- Network error handling

**Examples:**
```bash
# Prepare migrations (dry-run)
osmanage migrations migrate --address localhost:9002

# Apply migrations with progress
osmanage migrations finalize \
  --address localhost:9002 \
  --password-file secrets/internal_auth_password

# Apply migrations without progress output
osmanage migrations finalize \
  --address localhost:9002 \
  --interval 0

# Check migration status
osmanage migrations stats --address localhost:9002

# Monitor running migration
osmanage migrations progress --address localhost:9002

# Reset migrations
osmanage migrations reset --address localhost:9002
```

**Migration Stats Output:**
```
current_migration_index: 15
target_migration_index: 20
positions: 1500
events: 5000
partially_migrated_positions: 500
fully_migrated_positions: 1000
status: migration_running
```

---

### `set`

Update OpenSlides objects using backend actions.

**Usage:**
```bash
osmanage set <action> [payload] [flags]
```

**Supported Actions:**
- `agenda_item`
- `committee`
- `group`
- `meeting`
- `motion`
- `organization`
- `organization_tag`
- `projector`
- `theme`
- `topic`
- `user`

**Flags:**
- `-a, --address`: Backend service address (default: `localhost:9002`)
- `--password-file`: Authorization password file (default: `secrets/internal_auth_password`)
- `-f, --file`: JSON payload file or `-` for stdin

**Examples:**
```bash
# Update user
osmanage set user '[{"id": 5, "first_name": "Jane", "last_name": "Smith"}]'

# Update meeting from file
osmanage set meeting --file meeting-update.json

# Update organization
osmanage set organization '[{"id": 1, "name": "Updated Organization Name"}]'
```

---

### `set-password`

Change a user's password.

**Usage:**
```bash
osmanage set-password [flags]
```

**Flags:**
- `-a, --address`: Backend service address (default: `localhost:9002`)
- `--password-file`: Authorization password file (default: `secrets/internal_auth_password`)
- `-u, --user_id`: User ID (required)
- `-p, --password`: New password (required)

**Example:**
```bash
osmanage set-password \
  --address localhost:9002 \
  --user_id 5 \
  --password "newSecurePassword123"
```

---

## Configuration

### Logging Levels

Set via the `--log-level` flag (applies to all commands):

```bash
osmanage --log-level debug get user
```

**Available levels:**
- `debug`: Detailed diagnostic information
- `info`: General informational messages (default)
- `warn`: Warning messages
- `error`: Error messages only

**Example output:**
```
[INFO] === GET COLLECTION ===
[DEBUG] Collection: user
[DEBUG] Found 150 total users
[DEBUG] Fields to fetch: [id first_name last_name email is_active]
[INFO] Query completed successfully
```

---

## Examples

### Complete Setup Workflow

```bash
# 1. Generate deployment configuration
osmanage setup ./openslides-deployment \
  --config config.yml \
  --template templates/docker-compose.yml

# 2. Start services (Docker Compose example)
cd openslides-deployment
docker-compose up -d

# 3. Access OpenSlides
# Visit http://localhost:8000
# Login as 'superadmin' with password from:
cat secrets/superadmin
```

### Backup User Data

```bash
# Export all users to JSON
osmanage get user > backup-users-$(date +%Y%m%d).json

# Export specific fields only
osmanage get user \
  --fields username,first_name,last_name,email \
  > backup-users-minimal.json
```

### Query Active Meetings

 ```bash
 # Get all active meetings with details
 osmanage get meeting \
   --filter is_active_in_organization_id=1 \
   --fields name,start_time,end_time,location
 
 # Count active meetings
 osmanage get meeting \
   --filter is_active_in_organization_id=1 \
   | jq 'length'

 # See if an active meeting exists by id
 osmanage get meeting \
  --filter-raw '{"and_filter": [{"field": "id", "operator": "=", "value": 1}, {"field": "is_active_in_organization_id", "operator": "=", "value": 1}]}'
   --exists
 # easier (id=5)
 osmanage get meeting --filter is_active_in_organization=1 --fields name | jq '. "5"'
 ```

## Development

### Project Structure

```
openslides-cli/
├── cmd/
│   └── osmanage/              # Main entry point
│       └── main.go
├── internal/
│   ├── actions/                # Action commands
│   │   ├── action/
│   │   │   └── action.go
│   │   ├── createuser/
│   │   │   └── createuser.go
│   │   ├── get/
│   │   │   ├── get.go
│   │   │   └── get_test.go
│   │   ├── initialdata/
│   │   │   └── initialdata.go
│   │   ├── set/
│   │   │   └── set.go
│   │   └── setpassword/
│   │       └── setpassword.go
│   ├── client/                 # HTTP client
│   │   ├── client.go
│   │   └── client_test.go
│   ├── logger/                 # Logging
│   │   ├── logger.go
│   │   └── logger_test.go
│   ├── migrations/             # Migration commands
│   │   ├── migrations.go
│   │   └── migrations_test.go
│   ├── templating/             # Setup & templating
│   │   ├── config/
│   │   └── setup/
│   └── utils/                  # Utilities
│       ├── utils.go
│       └── utils_test.go
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

### Building

```bash
# Build for testing (bigger binary, debuggable)
go build -o osmanage ./cmd/osmanage
# Build for prod (smaller binary, no C code, no debug)
CGO_ENABLED=0 go build -a -ldflags="-s -w" -o osmanage ./cmd/osmanage
```

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`go test ./...`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

**Code Style:**
- Follow standard Go conventions
- Run `go fmt` before committing
- Add tests for new functionality
- Update documentation as needed

---

## Credits

This tool represents a significant refactor and expansion of the original [openslides-manage-service](https://github.com/OpenSlides/openslides-manage-service) created by **Norman Jäckel**.

**Major Changes from Original:**
- Migration from `datastorereader` to `openslides-go/datastore/dsfetch`
- Removed gRPC
- Filtering in-memory (until better solution is found -> TODO)
- retry mechanism for migrations
- Comprehensive test coverage
- Improved deployment configuration system
- Simplified the templating

**Refactored/Developed by:** Alexej Antoni @ Intevation GmbH

**Original work by:** Norman Jäckel

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Support

- **Issues**: [GitHub Issues](https://github.com/OpenSlides/openslides-cli/issues)
- **Discussions**: [GitHub Discussions](https://github.com/OpenSlides/openslides-cli/discussions)