package main

import (
	"testing"
)

func TestRootCmd(t *testing.T) {
	cmd := RootCmd()

	if cmd.Use != "osmanage" {
		t.Errorf("Expected Use 'osmanage', got %s", cmd.Use)
	}

	// Check global flags
	if !cmd.PersistentFlags().HasFlags() {
		t.Error("Expected persistent flags to be set")
	}

	logLevelFlag := cmd.PersistentFlags().Lookup("log-level")
	if logLevelFlag == nil {
		t.Fatal("Expected log-level flag to exist")
		return
	}
	if logLevelFlag.DefValue != "warn" {
		t.Errorf("Expected default log-level 'info', got %s", logLevelFlag.DefValue)
	}

	// Check subcommands exist
	expectedCommands := []string{
		"create-user",
		"initial-data",
		"set-password",
		"set",
		"get",
		"action",
		"migrations",
		"setup",
		"config",
	}

	commands := cmd.Commands()
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("Expected command %s to exist", expected)
		}
	}
}

func TestRunClient(t *testing.T) {
	// Test with invalid command should return non-zero
	// We can't easily test this without mocking os.Exit
	// but we can at least verify the function exists and compiles
	_ = RunClient
}
