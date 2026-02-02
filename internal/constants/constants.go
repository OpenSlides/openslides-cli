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
	// StackDirName is the directory containing Kubernetes manifests
	StackDirName string = "stack"

	// SecretsDirName is the directory containing sensitive files
	SecretsDirName string = "secrets"

	// TLS certificate secret name and filename for HTTPS
	TlsCertSecret     string = "tls-letsencrypt"             // Kubernetes Secret resource name
	TlsCertSecretYAML string = "tls-letsencrypt-secret.yaml" // Secret manifest filename

	// The template string for OpenSlides deployment filenames i. e. autoupdate-deployment.yaml
	DeploymentFileTemplate string = "%s-deployment.yaml"
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

// Password generation defaults for setup
const (
	DefaultSuperadminPasswordLength int = 20
	DefaultPostgresPasswordLength   int = 40
)

// Certificate file names (for HTTPS) for setup
const (
	CertCertName string = "cert_crt" // Certificate file
	CertKeyName  string = "cert_key" // Private key file
)

// Secret file names
const (
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
)

// Default timeouts for Kubernetes operations
const (
	DefaultInstanceTimeout   time.Duration = 3 * time.Minute // Wait for all instance pods to become ready
	DefaultDeploymentTimeout time.Duration = 3 * time.Minute // Wait for deployment rollout to complete
	DefaultNamespaceTimeout  time.Duration = 5 * time.Minute // Wait for namespace deletion (includes finalizers)
)

// OpenSlides K8s resource names
const (
	BackendmanageDeploymentName string = "backendmanage"
	BackendmanageContainerName  string = "backendmanage"
)
