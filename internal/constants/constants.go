// Package constants defines project-wide constants used across osmanage.
// These represent the standard OpenSlides instance directory structure
// and file permissions.
package constants

import (
	"io/fs"
	"time"
)

// Instance directory structure
const (
	// NamespaceYAML inside the instance root directory is applied to create instance namespace
	NamespaceYAML string = "namespace.yaml"

	// StackDirName is the directory containing Kubernetes manifests
	StackDirName string = "stack"

	// DeploymentFileTemplate for OpenSlides deployment filenames inside the stack dir, i. e. autoupdate-deployment.yaml
	DeploymentFileTemplate string = "%s-deployment.yaml"

	// SecretsDirName is the directory containing sensitive files
	SecretsDirName string = "secrets"

	// AdminSecretsFile contains the superadmin password
	AdminSecretsFile string = "superadmin"

	// PgPasswordFile contains the PostgreSQL database password
	PgPasswordFile string = "postgres_password"

	// AuthTokenKey contains the authentication token secret
	AuthTokenKey string = "auth_token_key"

	// AuthCookieKey contains the cookie signing secret
	AuthCookieKey string = "auth_cookie_key"

	// InternalAuthPassword contains the internal service authentication password
	InternalAuthPassword string = "internal_auth_password"

	// TlsCertSecret is kubernetes secret name for HTTPS
	TlsCertSecret string = "tls-letsencrypt"

	// TlsCertSecretYAML is the manifest file for the kubernetes secret enabling HTTPS
	TlsCertSecretYAML string = "tls-letsencrypt-secret.yaml"

	// DefaultConfigFile is the filename used, if none is set in config file(s)
	DefaultConfigFile string = "os-config.yaml"

	// CertCertName is filename for the HTTPS certificate file
	CertCertName string = "cert_crt"

	// CertKeyName is filename for the HTTPS key file
	CertKeyName string = "cert_key"
)

// File permissions
const (
	// SecretsDirPerm is the permission for the secrets directory (owner only)
	SecretsDirPerm fs.FileMode = 0700

	// SecretFilePerm is the permission for secret files (owner read/write only)
	SecretFilePerm fs.FileMode = 0600

	// InstanceDirPerm is the permission for project root directory (owner + others read)
	InstanceDirPerm fs.FileMode = 0755

	// StackDirPerm is the permission for the stack directory (owner + others read)
	StackDirPerm fs.FileMode = 0755

	// StackFilePerm is the permission for manifest files (owner write, others read)
	StackFilePerm fs.FileMode = 0644
)

// Secret generation defaults
const (
	// DefaultSuperadminPasswordLength is the default length for superadmin passwords
	DefaultSuperadminPasswordLength int = 20

	// DefaultPostgresPasswordLength is the default length for database passwords
	DefaultPostgresPasswordLength int = 40

	// DefaultSecretBytesLength is the number of random bytes used for base64-encoded secrets.
	// These 32 bytes produce a 44-character base64 string used for:
	// - auth_token_key
	// - auth_cookie_key
	// - internal_auth_password
	DefaultSecretBytesLength int64 = 32

	// PasswordCharset defines allowed characters for randomly generated passwords.
	// Includes lowercase, uppercase, digits, and safe special characters.
	// Used for generating postgres_password and superadmin passwords.
	PasswordCharset string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]"
)

// Default timeouts for Kubernetes operations
const (
	DefaultInstanceTimeout   time.Duration = 3 * time.Minute // Wait for all instance pods to become ready
	DefaultDeploymentTimeout time.Duration = 3 * time.Minute // Wait for deployment rollout to complete
	DefaultNamespaceTimeout  time.Duration = 5 * time.Minute // Wait for namespace deletion (includes finalizers)
)

