package ui

import (
	"sort"
	"strings"
	"time"

	todotxt "github.com/1set/todotxt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuucu/todotui/pkg/domain"
)

type taskSort int

const (
	sortByDueDate taskSort = iota
	sortByPriority
)

// moveSelectedTask swaps the selected todo with its visible neighbor. Tasks
// excluded by the current filter retain their slots in the underlying file.
func (m *Model) moveSelectedTask(delta int) tea.Cmd {
	selected := m.taskList.selected
	target := selected + delta
	if selected < 0 || selected >= m.filteredTasks.Len() || target < 0 || target >= m.filteredTasks.Len() {
		return nil
	}

	selectedDomainTask := m.filteredTasks.Get(selected)
	targetDomainTask := m.filteredTasks.Get(target)
	selectedTask := selectedDomainTask.ToTodoTxtTask()
	targetTask := targetDomainTask.ToTodoTxtTask()
	selectedIndex := m.taskIndexByPointer(selectedTask)
	targetIndex := m.taskIndexByPointer(targetTask)
	if selectedIndex < 0 || targetIndex < 0 {
		return m.setStatusMessage("❌ Failed to reorder task", 3*time.Second)
	}

	m.tasks[selectedIndex], m.tasks[targetIndex] = m.tasks[targetIndex], m.tasks[selectedIndex]
	cmd := m.saveAndRefresh()
	m.taskList.SetSelectedIndexPreserveScroll(target)
	return cmd
}

// sortVisibleTasks performs a one-time stable sort of the visible todos and
// writes that order back to their current slots in the full task list.
func (m *Model) sortVisibleTasks(by taskSort) tea.Cmd {
	if m.filteredTasks.Len() < 2 {
		return nil
	}

	selectedDomainTask, ok := m.filteredTasks.SafeGet(m.taskList.selected)
	if !ok {
		return nil
	}
	selectedTask := selectedDomainTask.ToTodoTxtTask()
	ordered := append(domain.Tasks(nil), m.filteredTasks...)

	switch by {
	case sortByDueDate:
		sort.SliceStable(ordered, func(i, j int) bool {
			leftHasDate := ordered[i].HasDueDate()
			rightHasDate := ordered[j].HasDueDate()
			if leftHasDate != rightHasDate {
				return leftHasDate
			}
			if !leftHasDate {
				return false
			}
			return ordered[i].GetDueDate().Before(ordered[j].GetDueDate())
		})
	case sortByPriority:
		sort.SliceStable(ordered, func(i, j int) bool {
			leftGroup, leftRank, leftName := m.prioritySortRank(ordered[i].GetPriority())
			rightGroup, rightRank, rightName := m.prioritySortRank(ordered[j].GetPriority())
			if leftGroup != rightGroup {
				return leftGroup < rightGroup
			}
			if leftRank != rightRank {
				return leftRank < rightRank
			}
			return strings.Compare(leftName, rightName) < 0
		})
	}

	visible := make(map[*todotxt.Task]struct{}, m.filteredTasks.Len())
	for _, task := range m.filteredTasks {
		visible[task.ToTodoTxtTask()] = struct{}{}
	}

	positions := make([]int, 0, len(visible))
	for i := range m.tasks {
		if _, ok := visible[m.tasks[i].ToTodoTxtTask()]; ok {
			positions = append(positions, i)
		}
	}
	if len(positions) != len(ordered) {
		return m.setStatusMessage("❌ Failed to sort tasks", 3*time.Second)
	}
	for i, position := range positions {
		m.tasks[position] = ordered[i]
	}

	cmd := m.saveAndRefresh()
	for i := range m.filteredTasks {
		if m.filteredTasks[i].ToTodoTxtTask() == selectedTask {
			m.taskList.SetSelectedIndexPreserveScroll(i)
			break
		}
	}

	label := "due date"
	if by == sortByPriority {
		label = "priority"
	}
	return tea.Batch(cmd, m.setStatusMessage("✅ Sorted by "+label, 2*time.Second))
}

func (m *Model) taskIndexByPointer(target *todotxt.Task) int {
	for i := range m.tasks {
		if m.tasks[i].ToTodoTxtTask() == target {
			return i
		}
	}
	return -1
}

// prioritySortRank honors configured priority order. Unknown priorities sort
// after configured priorities, while tasks without a priority sort last.
func (m *Model) prioritySortRank(priority string) (group int, rank int, name string) {
	if priority == "" {
		return 2, 0, ""
	}
	for i, configured := range m.appConfig.PriorityLevels {
		if configured == priority {
			return 0, i, priority
		}
	}
	return 1, 0, priority
}
