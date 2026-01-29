package set

import (
	"testing"
)

func TestActionMap(t *testing.T) {
	// Verify action map has expected entries
	expectedActions := []string{
		"agenda_item",
		"committee",
		"group",
		"meeting",
		"motion",
		"organization_tag",
		"organization",
		"projector",
		"theme",
		"topic",
		"user",
	}

	if len(actionMap) != len(expectedActions) {
		t.Errorf("Expected %d actions, got %d", len(expectedActions), len(actionMap))
	}

	for _, action := range expectedActions {
		if _, ok := actionMap[action]; !ok {
			t.Errorf("Expected action %s to be in actionMap", action)
		}
	}

	// Verify mappings are correct
	if actionMap["user"] != "user.update" {
		t.Errorf("Expected user -> user.update, got %s", actionMap["user"])
	}
	if actionMap["meeting"] != "meeting.update" {
		t.Errorf("Expected meeting -> meeting.update, got %s", actionMap["meeting"])
	}
}

func TestHelpTextActionList(t *testing.T) {
	list := helpTextActionList()

	if len(list) != len(actionMap) {
		t.Errorf("Expected %d actions in help text, got %d", len(actionMap), len(list))
	}

	// Should be sorted
	for i := 1; i < len(list); i++ {
		if list[i-1] > list[i] {
			t.Errorf("List not sorted: %s should come after %s", list[i-1], list[i])
		}
	}
}
