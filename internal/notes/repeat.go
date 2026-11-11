package notes

import (
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

// Repeat is how a note comes back after it is checked.
type Repeat struct {
	Kind    string `json:"kind"`              // "daily" | "weekdays" | "weekly" | "monthly"
	Weekday int    `json:"weekday,omitempty"` // weekly: 0=Sunday … 6=Saturday
	Day     int    `json:"day,omitempty"`     // monthly: 1–31; past the month's end means its last day
}

func (r Repeat) Validate() error {
	switch r.Kind {
	case "daily", "weekdays":
		return nil
	case "weekly":
		if r.Weekday < 0 || r.Weekday > 6 {
			return fmt.Errorf("dia da semana inválido: %d", r.Weekday)
		}
		return nil
	case "monthly":
		if r.Day < 1 || r.Day > 31 {
			return fmt.Errorf("dia do mês inválido: %d", r.Day)
		}
		return nil
	}
	return fmt.Errorf("repetição desconhecida: %q", r.Kind)
}

func (r Repeat) matches(d time.Time) bool {
	switch r.Kind {
	case "daily":
		return true
	case "weekdays":
		return d.Weekday() != time.Saturday && d.Weekday() != time.Sunday
	case "weekly":
		return int(d.Weekday()) == r.Weekday
	case "monthly":
		last := time.Date(d.Year(), d.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		return d.Day() == min(r.Day, last)
	}
	return false
}

// Next is the first day after `after` that the repeat falls on. Dates are days, so the
// arithmetic runs in UTC and no time zone shifts them.
func Next(r Repeat, after string) (string, error) {
	if err := r.Validate(); err != nil {
		return "", err
	}
	d, err := time.Parse(dateLayout, after)
	if err != nil {
		return "", fmt.Errorf("data inválida: %q", after)
	}
	for range 62 {
		d = d.AddDate(0, 0, 1)
		if r.matches(d) {
			return d.Format(dateLayout), nil
		}
	}
	return "", fmt.Errorf("sem próxima data para %+v depois de %s", r, after)
}
