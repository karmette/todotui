package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	todotxt "github.com/1set/todotxt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuucu/todotui/pkg/domain"
)

func newReorderTestModel(t *testing.T, taskStrings ...string) *Model {
	t.Helper()

	var taskList todotxt.TaskList
	for _, taskString := range taskStrings {
		task, err := todotxt.ParseTask(taskString)
		if err != nil {
			t.Fatalf("ParseTask(%q): %v", taskString, err)
		}
		taskList = append(taskList, *task)
	}

	path := filepath.Join(t.TempDir(), "todo.txt")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("create todo file: %v", err)
	}
	config := DefaultAppConfig()
	theme := GetTheme(config.Theme)
	model := &Model{
		appConfig:    config,
		tasks:        domain.NewTasks(taskList),
		activePane:   paneTask,
		viewMode:     ViewTask,
		width:        DefaultTerminalWidth,
		height:       DefaultTerminalHeight,
		currentTheme: &theme,
		todoFilePath: path,
	}
	model.filterList.SetTheme(&theme)
	model.taskList.SetTheme(&theme)
	model.taskList.SetTaskList(true)
	model.updatePaneSizes()
	model.refreshLists()
	selectTestFilter(t, model, FilterAllTasks)
	return model
}

func selectTestFilter(t *testing.T, model *Model, name string) {
	t.Helper()
	for i, filter := range model.filters {
		if filter.name == name {
			model.filterList.SetSelectedIndex(i)
			model.refreshTaskList()
			return
		}
	}
	t.Fatalf("filter %q not found", name)
}

func taskStrings(tasks domain.Tasks) []string {
	result := make([]string, tasks.Len())
	for i := range tasks {
		result[i] = tasks[i].String()
	}
	return result
}

func TestSortMenuOnlyOpensInTodoPane(t *testing.T) {
	model := newReorderTestModel(t, "First", "Second")
	ctrlS := tea.KeyMsg{Type: tea.KeyCtrlS}

	model.activePane = paneFilter
	model.Update(ctrlS)
	if model.viewMode == ViewSort {
		t.Fatal("Ctrl+S must not open sorting from the filter pane")
	}

	model.activePane = paneTask
	model.Update(ctrlS)
	if model.viewMode != ViewSort {
		t.Fatal("Ctrl+S should open sorting from the todo pane")
	}
	if view := model.renderCombinedHelpStatusBar(); !strings.Contains(view, "Sort todos:") {
		t.Fatalf("sort options missing from bottom bar: %q", view)
	}

	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if model.viewMode != ViewTask {
		t.Fatal("Esc should close the sort menu")
	}
}

func TestSortVisibleTasksByDueDate(t *testing.T) {
	model := newReorderTestModel(t,
		"No date",
		"Later due:2030-02-01",
		"Sooner due:2030-01-01",
	)
	model.taskList.SetSelectedIndex(1)

	model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	want := []string{"Sooner due:2030-01-01", "Later due:2030-02-01", "No date"}
	assertTaskOrder(t, model.tasks, want)
	if model.taskList.selected != 1 {
		t.Fatalf("selected index = %d, want selected task to remain at its new index 1", model.taskList.selected)
	}
	assertSavedTaskOrder(t, model.todoFilePath, want)
}

func TestSortVisibleTasksByConfiguredPriority(t *testing.T) {
	model := newReorderTestModel(t,
		"No priority",
		"(C) Low",
		"(A) High",
		"(B) Medium",
	)

	model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})

	want := []string{"(A) High", "(B) Medium", "(C) Low", "No priority"}
	assertTaskOrder(t, model.tasks, want)
	assertSavedTaskOrder(t, model.todoFilePath, want)
}

func TestShiftJAndKReorderOnlyVisibleTodos(t *testing.T) {
	model := newReorderTestModel(t,
		"First +work",
		"Hidden +home",
		"Second +work",
	)
	selectTestFilter(t, model, "  +work")

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("J")})
	wantMoved := []string{"Second +work", "Hidden +home", "First +work"}
	assertTaskOrder(t, model.tasks, wantMoved)
	if model.taskList.selected != 1 {
		t.Fatalf("selected index after Shift+J = %d, want 1", model.taskList.selected)
	}

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	wantRestored := []string{"First +work", "Hidden +home", "Second +work"}
	assertTaskOrder(t, model.tasks, wantRestored)
	if model.taskList.selected != 0 {
		t.Fatalf("selected index after Shift+K = %d, want 0", model.taskList.selected)
	}
	assertSavedTaskOrder(t, model.todoFilePath, wantRestored)
}

func TestShiftJDoesNothingInFilterPane(t *testing.T) {
	model := newReorderTestModel(t, "First", "Second")
	model.activePane = paneFilter

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("J")})
	assertTaskOrder(t, model.tasks, []string{"First", "Second"})
}

func assertTaskOrder(t *testing.T, tasks domain.Tasks, want []string) {
	t.Helper()
	got := taskStrings(tasks)
	if len(got) != len(want) {
		t.Fatalf("task count = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("task order = %v, want %v", got, want)
		}
	}
}

func assertSavedTaskOrder(t *testing.T, path string, want []string) {
	t.Helper()
	taskList, err := todotxt.LoadFromPath(path)
	if err != nil {
		t.Fatalf("load saved tasks: %v", err)
	}
	assertTaskOrder(t, domain.NewTasks(taskList), want)
}
