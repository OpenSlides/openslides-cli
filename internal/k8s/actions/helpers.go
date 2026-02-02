package actions

import (
	"os"
	"path/filepath"
	"strings"
)

// extractNamespace gets the namespace from instance directory path
// Example: "/real/path/to/my.instance.dir.url" -> "myinstancedirurl"
func extractNamespace(instanceDir string) string {
	dirName := filepath.Base(instanceDir)
	namespace := strings.ReplaceAll(dirName, ".", "")
	return namespace
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isYAMLFile checks if filename has YAML extension
func isYAMLFile(filename string) bool {
	ext := filepath.Ext(filename)
	return ext == ".yaml" || ext == ".yml"
}
