package ui

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestParseDueDate(t *testing.T) {
	now := time.Date(2026, time.October, 13, 10, 30, 0, 0, time.Local)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "month and day", input: "oct 15", want: "2026-10-15"},
		{name: "next weekday", input: "thursday", want: "2026-10-15"},
		{name: "relative date", input: "tomorrow", want: "2026-10-14"},
		{name: "ISO date", input: "2027-01-20", want: "2027-01-20"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDueDate(tt.input, now)
			if err != nil {
				t.Fatalf("parseDueDate(%q) returned error: %v", tt.input, err)
			}
			if got.Format("2006-01-02") != tt.want {
				t.Fatalf("parseDueDate(%q) = %s, want %s", tt.input, got.Format("2006-01-02"), tt.want)
			}
		})
	}
}

func TestParseDueDateRejectsUnknownText(t *testing.T) {
	_, err := parseDueDate("definitely not a date", time.Date(2026, time.October, 13, 10, 30, 0, 0, time.Local))
	if !errors.Is(err, errInvalidDueDate) {
		t.Fatalf("parseDueDate() error = %v, want errInvalidDueDate", err)
	}
}

func TestTaskTextWithDueDate(t *testing.T) {
	dueDate := time.Date(2026, time.October, 15, 0, 0, 0, 0, time.Local)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "adds tag", input: "Call mom @phone", want: "Call mom @phone due:2026-10-15"},
		{name: "replaces existing tag", input: "Call mom due:2025-01-01 @phone", want: "Call mom @phone due:2026-10-15"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := taskTextWithDueDate(tt.input, dueDate); got != tt.want {
				t.Fatalf("taskTextWithDueDate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAddTaskFormSavesSeparateDueDate(t *testing.T) {
	todoFile := t.TempDir() + "/todo.txt"
	if err := os.WriteFile(todoFile, nil, 0o600); err != nil {
		t.Fatalf("create todo file: %v", err)
	}

	model, err := NewModel(todoFile, DefaultAppConfig())
	if err != nil {
		t.Fatalf("NewModel() error: %v", err)
	}
	defer model.Cleanup()

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if model.viewMode != ViewAdd {
		t.Fatalf("view mode = %v, want ViewAdd", model.viewMode)
	}
	if view := model.View(); !strings.Contains(view, "Due date (optional)") {
		t.Fatalf("add form does not contain a separate due-date field: %q", view)
	}
	model.Update(tea.KeyMsg{Type: tea.KeyTab})
	if model.activeAddInput != addInputDueDate || !model.dueDateInput.Focused() {
		t.Fatal("Tab should move focus to the due-date field")
	}

	model.textInput.SetValue("Call mom @phone")
	model.dueDateInput.SetValue("2027-01-20")
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if model.tasks.Len() != 1 {
		t.Fatalf("task count = %d, want 1", model.tasks.Len())
	}
	savedTask := model.tasks.Get(0)
	if got := savedTask.String(); got != "Call mom @phone due:2027-01-20" {
		t.Fatalf("saved task = %q, want separate due date to be appended", got)
	}

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	model.textInput.SetValue("Task without a deadline")
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if model.tasks.Len() != 2 {
		t.Fatalf("task count = %d, want optional due date to allow a second task", model.tasks.Len())
	}
	undatedTask := model.tasks.Get(1)
	if got := undatedTask.String(); got != "Task without a deadline" {
		t.Fatalf("saved task = %q, want no due tag when field is blank", got)
	}
}

func TestAddTaskFormKeepsInvalidDueDateOpen(t *testing.T) {
	todoFile := t.TempDir() + "/todo.txt"
	if err := os.WriteFile(todoFile, nil, 0o600); err != nil {
		t.Fatalf("create todo file: %v", err)
	}

	model, err := NewModel(todoFile, DefaultAppConfig())
	if err != nil {
		t.Fatalf("NewModel() error: %v", err)
	}
	defer model.Cleanup()

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	model.textInput.SetValue("Call mom")
	model.dueDateInput.SetValue("definitely not a date")
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if model.viewMode != ViewAdd {
		t.Fatalf("view mode = %v, want invalid date to keep ViewAdd open", model.viewMode)
	}
	if model.activeAddInput != addInputDueDate || model.addInputError == "" {
		t.Fatal("invalid date should focus the due-date field and show an error")
	}
	if model.tasks.Len() != 0 {
		t.Fatalf("task count = %d, want 0 after invalid date", model.tasks.Len())
	}
}