// constants for wait functions in health_check.go
const (
	// progress bar settings
	ProgressBarWidth int           = 40
	Saucer           string        = "█"
	SaucerPadding    string        = "░"
	BarStart         string        = "["
	BarEnd           string        = "]"
	ThrottleDuration time.Duration = 100 * time.Millisecond

	// wait function settings
	TickerDuration time.Duration = 2 * time.Second // checks health conditions every tick
	IconReady      string        = "✓"             // for pod/deployment status printouts
	IconNotReady   string        = "✗"
)

// OpenSlides K8s resource names and templates
const (
	// BackendmanageDeploymentName is the Kubernetes Deployment name for backendmanage
	BackendmanageDeploymentName string = "backendmanage"

	// BackendmanageContainerName is the container name within the backendmanage deployment
	BackendmanageContainerName string = "backendmanage"

	// BackendmanageImageTemplate is the format string for backendmanage container images.
	BackendmanageImageTemplate string = "%s/openslides-backend:%s"

	// BackendmanagePatchTemplate is the JSON patch template for updating the backendmanage image.
	BackendmanagePatchTemplate string = `{"spec":{"template":{"spec":{"containers":[{"name":"%s","image":"%s"}]}}}}`
)

// OpenSlides backend API endpoints and defaults
const (
	// BackendHTTPScheme is the HTTP scheme used for backend connections
	BackendHTTPScheme string = "http://"

	// BackendHandleRequestPath is the API endpoint for sending actions
	BackendHandleRequestPath string = "/internal/handle_request"

	// BackendMigrationsPath is the API endpoint for migrations commands
	BackendMigrationsPath string = "/internal/migrations"

	// BackendContentType is the Content-Type header for backend requests
	BackendContentType string = "application/json"
)

// PostgreSQL datastore environment variable keys (used by get command)
const (
	// EnvDatabaseHost is the environment variable for PostgreSQL host
	EnvDatabaseHost string = "DATABASE_HOST"

	// EnvDatabasePort is the environment variable for PostgreSQL port
	EnvDatabasePort string = "DATABASE_PORT"

	// EnvDatabaseUser is the environment variable for PostgreSQL user
	EnvDatabaseUser string = "DATABASE_USER"

	// EnvDatabaseName is the environment variable for PostgreSQL database name
	EnvDatabaseName string = "DATABASE_NAME"

	// EnvDatabasePasswordFile is the environment variable for PostgreSQL password file path
	EnvDatabasePasswordFile string = "DATABASE_PASSWORD_FILE"

	// EnvOpenSlidesDevelopment is the environment variable for development mode
	EnvOpenSlidesDevelopment string = "OPENSLIDES_DEVELOPMENT"
)

// PostgreSQL datastore environment variable values
const (
	// DevelopmentModeDisabled is the value to disable OpenSlides development mode
	DevelopmentModeDisabled string = "false"

	// DevelopmentModeEnabled is the value to enable OpenSlides development mode
	DevelopmentModeEnabled string = "true"
)

// OpenSlides datastore defaults
const (
	// DefaultOrganizationID is the organization ID in OpenSlides (always 1)
	DefaultOrganizationID int = 1

	// DefaultOrganizationFields are the default fields fetched for organization queries
	DefaultOrganizationFields string = "id,name"
)

// Migration command defaults and configuration
const (
	// DefaultMigrationProgressInterval is the default interval for checking migration progress
	DefaultMigrationProgressInterval time.Duration = 1 * time.Second

	// MigrationStatusRunning indicates a migration is currently in progress
	MigrationStatusRunning string = "migration_running"

	// MigrationMaxRetries is the maximum number of retry attempts for failed migration requests
	MigrationMaxRetries int = 5

	// MigrationRetryDelay is the delay between retry attempts
	MigrationRetryDelay time.Duration = 5 * time.Second

	// MigrationTotalTimeout is the maximum time allowed for all retry attempts
	MigrationTotalTimeout time.Duration = 3 * time.Minute
)

// Migration stats field names (for ordered output)
var MigrationStatsFields = []string{
	"current_migration_index",
	"target_migration_index",
	"positions",
	"events",
	"partially_migrated_positions",
	"fully_migrated_positions",
	"status",
}
