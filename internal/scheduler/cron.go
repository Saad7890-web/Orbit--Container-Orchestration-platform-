package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Saad7890-web/orbit/internal/models"
)

type CronSchedule struct {
	spec       string
	timezone   *time.Location
	hasSeconds bool

	seconds *cronField
	minutes *cronField
	hours   *cronField
	dom     *cronField
	months  *cronField
	dow     *cronField
}

func ParseSchedule(s models.Schedule) (*CronSchedule, error) {
	spec := strings.TrimSpace(s.Cron)
	if spec == "" {
		return nil, fmt.Errorf("cron schedule is required")
	}

	loc := time.UTC
	if strings.TrimSpace(s.Timezone) != "" {
		l, err := time.LoadLocation(s.Timezone)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", s.Timezone, err)
		}
		loc = l
	}

	fields := strings.Fields(spec)
	switch len(fields) {
	case 5:
		minutes, err := parseCronField(fields[0], 0, 59, false)
		if err != nil {
			return nil, fmt.Errorf("minute field: %w", err)
		}
		hours, err := parseCronField(fields[1], 0, 23, false)
		if err != nil {
			return nil, fmt.Errorf("hour field: %w", err)
		}
		dom, err := parseCronField(fields[2], 1, 31, false)
		if err != nil {
			return nil, fmt.Errorf("day-of-month field: %w", err)
		}
		months, err := parseCronField(fields[3], 1, 12, false)
		if err != nil {
			return nil, fmt.Errorf("month field: %w", err)
		}
		dow, err := parseCronField(fields[4], 0, 7, true)
		if err != nil {
			return nil, fmt.Errorf("day-of-week field: %w", err)
		}
		return &CronSchedule{
			spec:       spec,
			timezone:   loc,
			hasSeconds: false,
			minutes:    minutes,
			hours:      hours,
			dom:        dom,
			months:     months,
			dow:        dow,
		}, nil

	case 6:
		seconds, err := parseCronField(fields[0], 0, 59, false)
		if err != nil {
			return nil, fmt.Errorf("second field: %w", err)
		}
		minutes, err := parseCronField(fields[1], 0, 59, false)
		if err != nil {
			return nil, fmt.Errorf("minute field: %w", err)
		}
		hours, err := parseCronField(fields[2], 0, 23, false)
		if err != nil {
			return nil, fmt.Errorf("hour field: %w", err)
		}
		dom, err := parseCronField(fields[3], 1, 31, false)
		if err != nil {
			return nil, fmt.Errorf("day-of-month field: %w", err)
		}
		months, err := parseCronField(fields[4], 1, 12, false)
		if err != nil {
			return nil, fmt.Errorf("month field: %w", err)
		}
		dow, err := parseCronField(fields[5], 0, 7, true)
		if err != nil {
			return nil, fmt.Errorf("day-of-week field: %w", err)
		}
		return &CronSchedule{
			spec:       spec,
			timezone:   loc,
			hasSeconds: true,
			seconds:    seconds,
			minutes:    minutes,
			hours:      hours,
			dom:        dom,
			months:     months,
			dow:        dow,
		}, nil

	default:
		return nil, fmt.Errorf("cron expression must have 5 or 6 fields, got %d", len(fields))
	}
}

func (c *CronSchedule) Next(after time.Time) time.Time {
	if c == nil {
		return time.Time{}
	}

	start := after.In(c.timezone)
	if c.hasSeconds {
		start = start.Add(time.Second).Truncate(time.Second)
		return c.nextBySecond(start)
	}

	start = start.Add(time.Minute).Truncate(time.Minute)
	return c.nextByMinute(start)
}

func (c *CronSchedule) nextByMinute(start time.Time) time.Time {
	limit := start.AddDate(1, 0, 0)

	for t := start; t.Before(limit); t = t.Add(time.Minute) {
		if c.matchesMinute(t) {
			return t
		}
	}
	return time.Time{}
}

