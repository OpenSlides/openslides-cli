
# osmanage

A command-line interface for managing OpenSlides instances. This tool provides deployment automation, Kubernetes orchestration, direct access to OpenSlides backend actions, database queries, and database migrations.


## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
  - [Docker Compose Setup](#docker-compose-setup)
- [Commands](#commands)
  - [Instance Management](#instance-management)
    - [setup](#setup)
    - [config](#config)
  - [Backend Actions](#backend-actions)
    - [migrations](#migrations)
    - [initial-data](#initial-data)
    - [create-user](#create-user)
    - [set-password](#set-password)
    - [get](#get)
    - [set](#set)
    - [action](#action)
  - [Kubernetes Operations](#kubernetes-operations)
- [Examples](#examples)
- [Development](#development)
- [Credits](#credits)
- [License](#license)


## Overview

`osmanage` is a comprehensive management utility for OpenSlides 4.x instances. It combines deployment automation, Kubernetes orchestration, database operations, and administrative tasks into a single tool.

**Key Features:**
- **Deployment Setup**: Generate Docker Compose and Kubernetes configurations
- **Kubernetes Management**: Deploy, update, and manage OpenSlides in Kubernetes
- **Secrets Management**: Automatic generation of secure passwords and certificates
- **Database Queries**: Direct PostgreSQL access with advanced filtering
- **Database Migrations**: Manage OpenSlides schema migrations with progress tracking
- **User Management**: Create users and manage passwords
- **Backend Actions**: Execute arbitrary OpenSlides actions


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


## Quick Start


### Docker Compose Setup

Standard workflow for Docker Compose deployments:

```bash
# 1. Generate deployment configuration with random secrets
osmanage setup ./my.instance.dir.org \
  --config config.yml \
  --template docker-compose.yml.tmpl

# 2. Start services
cd my.instance.dir.org
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


## Commands

This is giving an overview of a selection of available commands. See `--help` messages for more complete information.


### Instance Management

Commands for creating and managing OpenSlides instance directories.


#### `setup`

Creates a new instance directory with deployment configuration, secrets, and SSL certificates.

**Usage:**

```bash
osmanage setup <instance-dir> [flags]
```

**Generated Structure:**

```
my.instance.dir.org/
├── docker-compose.yml          # (if using Docker Compose template)
└── secrets/
    ├── auth_token_key
    ├── auth_cookie_key
    ├── internal_auth_password
    ├── postgres_password
    ├── superadmin
    ├── cert_crt               # (if HTTPS enabled)
    └── cert_key
```

**Examples:**

```bash
# Docker Compose deployment
osmanage setup ./my.instance.dir.org \
  --config config.yml \
  --template docker-compose.yml.tmpl

# Multiple config files (merged, later files override earlier)
osmanage setup ./my.instance.dir.org \
  --config base-config.yml \
  --config prod-overrides.yml \
  --template docker-compose.yml.tmpl

# Overwrite existing instance
osmanage setup ./my.instance.dir.org \
  --config config.yml \
  --template docker-compose.yml.tmpl \
  --force
```


#### `config`

(Re)creates deployment configuration files from templates and YAML config files.

**Usage:**

```bash
osmanage config <instance-dir> [flags]
```

**Behavior:**
- Merges multiple YAML config files (later file's fields override earlier ones)
- Renders templates with merged configuration
- Creates or overwrites deployment files in the instance directory

**Use Cases:**
- Regenerate deployment files after config changes
- Update templates without recreating secrets
- Apply new configuration to existing instance
- Fix or modify deployment manifests

**Examples:**

```bash
# Regenerate deployment files
osmanage config ./my.instance.dir.org \
  --template ./k8s-templates \
  --config ./config.yml

# Update with multiple configs (merged)
osmanage config ./my.instance.dir.org \
  --template ./k8s-templates \
  --config base-config.yml \
  --config prod-overrides.yml

# Force overwrite existing files
osmanage config ./my.instance.dir.org \
  --template docker-compose.yml \
  --config config.yml \
  --force
```

**Note:** This command does NOT regenerate secrets - it only (re)creates deployment files. Use `osmanage setup` for initial instance creation with secrets, or `osmanage create` to update passwords.


### Backend Actions

Commands for interacting with the OpenSlides backend API.

**Note:** All backend action commands require `--address` and `--password-file` flags.


#### `migrations`

Manage OpenSlides database migrations.

**Subcommands:**
- `migrate`: Run migrations on auxiliary tables
- `finalize`: Apply migrations to live tables
- `reset`: Reset unapplied migrations
- `stats`: Show migration statistics
- `progress`: Check running migration progress

**Examples:**

```bash
# Check migration status
osmanage migrations stats \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password

# Run migrations
osmanage migrations migrate \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password

# Apply migrations
osmanage migrations finalize \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password

# Apply without progress output
osmanage migrations finalize \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password \
  --interval 0
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


#### `initial-data`

Initialize a new OpenSlides database.

**Usage:**

```bash
osmanage initial-data [flags]
```

**Behavior:**
- Sets up organization and default data
- Sets superadmin (user ID 1) password
- Returns error if database is not empty (exit code 2)

**Examples:**

```bash
# Docker Compose
osmanage initial-data \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password \
  --superadmin-password-file ./secrets/superadmin
```


#### `create-user`

Create a new OpenSlides user.

**Usage:**

```bash
osmanage create-user [user-data] [flags]
```

**Required JSON Fields:**
- `username`: User login name
- `default_password`: Initial password

**Examples:**

```bash
# Inline JSON
osmanage create-user '{"username": "admin", "default_password": "secret123"}' \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password

# From file
osmanage create-user \
  --file user.json \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password
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


#### `set-password`

Change a user's password.

**Usage:**

```bash
osmanage set-password [flags]
```

**Example:**

```bash
osmanage set-password \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password \
  --user_id 5 \
  --password "newSecurePassword123"
```


#### `get`

Query the OpenSlides database with advanced filtering.

> [!IMPORTANT]
> Requires access to postgres. Examples assume port is forwarded.

**Usage:**

```bash
osmanage get <collection> [flags]
```

**Supported Collections:**
- `user`
- `meeting`
- `organization`

**Supported Operators (in `--filter-raw`):**
- `=`: Equal
- `!=`: Not equal
- `>`: Greater than
- `<`: Less than
- `>=`: Greater than or equal
- `<=`: Less than or equal
- `~=`: Regex match

**Examples:**

```bash
# Simple query
osmanage get user --fields first_name,last_name,email \
  --postgres-host localhost \
  --postgres-port 5432 \
  --postgres-user openslides \
  --postgres-database openslides \
  --postgres-password-file ./secrets/postgres_password

# With filter
osmanage get user --filter is_active=true \
  --postgres-host localhost \
  --postgres-port 5432 \
  --postgres-user openslides \
  --postgres-database openslides \
  --postgres-password-file ./secrets/postgres_password
```

**Complex filters:**

```bash
# Regex matching
osmanage get user \
  --filter-raw '{"field":"username","operator":"~=","value":"^admin"}' \
  --postgres-host localhost \
  --postgres-port 5432 \
  --postgres-user openslides \
  --postgres-database openslides \
  --postgres-password-file ./secrets/postgres_password

# AND filter
osmanage get user \
  --filter-raw '{"and_filter":[{"field":"is_active","operator":"=","value":true},{"field":"first_name","operator":"~=","value":"^M"}]}' \
  --postgres-host localhost \
  --postgres-port 5432 \
  --postgres-user openslides \
  --postgres-database openslides \
  --postgres-password-file ./secrets/postgres_password

# Check existence
osmanage get meeting \
  --filter id=1 \
  --exists \
  --postgres-host localhost \
  --postgres-port 5432 \
  --postgres-user openslides \
  --postgres-database openslides \
  --postgres-password-file ./secrets/postgres_password
```


#### `set`

Update OpenSlides objects using backend actions.

**Usage:**

```bash
osmanage set <action> [payload] [flags]
```

**Supported Actions:**
- `agenda_item`, `committee`, `group`, `meeting`, `motion`, `organization`, `organization_tag`, `projector`, `theme`, `topic`, `user`

**Examples:**

```bash
# Update user
osmanage set user '[{"id": 5, "first_name": "Jane", "last_name": "Smith"}]' \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password

# Update from file
osmanage set meeting \
  --file meeting-update.json \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password
```


#### `action`

Execute arbitrary OpenSlides backend actions.

**Usage:**

```bash
osmanage action <action-name> [payload] [flags]
```

**Examples:**

```bash
# Docker Compose (localhost)
osmanage action meeting.create '[{"name": "Annual Meeting", "committee_id": 1, "language": "de", "admin_ids": [1]}]' \
  --address localhost:9002 \
  --password-file ./my.instance.dir.org/secrets/internal_auth_password

# Kubernetes (port-forwarded)
kubectl port-forward -n myinstancedirorg svc/backendmanage 9002:9002 &
osmanage action meeting.create '[{"name": "Board Meeting", "committee_id": 1, "language": "de", "admin_ids": [1]}]' \
  --address localhost:9002 \
  --password-file ./my.instance.dir.org/secrets/internal_auth_password

# From file
osmanage action meeting.create \
  --file meeting.json \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password

# From stdin
echo '[{"name": "Test", "committee_id": 1, "language": "de", "admin_ids": [1]}]' | \
  osmanage action meeting.create --file - \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password
```


### Kubernetes Operations

Commands for managing OpenSlides instances in Kubernetes.

> [!CAUTION]
> These commands are still experimental. Use with caution!

**Requirements:**
- Valid kubeconfig file with cluster access (typically `~/.kube/config`)
  - Or running inside a Kubernetes cluster with service account permissions
- Sufficient Kubernetes RBAC permissions to create/manage namespaces and resources

**Note:** `osmanage` uses the Kubernetes Go client library and does **not** require `kubectl` to be installed.


#### `k8s start`

Deploys an OpenSlides instance to Kubernetes.

**Usage:**

```bash
osmanage k8s start <instance-dir> [flags]
```

**Features:**
- Creates dedicated namespace from namespace.yaml
- Creates secrets from instance secrets/ directory (base64-encoded)
- Applies all Kubernetes manifests from stack/ directory
- Shows progress bars for deployment readiness
- Waits for all pods to be healthy


#### `k8s stop`

Stops and removes an OpenSlides instance from Kubernetes.

**Usage:**

```bash
osmanage k8s stop <instance-dir> [flags]
```

**Behavior:**
- Saves TLS certificate secret (if exists) to `secrets/tls-letsencrypt-secret.yaml`
- Deletes the namespace and all resources

**Warning:** This deletes the namespace and all resources, including persistent volumes.


#### `k8s update-instance`

Updates an existing Kubernetes instance with new manifests.

**Usage:**

```bash
osmanage k8s update-instance <instance-dir> [flags]
```

**Use Cases:**
- Apply configuration changes
- Update resource limits
- Modify service definitions
- Change replica counts


#### `k8s update-backendmanage`

Updates the backendmanage container image.

**Usage:**

```bash
osmanage k8s update-backendmanage <instance-dir> [flags]
```


#### `k8s scale`

Scales a specific service deployment.

**Usage:**

```bash
osmanage k8s scale <instance-dir> [flags]
```

**Note:** You must edit the deployment manifest file (`stack/<service>-deployment.yaml`) to change replica count before running this command.


#### `k8s health`

Checks the health status of an OpenSlides instance.

**Usage:**

```bash
osmanage k8s health <instance-dir> [flags]
```

**Features:**
- Reports pod status for all deployments
- Shows ready/total pod counts
- Indicates overall instance health


#### `k8s cluster-status`

Displays comprehensive cluster status.

**Usage:**

```bash
osmanage k8s cluster-status [flags]
```

**Features:**
- Shows cluster-wide node health
- Reports ready vs total nodes


## Examples

### Complete Kubernetes Workflow

```bash
# 1. Generate instance
osmanage setup ./prod.instance.org \
  --config prod-config.yml \
  --template k8s-template-dir

# 2. Customize secrets (optional)
osmanage create ./prod.instance.org \
  --db-password "$SECURE_DB_PASS" \
  --superadmin-password "$SECURE_ADMIN_PASS"

# 3. Deploy to Kubernetes
osmanage k8s start ./prod.instance.org

# 4. Check health
osmanage k8s health ./prod.instance.org

# 5. Scale backend deployment (after editing manifest)
osmanage k8s scale ./prod.instance.org --service projector

# 6. Update backend image
osmanage k8s update-backendmanage ./prod.instance.org \
  --tag 4.2.1 \
  --container-registry myregistry

# 7. Stop instance
osmanage k8s stop ./prod.instance.org
```


### Query User Data

```bash
# Export all users
osmanage get user \
  --postgres-host localhost \
  --postgres-port 5432 \
  --postgres-user instance_user \
  --postgres-database instance_db \
  --postgres-password-file ./my.instance.dir.org/secrets/postgres_password \
  > backup-users-$(date +%Y%m%d).json

# Export specific fields
osmanage get user \
  --fields username,first_name,last_name,email \
  --postgres-host localhost \
  --postgres-port 5432 \
  --postgres-user instance_user \
  --postgres-database instance_db \
  --postgres-password-file ./my.instance.dir.org/secrets/postgres_password \
  > backup-users-minimal.json
```


### Query Active Meetings

```bash
# Get all active meetings with details
osmanage get meeting \
  --filter is_active_in_organization_id=1 \
  --fields name,start_time,end_time,location \
  --postgres-host localhost \
  --postgres-port 5432 \
  --postgres-user openslides \
  --postgres-database openslides \
  --postgres-password-file ./secrets/postgres_password

# Count active meetings
osmanage get meeting \
  --filter is_active_in_organization_id=1 \
  --postgres-host localhost \
  --postgres-port 5432 \
  --postgres-user openslides \
  --postgres-database openslides \
  --postgres-password-file ./secrets/postgres_password \
  | jq 'length'

# Check if specific meeting exists and is active
osmanage get meeting \
  --filter-raw '{"and_filter":[{"field":"id","operator":"=","value":1},{"field":"is_active_in_organization_id","operator":"=","value":1}]}' \
  --exists \
  --postgres-host localhost \
  --postgres-port 5432 \
  --postgres-user openslides \
  --postgres-database openslides \
  --postgres-password-file ./secrets/postgres_password
```


## Development


### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/k8s/actions

# Verbose output
go test -v ./...
```


### Building

```bash
# Development build (larger binary, debuggable)
go build -o osmanage ./cmd/osmanage

# Production build (smaller binary, optimized, no debug symbols)
CGO_ENABLED=0 go build -a -ldflags="-s -w" -o osmanage ./cmd/osmanage
```


### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`go test ./...`)
5. Run `go fmt ./...` to format code
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

**Code Style:**
- Follow standard Go conventions
- Run `go fmt` before committing
- Add tests for new functionality
- Update documentation as needed
- Use project constants from `internal/constants`


## Credits

This tool represents a significant refactor and expansion of the original [openslides-manage-service](https://github.com/OpenSlides/openslides-manage-service) created by **Norman Jäckel**.

**Major Changes from Original:**
- Complete Kubernetes orchestration system with health checks and progress tracking
- Migration from `datastorereader` to `openslides-go/datastore/dsfetch`
- Removed gRPC dependencies
- In-memory filtering for datastore queries
- Comprehensive retry mechanisms for migrations
- Extensive test coverage
- Improved deployment configuration and templating
- Centralized constants and project structure
- Instance management commands (setup/config/create/remove)
- Real-time deployment monitoring with progress bars
- Cluster status and health monitoring

**Refactored/Developed by:** Alexej Antoni @ Intevation GmbH

**Original work by:** Norman Jäckel


## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.


## Support

- **Issues**: [GitHub Issues](https://github.com/OpenSlides/openslides-cli/issues)
- **Discussions**: [GitHub Discussions](https://github.com/OpenSlides/openslides-cli/discussions)
