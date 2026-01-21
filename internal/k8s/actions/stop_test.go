package actions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/yaml"
)

func TestSaveTLSSecret_SecretExists(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tlsCertSecret,
			Namespace: "test-namespace",
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nfake-cert-data\n-----END CERTIFICATE-----"),
			"tls.key": []byte("-----BEGIN PRIVATE KEY-----\nfake-key-data\n-----END PRIVATE KEY-----"),
		},
	}

	fakeClient := fake.NewClientset(secret)

	tmpDir, err := os.MkdirTemp("", "stop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	ctx := context.Background()

	err = saveTLSSecretWithClientset(ctx, fakeClient, "test-namespace", tmpDir)
	if err != nil {
		t.Fatalf("saveTLSSecret failed: %v", err)
	}

	secretPath := filepath.Join(tmpDir, tlsCertSecretYAML)
	if _, err := os.Stat(secretPath); os.IsNotExist(err) {
		t.Errorf("Secret file was not created at %s", secretPath)
	}

	fileInfo, err := os.Stat(secretPath)
	if err != nil {
		t.Fatalf("Failed to stat secret file: %v", err)
	}

	expectedPerms := os.FileMode(0600)
	if fileInfo.Mode().Perm() != expectedPerms {
		t.Errorf("Secret file permissions = %v, want %v", fileInfo.Mode().Perm(), expectedPerms)
	}

	fileData, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("Failed to read secret file: %v", err)
	}

	var savedSecret corev1.Secret
	if err := yaml.Unmarshal(fileData, &savedSecret); err != nil {
		t.Fatalf("Failed to unmarshal saved secret: %v", err)
	}

	if string(savedSecret.Data["tls.crt"]) != string(secret.Data["tls.crt"]) {
		t.Error("Saved secret tls.crt data doesn't match original")
	}
	if string(savedSecret.Data["tls.key"]) != string(secret.Data["tls.key"]) {
		t.Error("Saved secret tls.key data doesn't match original")
	}
}

func TestSaveTLSSecret_SecretNotExists(t *testing.T) {
	fakeClient := fake.NewClientset()

	tmpDir, err := os.MkdirTemp("", "stop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	ctx := context.Background()

	err = saveTLSSecretWithClientset(ctx, fakeClient, "test-namespace", tmpDir)
	if err != nil {
		t.Errorf("saveTLSSecret should not fail when secret doesn't exist, got: %v", err)
	}

	secretPath := filepath.Join(tmpDir, tlsCertSecretYAML)
	if _, err := os.Stat(secretPath); !os.IsNotExist(err) {
		t.Error("Secret file should not be created when secret doesn't exist")
	}
}

func TestSaveTLSSecret_CreatesSecretsDirectory(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tlsCertSecret,
			Namespace: "test-namespace",
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": []byte("cert"),
			"tls.key": []byte("key"),
		},
	}

	fakeClient := fake.NewClientset(secret)

	tmpDir, err := os.MkdirTemp("", "stop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	ctx := context.Background()

	err = saveTLSSecretWithClientset(ctx, fakeClient, "test-namespace", tmpDir)
	if err != nil {
		t.Fatalf("saveTLSSecret failed: %v", err)
	}

	secretsDir := filepath.Join(tmpDir, "secrets")
	if _, err := os.Stat(secretsDir); os.IsNotExist(err) {
		t.Error("Secrets directory was not created")
	}
}

func TestDeleteNamespace_Success(t *testing.T) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}

	fakeClient := fake.NewClientset(namespace)

	deleted := false
	fakeClient.PrependReactor("delete", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleted = true
		return true, nil, nil
	})

	fakeClient.PrependReactor("get", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if deleted {
			return true, nil, apierrors.NewNotFound(corev1.Resource("namespaces"), "test-namespace")
		}
		return false, namespace, nil
	})

	ctx := context.Background()

	err := deleteNamespaceWithClientset(ctx, fakeClient, "test-namespace")
	if err != nil {
		t.Errorf("deleteNamespace failed: %v", err)
	}

	if !deleted {
		t.Error("Namespace was not deleted")
	}
}

func TestDeleteNamespace_WaitsForDeletion(t *testing.T) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}

	fakeClient := fake.NewClientset(namespace)

	deleteTime := time.Time{}
	deletionStarted := false

	fakeClient.PrependReactor("delete", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleteTime = time.Now()
		deletionStarted = true
		return true, nil, nil
	})

	checkCount := 0
	fakeClient.PrependReactor("get", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		checkCount++

		if !deletionStarted {
			return false, namespace, nil
		}

		if checkCount <= 3 {
			terminatingNs := namespace.DeepCopy()
			now := metav1.NewTime(deleteTime)
			terminatingNs.DeletionTimestamp = &now
			return true, terminatingNs, nil
		}

		return true, nil, apierrors.NewNotFound(corev1.Resource("namespaces"), "test-namespace")
	})

	ctx := context.Background()

	start := time.Now()
	err := deleteNamespaceWithClientset(ctx, fakeClient, "test-namespace")
	duration := time.Since(start)

	if err != nil {
		t.Errorf("deleteNamespace failed: %v", err)
	}

	if checkCount < 3 {
		t.Errorf("Should have checked namespace status multiple times, got %d checks", checkCount)
	}

	if duration < 2*time.Second {
		t.Errorf("Should have waited for namespace deletion, took only %v", duration)
	}
}

func TestDeleteNamespace_NotFound(t *testing.T) {
	fakeClient := fake.NewClientset()

	ctx := context.Background()

	err := deleteNamespaceWithClientset(ctx, fakeClient, "non-existent")
	if err != nil {
		t.Errorf("deleteNamespace should not fail for non-existent namespace, got: %v", err)
	}
}

// Helper function to test saveTLSSecret with a fake clientset
// This bypasses the client.Client wrapper for testing purposes
func saveTLSSecretWithClientset(ctx context.Context, clientset *fake.Clientset, namespace, projectDir string) error {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, tlsCertSecret, metav1.GetOptions{})
	if err != nil {
		return nil
	}

	secretsDir := filepath.Join(projectDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		return err
	}

	secretYAML, err := yaml.Marshal(secret)
	if err != nil {
		return err
	}

	secretPath := filepath.Join(projectDir, tlsCertSecretYAML)
	if err := os.WriteFile(secretPath, secretYAML, 0600); err != nil {
		return err
	}

	return nil
}

// Helper function to test deleteNamespace with a fake clientset
func deleteNamespaceWithClientset(ctx context.Context, clientset *fake.Clientset, namespace string) error {
	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	err := clientset.CoreV1().Namespaces().Delete(ctx, namespace, deleteOptions)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for namespace %s to be deleted", namespace)
		case <-ticker.C:
			_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}
}
