package ui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/olebedev/when"
	"github.com/olebedev/when/rules"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"
	"github.com/yuucu/todotui/pkg/domain"
)

var errInvalidDueDate = errors.New("invalid due date")

// parseDueDate resolves an ISO date or an English natural-language date using
// now as its reference point. Month/day inputs inherit the reference year;
// bare weekdays resolve to their next occurrence.
func parseDueDate(value string, now time.Time) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errInvalidDueDate
	}

	if parsed, err := time.ParseInLocation(domain.DateFormat, value, now.Location()); err == nil {
		return parsed, nil
	}

	parser := when.New(nil)
	parser.Add(
		en.Weekday(rules.Override),
		en.CasualDate(rules.Override),
		en.Deadline(rules.Override),
		en.PastTime(rules.Override),
		en.ExactMonthDate(rules.Override),
	)
	parser.Add(common.All...)

	result, err := parser.Parse(value, now)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %v", errInvalidDueDate, err)
	}
	if result == nil {
		return time.Time{}, errInvalidDueDate
	}

	return result.Time, nil
}

// taskTextWithDueDate makes the separate due-date field authoritative while
// avoiding duplicate due: tags if one was also typed in the description.
func taskTextWithDueDate(taskText string, dueDate time.Time) string {
	parts := strings.Fields(taskText)
	withoutDueDate := make([]string, 0, len(parts)+1)
	for _, part := range parts {
		if !strings.HasPrefix(part, domain.TaskFieldDuePrefix) {
			withoutDueDate = append(withoutDueDate, part)
		}
	}

	return strings.Join(append(withoutDueDate,
		domain.TaskFieldDuePrefix+dueDate.Format(domain.DateFormat)), " ")
}
