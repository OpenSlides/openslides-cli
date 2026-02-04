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

A command-line interface for managing OpenSlides instances. This tool provides deployment automation, Kubernetes orchestration, direct access to OpenSlides backend actions, datastore queries, and database migrations.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
  - [Docker Compose Setup](#docker-compose-setup)
- [Commands](#commands)
  - [Instance Management](#instance-management)
    - [setup](#setup)
    - [config](#config)
    - [create](#create)
    - [remove](#remove)
  - [Kubernetes Operations](#kubernetes-operations)
    - [k8s start](#k8s-start)
    - [k8s stop](#k8s-stop)
    - [k8s update-instance](#k8s-update-instance)
    - [k8s update-backendmanage](#k8s-update-backendmanage)
    - [k8s scale](#k8s-scale)
    - [k8s health](#k8s-health)
    - [k8s cluster-status](#k8s-cluster-status)
  - [Backend Actions](#backend-actions)
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

`osmanage` is a comprehensive management utility for OpenSlides 4.x instances. It combines deployment automation, Kubernetes orchestration, database operations, and administrative tasks into a single tool.

**Key Features:**
- **Deployment Setup**: Generate Docker Compose and Kubernetes configurations
- **Kubernetes Management**: Deploy, update, and manage OpenSlides in Kubernetes
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

### Docker Compose Setup

Standard workflow for Docker Compose deployments:
```bash
# 1. Generate deployment configuration with random secrets
osmanage setup ./my.instance.dir.org \
  --config config.yml \
  --template docker-compose.yml

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

---

## Commands

### Instance Management

Commands for creating and managing OpenSlides instance directories.

---

#### `setup`

Creates a new instance directory with deployment configuration, secrets, and SSL certificates.

**Usage:**
```bash
osmanage setup <instance-dir> [flags]
```

**Flags:**
- `-t, --template <path>`: Template file or directory (required)
- `-c, --config <files>`: YAML config file(s) (can be used multiple times, required)
- `-f, --force`: Overwrite existing files

**Generated Structure:**
```
my.instance.dir.org/
├── docker-compose.yml          # (if using Docker Compose template)
├── namespace.yaml              # (if using Kubernetes template)
├── stack/                      # (if using Kubernetes template)
│   ├── autoupdate-deployment.yaml
│   ├── backend-deployment.yaml
│   ├── postgres-deployment.yaml
│   └── ...
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
  --template docker-compose.yml

# Kubernetes deployment
osmanage setup ./my.instance.dir.org \
  --config k8s-config.yml \
  --template k8s-template-dir

# Multiple config files (merged, later files override earlier)
osmanage setup ./my.instance.dir.org \
  --config base-config.yml \
  --config prod-overrides.yml \
  --template k8s-template-dir

# Overwrite existing instance
osmanage setup ./my.instance.dir.org \
  --config config.yml \
  --template k8s-template-dir \
  --force
```

---

#### `config`

(Re)creates deployment configuration files from templates and YAML config files.

**Usage:**
```bash
osmanage config <instance-dir> [flags]
```

**Flags:**
- `-t, --template <path>`: Template file or directory (required)
- `-c, --config <files>`: YAML config file(s) (can be used multiple times, required)
- `-f, --force`: Overwrite existing files

**Behavior:**
- Merges multiple YAML config files (later files override earlier ones)
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

---

#### `create`

Updates an existing instance with new passwords.

**Usage:**
```bash
osmanage create <instance-dir> [flags]
```

**Flags:**
- `--db-password <password>`: Set PostgreSQL password (required)
- `--superadmin-password <password>`: Set superadmin password (required)

**Use Cases:**
- Set specific passwords instead of random ones
- Rotate passwords for security
- Fix incorrect secret file permissions

**Examples:**
```bash
# Set specific passwords
osmanage create ./my.instance.dir.org \
  --db-password "MySecureDBPassword123" \
  --superadmin-password "AdminPassword456"

# Password rotation
osmanage create ./my.instance.dir.org \
  --db-password "$NEW_DB_PASS" \
  --superadmin-password "$NEW_ADMIN_PASS"
```

---

#### `remove`

Deletes an instance directory and all its contents.

**Usage:**
```bash
osmanage remove <instance-dir> [flags]
```

**Flags:**
- `-f, --force`: Skip confirmation prompt

**Warning:** This permanently deletes all files in the instance directory, including secrets and manifests.

**Examples:**
```bash
# With confirmation prompt
osmanage remove ./my.instance.dir.org

# Skip confirmation
osmanage remove ./my.instance.dir.org --force
```

---

### Kubernetes Operations

Commands for managing OpenSlides instances in Kubernetes.

**Requirements:**
- Valid kubeconfig file with cluster access (typically `~/.kube/config`)
  - Or running inside a Kubernetes cluster with service account permissions
- Sufficient Kubernetes RBAC permissions to create/manage namespaces and resources

**Note:** `osmanage` uses the Kubernetes Go client library and does **not** require `kubectl` to be installed.

---

#### `k8s start`

Deploys an OpenSlides instance to Kubernetes.

**Usage:**
```bash
osmanage k8s start <instance-dir> [flags]
```

**Flags:**
- `--kubeconfig <path>`: Path to kubeconfig file (optional)
- `--skip-ready-check`: Skip waiting for instance to become ready
- `--timeout <duration>`: Maximum time to wait for deployment (default: 3m)

**Features:**
- Creates dedicated namespace from namespace.yaml
- Creates secrets from instance secrets/ directory (base64-encoded)
- Applies all Kubernetes manifests from stack/ directory
- Shows progress bars for deployment readiness
- Waits for all pods to be healthy

**Examples:**
```bash
# Standard deployment
osmanage k8s start ./my.instance.dir.org

# Custom timeout
osmanage k8s start ./my.instance.dir.org --timeout 5m

# Skip health check
osmanage k8s start ./my.instance.dir.org --skip-ready-check
```

**Output:**
```
Applying manifest: my.instance.dir.org/namespace.yaml
Applied namespace: myinstancedirorg
Applying stack manifests from: my.instance.dir.org/stack/
...

Waiting for instance to become ready:
[████████████████████████████████████████] Pods ready (13/13)

Instance is healthy: 13/13 pods ready
Instance started successfully
```

---

#### `k8s stop`

Stops and removes an OpenSlides instance from Kubernetes.

**Usage:**
```bash
osmanage k8s stop <instance-dir> [flags]
```

**Flags:**
- `--kubeconfig <path>`: Path to kubeconfig file (optional)
- `--timeout <duration>`: Maximum time to wait for deletion (default: 5m)

**Behavior:**
- Saves TLS certificate secret (if exists) to `secrets/tls-letsencrypt-secret.yaml`
- Deletes the namespace and all resources

**Warning:** This deletes the namespace and all resources, including persistent volumes.

**Examples:**
```bash
# Stop instance
osmanage k8s stop ./my.instance.dir.org

# Custom timeout
osmanage k8s stop ./my.instance.dir.org --timeout 10m
```

---

#### `k8s update-instance`

Updates an existing Kubernetes instance with new manifests.

**Usage:**
```bash
osmanage k8s update-instance <instance-dir> [flags]
```

**Flags:**
- `--kubeconfig <path>`: Path to kubeconfig file (optional)
- `--skip-ready-check`: Skip waiting for instance to become ready
- `--timeout <duration>`: Maximum time to wait for rollout (default: 3m)

**Use Cases:**
- Apply configuration changes
- Update resource limits
- Modify service definitions
- Change replica counts

**Examples:**
```bash
# Update after config changes
osmanage k8s update-instance ./my.instance.dir.org

# Update with custom timeout
osmanage k8s update-instance ./my.instance.dir.org --timeout 5m

# Skip health check
osmanage k8s update-instance ./my.instance.dir.org --skip-ready-check
```

---

#### `k8s update-backendmanage`

Updates the backendmanage container image.

**Usage:**
```bash
osmanage k8s update-backendmanage <instance-dir> [flags]
```

**Flags (Required):**
- `--tag <string>`: OpenSlides version tag (required)
- `--container-registry <string>`: Container registry (required)

**Flags (Optional):**
- `--kubeconfig <path>`: Path to kubeconfig file
- `--timeout <duration>`: Maximum time to wait for rollout (default: 3m)
- `--revert`: Revert to previous image (uses tag and registry as revert target)

**Examples:**
```bash
# Update to specific version
osmanage k8s update-backendmanage ./my.instance.dir.org \
  --tag 4.2.0 \
  --container-registry myregistry

# Update to latest
osmanage k8s update-backendmanage ./my.instance.dir.org \
  --tag latest \
  --container-registry myregistry

# Revert to previous version
osmanage k8s update-backendmanage ./my.instance.dir.org \
  --tag 4.1.9 \
  --container-registry myregistry
  --revert

# Custom timeout
osmanage k8s update-backendmanage ./my.instance.dir.org \
  --tag 4.2.1 \
  --container-registry myregistry
  --timeout 5m
```

---

#### `k8s scale`

Scales a specific service deployment.

**Usage:**
```bash
osmanage k8s scale <instance-dir> [flags]
```

**Flags (Required):**
- `--service <name>`: Service deployment to scale (required)

**Flags (Optional):**
- `--kubeconfig <path>`: Path to kubeconfig file
- `--skip-ready-check`: Skip waiting for deployment to become ready
- `--timeout <duration>`: Maximum time to wait for scaling (default: 3m)

**Note:** You must edit the deployment manifest file (`stack/<service>-deployment.yaml`) to change replica count before running this command.

**Examples:**
```bash
# Scale backend deployment (after editing manifest)
osmanage k8s scale ./my.instance.dir.org --service backend

# Scale autoupdate without health check
osmanage k8s scale ./my.instance.dir.org --service autoupdate --skip-ready-check

# Scale with custom timeout
osmanage k8s scale ./my.instance.dir.org --service backend --timeout 5m
```

---

#### `k8s health`

Checks the health status of an OpenSlides instance.

**Usage:**
```bash
osmanage k8s health <instance-dir> [flags]
```

**Flags:**
- `--kubeconfig <path>`: Path to kubeconfig file (optional)
- `--wait`: Wait for instance to become healthy
- `--timeout <duration>`: Timeout for health check (default: 3m, only with --wait)

**Features:**
- Reports pod status for all deployments
- Shows ready/total pod counts
- Indicates overall instance health

**Example:**
```bash
# Check current health
osmanage k8s health ./my.instance.dir.org

# Wait for instance to become healthy
osmanage k8s health ./my.instance.dir.org --wait --timeout 5m
```

**Output:**
```
Namespace: myinstancedirorg
Ready: 13/13 pods

Pod Status:
  ✓ auth-abc123                               Running
  ✓ autoupdate-def456                         Running
  ✓ backendaction-ghi789                      Running
  ✓ backendmanage-jkl012                      Running
  ✓ backendpresenter-mno345                   Running
  ✓ client-pqr678                             Running
  ✓ datastorereader-stu901                    Running
  ✓ datastorewriter-vwx234                    Running
  ✓ icc-yza567                                Running
  ✓ media-bcd890                              Running
  ✓ redis-efg123                              Running
  ✓ search-hij456                             Running
  ✓ vote-klm789                               Running
```

---

#### `k8s cluster-status`

Displays comprehensive cluster status.

**Usage:**
```bash
osmanage k8s cluster-status [flags]
```

**Flags:**
- `--kubeconfig <path>`: Path to kubeconfig file (optional)

**Features:**
- Shows cluster-wide node health
- Reports ready vs total nodes

**Example:**
```bash
osmanage k8s cluster-status
```

**Output:**
```
cluster_status: 3 3

Total nodes: 3
Ready nodes: 3
Node node1: Ready
Node node2: Ready
Node node3: Ready
Cluster is healthy
```

---

### Backend Actions

Commands for interacting with the OpenSlides backend API.

**Note:** All backend action commands require `--address` and `--password-file` flags.

---

#### `action`

Execute arbitrary OpenSlides backend actions.

**Usage:**
```bash
osmanage action <action-name> [payload] [flags]
```

**Flags (Required):**
- `-a, --address <host:port>`: Backend service address (required)
- `--password-file <path>`: Authorization password file (required)

**Flags (Optional):**
- `-f, --file <path>`: JSON payload file or `-` for stdin

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

---

#### `create-user`

Create a new OpenSlides user.

**Usage:**
```bash
osmanage create-user [user-data] [flags]
```

**Flags (Required):**
- `-a, --address <host:port>`: Backend service address (required)
- `--password-file <path>`: Authorization password file (required)

**Flags (Optional):**
- `-f, --file <path>`: JSON user data file or `-` for stdin

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

---

#### `get`

Query the OpenSlides datastore with advanced filtering.

**Usage:**
```bash
osmanage get <collection> [flags]
```

**Supported Collections:**
- `user`
- `meeting`
- `organization`

**Flags (Required):**
- `--postgres-host <host>`: PostgreSQL host (required)
- `--postgres-port <port>`: PostgreSQL port (required)
- `--postgres-user <user>`: PostgreSQL user (required)
- `--postgres-database <db>`: PostgreSQL database (required)
- `--postgres-password-file <path>`: PostgreSQL password file (required)

**Flags (Optional):**
- `--fields <list>`: Comma-separated field list
- `--filter <key=value>`: Simple equality filters (can be used multiple times, AND'ed together)
- `--filter-raw <json>`: Complex JSON filter with operators
- `--exists`: Return boolean instead of data (requires filter)

**Supported Operators (in `--filter-raw`):**
- `=`: Equal
- `!=`: Not equal
- `>`: Greater than
- `<`: Less than
- `>=`: Greater than or equal
- `<=`: Less than or equal
- `~=`: Regex match

**Examples:**

**Docker Compose:**
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

#### `initial-data`

Initialize a new OpenSlides datastore.

**Usage:**
```bash
osmanage initial-data [flags]
```

**Flags (Required):**
- `-a, --address <host:port>`: Backend service address (required)
- `--password-file <path>`: Authorization password file (required)
- `--superadmin-password-file <path>`: Superadmin password file (required)

**Flags (Optional):**
- `-f, --file <path>`: JSON initial data file or `-` for stdin

**Behavior:**
- Sets up organization and default data
- Sets superadmin (user ID 1) password
- Returns error if datastore is not empty (exit code 2)

**Examples:**
```bash
# Docker Compose
osmanage initial-data \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password \
  --superadmin-password-file ./secrets/superadmin
```

---

#### `migrations`

Manage OpenSlides database migrations.

**Subcommands:**
- `migrate`: Prepare migrations (dry-run)
- `finalize`: Apply migrations to datastore
- `reset`: Reset unapplied migrations
- `clear-collectionfield-tables`: Clear auxiliary tables (offline only)
- `stats`: Show migration statistics
- `progress`: Check running migration progress

**Flags (Required):**
- `-a, --address <host:port>`: Backend service address (required)
- `--password-file <path>`: Authorization password file (required)

**Flags (Optional):**
- `--interval <duration>`: Progress check interval (default: `1s`, use `0` to disable, only for migrate/finalize)

**Examples:**
```bash
# Check migration status
osmanage migrations stats \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password

# Prepare migrations (dry-run)
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

---

#### `set`

Update OpenSlides objects using backend actions.

**Usage:**
```bash
osmanage set <action> [payload] [flags]
```

**Supported Actions:**
- `agenda_item`, `committee`, `group`, `meeting`, `motion`, `organization`, `organization_tag`, `projector`, `theme`, `topic`, `user`

**Flags (Required):**
- `-a, --address <host:port>`: Backend service address (required)
- `--password-file <path>`: Authorization password file (required)

**Flags (Optional):**
- `-f, --file <path>`: JSON payload file or `-` for stdin

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

---

#### `set-password`

Change a user's password.

**Usage:**
```bash
osmanage set-password [flags]
```

**Flags (Required):**
- `-a, --address <host:port>`: Backend service address (required)
- `--password-file <path>`: Authorization password file (required)
- `-u, --user_id <id>`: User ID (required)
- `-p, --password <password>`: New password (required)

**Example:**
```bash
osmanage set-password \
  --address localhost:9002 \
  --password-file ./secrets/internal_auth_password \
  --user_id 5 \
  --password "newSecurePassword123"
```

---

## Configuration

### Logging Levels

Control verbosity with the global `--log-level` flag:
```bash
osmanage --log-level debug k8s start ./my.instance.dir.org
```

**Available levels:**
- `debug`: Detailed diagnostic information
- `info`: General informational messages
- `warn`: Warning messages only (default)
- `error`: Error messages only

**Example output:**
```
[INFO] === K8S START ===
[DEBUG] Namespace: myinstancedirorg
[INFO] Applying Kubernetes manifests...
[DEBUG] Applied manifest: namespace.yaml
[INFO] Waiting for instance to become ready...
[INFO] Instance started successfully
```

---

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

---

### Backup User Data
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

---

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

---

## Development

### Project Structure
```
openslides-cli/
├── cmd/
│   └── osmanage/                   # Main entry point
│       ├── main.go
│       └── main_test.go
├── internal/
│   ├── constants/                  # Project-wide constants
│   │   └── constants.go
│   ├── instance/                   # Instance management
│   │   ├── config/
│   │   │   ├── config.go
│   │   │   └── config_test.go
│   │   ├── create/
│   │   │   ├── create.go
│   │   │   └── create_test.go
│   │   ├── remove/
│   │   │   ├── remove.go
│   │   │   └── remove_test.go
│   │   └── setup/
│   │       ├── setup.go
│   │       └── setup_test.go
│   ├── k8s/                        # Kubernetes operations
│   │   ├── actions/
│   │   │   ├── apply.go
│   │   │   ├── cluster_status.go
│   │   │   ├── cluster_status_test.go
│   │   │   ├── health.go
│   │   │   ├── health_check.go
│   │   │   ├── health_check_test.go
│   │   │   ├── helpers.go
│   │   │   ├── helpers_test.go
│   │   │   ├── scale.go
│   │   │   ├── start.go
│   │   │   ├── stop.go
│   │   │   ├── update_backendmanage.go
│   │   │   └── update_instance.go
│   │   └── client/
│   │       └── client.go
│   ├── manage/                     # Backend action commands
│   │   ├── actions/
│   │   │   ├── action/
│   │   │   │   └── action.go
│   │   │   ├── createuser/
│   │   │   │   └── createuser.go
│   │   │   ├── get/
│   │   │   │   ├── get.go
│   │   │   │   └── get_test.go
│   │   │   ├── initialdata/
│   │   │   │   └── initialdata.go
│   │   │   ├── integration_test.go
│   │   │   ├── migrations/
│   │   │   │   ├── migrations.go
│   │   │   │   └── migrations_test.go
│   │   │   ├── set/
│   │   │   │   ├── set.go
│   │   │   │   └── set_test.go
│   │   │   └── setpassword/
│   │   │       └── setpassword.go
│   │   └── client/
│   │       ├── client.go
│   │       └── client_test.go
│   ├── logger/                     # Logging utilities
│   │   ├── logger.go
│   │   └── logger_test.go
│   └── utils/                      # Common utilities
│       ├── utils.go
│       └── utils_test.go
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

---

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

---

### Building
```bash
# Development build (larger binary, debuggable)
go build -o osmanage ./cmd/osmanage

# Production build (smaller binary, optimized, no debug symbols)
CGO_ENABLED=0 go build -a -ldflags="-s -w" -o osmanage ./cmd/osmanage
```

---

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

---

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

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Support

- **Issues**: [GitHub Issues](https://github.com/OpenSlides/openslides-cli/issues)
- **Discussions**: [GitHub Discussions](https://github.com/OpenSlides/openslides-cli/discussions)