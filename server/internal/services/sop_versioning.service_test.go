package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
)

func TestCompareSteps_NoChanges(t *testing.T) {
	instructions := "Do the thing"
	estTime := 30
	stationID := uuid.New()

	from := &models.SOPStep{
		Title:                "Cut boards",
		Instructions:         &instructions,
		EstimatedTimeMinutes: &estTime,
		RequiresApproval:     false,
		StationID:            &stationID,
	}
	to := &models.SOPStep{
		Title:                "Cut boards",
		Instructions:         &instructions,
		EstimatedTimeMinutes: &estTime,
		RequiresApproval:     false,
		StationID:            &stationID,
	}

	changes := compareSteps(from, to)
	if len(changes) != 0 {
		t.Errorf("Expected no changes, got %d: %v", len(changes), changes)
	}
}

func TestCompareSteps_TitleChanged(t *testing.T) {
	from := &models.SOPStep{Title: "Cut boards"}
	to := &models.SOPStep{Title: "Rip boards to width"}

	changes := compareSteps(from, to)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}
	if changes[0].Field != "title" {
		t.Errorf("Expected field 'title', got %q", changes[0].Field)
	}
	if changes[0].OldValue != "Cut boards" {
		t.Errorf("Expected old value 'Cut boards', got %q", changes[0].OldValue)
	}
	if changes[0].NewValue != "Rip boards to width" {
		t.Errorf("Expected new value 'Rip boards to width', got %q", changes[0].NewValue)
	}
}

func TestCompareSteps_InstructionsChanged(t *testing.T) {
	old := "Use table saw"
	new := "Use table saw with crosscut sled"
	from := &models.SOPStep{Title: "Cut", Instructions: &old}
	to := &models.SOPStep{Title: "Cut", Instructions: &new}

	changes := compareSteps(from, to)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}
	if changes[0].Field != "instructions" {
		t.Errorf("Expected field 'instructions', got %q", changes[0].Field)
	}
}

func TestCompareSteps_InstructionsNilToSet(t *testing.T) {
	new := "Use table saw"
	from := &models.SOPStep{Title: "Cut", Instructions: nil}
	to := &models.SOPStep{Title: "Cut", Instructions: &new}

	changes := compareSteps(from, to)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}
	if changes[0].Field != "instructions" {
		t.Errorf("Expected field 'instructions', got %q", changes[0].Field)
	}
	if changes[0].OldValue != "" {
		t.Errorf("Expected empty old value, got %q", changes[0].OldValue)
	}
}

func TestCompareSteps_MultipleChanges(t *testing.T) {
	oldInstr := "old"
	newInstr := "new"
	oldTime := 10
	newTime := 20
	from := &models.SOPStep{
		Title:                "Step 1",
		Instructions:         &oldInstr,
		EstimatedTimeMinutes: &oldTime,
		RequiresApproval:     false,
	}
	to := &models.SOPStep{
		Title:                "Step 1 (updated)",
		Instructions:         &newInstr,
		EstimatedTimeMinutes: &newTime,
		RequiresApproval:     true,
	}

	changes := compareSteps(from, to)
	if len(changes) != 4 {
		t.Errorf("Expected 4 changes, got %d: %v", len(changes), changes)
	}

	fieldSet := make(map[string]bool)
	for _, c := range changes {
		fieldSet[c.Field] = true
	}
	expectedFields := []string{"title", "instructions", "estimatedTimeMinutes", "requiresApproval"}
	for _, f := range expectedFields {
		if !fieldSet[f] {
			t.Errorf("Expected field %q in changes, not found", f)
		}
	}
}

func TestCompareSteps_StationIDChanged(t *testing.T) {
	oldStation := uuid.New()
	newStation := uuid.New()
	from := &models.SOPStep{Title: "Cut", StationID: &oldStation}
	to := &models.SOPStep{Title: "Cut", StationID: &newStation}

	changes := compareSteps(from, to)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}
	if changes[0].Field != "stationId" {
		t.Errorf("Expected field 'stationId', got %q", changes[0].Field)
	}
}

func TestCompareSteps_StationIDNilToSet(t *testing.T) {
	newStation := uuid.New()
	from := &models.SOPStep{Title: "Cut", StationID: nil}
	to := &models.SOPStep{Title: "Cut", StationID: &newStation}

	changes := compareSteps(from, to)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}
	if changes[0].Field != "stationId" {
		t.Errorf("Expected field 'stationId', got %q", changes[0].Field)
	}
}

func TestCompareSteps_LinkedSOPTemplateIDChanged(t *testing.T) {
	oldLink := 5
	newLink := 10
	from := &models.SOPStep{Title: "Step", LinkedSOPTemplateID: &oldLink}
	to := &models.SOPStep{Title: "Step", LinkedSOPTemplateID: &newLink}

	changes := compareSteps(from, to)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(changes))
	}
	if changes[0].Field != "linkedSopTemplateId" {
		t.Errorf("Expected field 'linkedSopTemplateId', got %q", changes[0].Field)
	}
}

func TestPtrToString(t *testing.T) {
	s := "hello"
	if ptrToString(&s) != "hello" {
		t.Errorf("Expected 'hello', got %q", ptrToString(&s))
	}
	if ptrToString(nil) != "" {
		t.Errorf("Expected empty string for nil, got %q", ptrToString(nil))
	}
}

func TestPtrToInt(t *testing.T) {
	i := 42
	if ptrToInt(&i) != 42 {
		t.Errorf("Expected 42, got %d", ptrToInt(&i))
	}
	if ptrToInt(nil) != 0 {
		t.Errorf("Expected 0 for nil, got %d", ptrToInt(nil))
	}
}

func TestPtrUUIDToString(t *testing.T) {
	u := uuid.New()
	if ptrUUIDToString(&u) != u.String() {
		t.Errorf("Expected %q, got %q", u.String(), ptrUUIDToString(&u))
	}
	if ptrUUIDToString(nil) != "" {
		t.Errorf("Expected empty string for nil, got %q", ptrUUIDToString(nil))
	}
}