func (c *CronSchedule) nextBySecond(start time.Time) time.Time {
	limit := start.AddDate(1, 0, 0)

	for t := start; t.Before(limit); t = t.Add(time.Second) {
		if c.matchesSecond(t) {
			return t
		}
	}
	return time.Time{}
}

func (c *CronSchedule) matchesMinute(t time.Time) bool {
	if !c.months.contains(int(t.Month())) {
		return false
	}
	if !c.hours.contains(t.Hour()) {
		return false
	}
	if !c.minutes.contains(t.Minute()) {
		return false
	}
	if !matchesDay(c.dom, c.dow, t) {
		return false
	}
	return true
}

func (c *CronSchedule) matchesSecond(t time.Time) bool {
	if !c.months.contains(int(t.Month())) {
		return false
	}
	if !c.hours.contains(t.Hour()) {
		return false
	}
	if !c.minutes.contains(t.Minute()) {
		return false
	}
	if !c.seconds.contains(t.Second()) {
		return false
	}
	if !matchesDay(c.dom, c.dow, t) {
		return false
	}
	return true
}

func matchesDay(dom, dow *cronField, t time.Time) bool {
	domWildcard := dom.any
	dowWildcard := dow.any

	domMatch := dom.contains(t.Day())
	dowVal := int(t.Weekday())
	dowMatch := dow.contains(dowVal)
	if dowVal == 0 {
		dowMatch = dowMatch || dow.contains(7)
	}

	switch {
	case domWildcard && dowWildcard:
		return true
	case domWildcard:
		return dowMatch
	case dowWildcard:
		return domMatch
	default:
		return domMatch || dowMatch
	}
}

type cronField struct {
	any    bool
	values map[int]struct{}
	min    int
	max    int
}

func parseCronField(expr string, min, max int, allowSundaySeven bool) (*cronField, error) {
	expr = strings.TrimSpace(expr)
	if expr == "*" {
		return &cronField{any: true, min: min, max: max}, nil
	}

	f := &cronField{
		values: make(map[int]struct{}),
		min:    min,
		max:    max,
	}

	parts := strings.Split(expr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty cron field segment")
		}

		if strings.HasPrefix(part, "*/") {
			step, err := strconv.Atoi(strings.TrimPrefix(part, "*/"))
			if err != nil || step <= 0 {
				return nil, fmt.Errorf("invalid step %q", part)
			}
			for i := min; i <= max; i += step {
				if err := f.add(i, allowSundaySeven); err != nil {
					return nil, err
				}
			}
			continue
		}

		if strings.Contains(part, "-") {
			rng := strings.Split(part, "-")
			if len(rng) != 2 {
				return nil, fmt.Errorf("invalid range %q", part)
			}
			start, err := strconv.Atoi(rng[0])
			if err != nil {
				return nil, fmt.Errorf("invalid range start %q", rng[0])
			}
			end, err := strconv.Atoi(rng[1])
			if err != nil {
				return nil, fmt.Errorf("invalid range end %q", rng[1])
			}
			if start > end {
				return nil, fmt.Errorf("invalid range %q", part)
			}
			for i := start; i <= end; i++ {
				if err := f.add(i, allowSundaySeven); err != nil {
					return nil, err
				}
			}
			continue
		}

		v, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid value %q", part)
		}
		if err := f.add(v, allowSundaySeven); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (f *cronField) add(v int, allowSundaySeven bool) error {
	if allowSundaySeven && v == 7 {
		v = 0
	}
	if v < f.min || v > f.max {
		return fmt.Errorf("value %d out of range [%d,%d]", v, f.min, f.max)
	}
	f.values[v] = struct{}{}
	return nil
}

func (f *cronField) contains(v int) bool {
	if f == nil {
		return false
	}
	if f.any {
		return true
	}
	_, ok := f.values[v]
	return ok
}