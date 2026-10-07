package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// HumanizeSchedule translates standard cron expression into a human-readable description
func HumanizeSchedule(schedule string) string {
	s := strings.TrimSpace(schedule)
	switch strings.ToLower(s) {
	case "@reboot":
		return "При загрузке системы (At system boot)"
	case "@yearly", "@annually":
		return "Ежегодно 1 января в 00:00 (Yearly on Jan 1)"
	case "@monthly":
		return "Ежемесячно 1-го числа в 00:00 (Monthly on 1st)"
	case "@weekly":
		return "Еженедельно по воскресеньям в 00:00 (Weekly on Sun)"
	case "@daily", "@midnight":
		return "Ежедневно в 00:00 (Daily at midnight)"
	case "@hourly":
		return "Каждый час (Hourly at minute 0)"
	}

	parts := strings.Fields(s)
	if len(parts) != 5 {
		return s
	}

	min, hour, dom, mon, dow := parts[0], parts[1], parts[2], parts[3], parts[4]

	// 1. Every minute
	if min == "*" && hour == "*" && dom == "*" && mon == "*" && dow == "*" {
		return "Каждую минуту (Every minute)"
	}

	// 2. Every N minutes: */N * * * *
	if strings.HasPrefix(min, "*/") && hour == "*" && dom == "*" && mon == "*" && dow == "*" {
		n := strings.TrimPrefix(min, "*/")
		return fmt.Sprintf("Каждые %s мин. (Every %s min)", n, n)
	}

	// 3. Hourly at minute M: M * * * *
	if isNum(min) && hour == "*" && dom == "*" && mon == "*" && dow == "*" {
		return fmt.Sprintf("Каждый час в %s мин. (Hourly at minute %s)", min, min)
	}

	// 4. Every N hours at minute 0: 0 */N * * *
	if isNum(min) && strings.HasPrefix(hour, "*/") && dom == "*" && mon == "*" && dow == "*" {
		n := strings.TrimPrefix(hour, "*/")
		return fmt.Sprintf("Каждые %s ч. в %02s мин. (Every %s hours)", n, min, n)
	}

	// 5. Hour ranges: M H1-H2 * * *
	if isNum(min) && strings.Contains(hour, "-") && dom == "*" && mon == "*" && dow == "*" {
		return fmt.Sprintf("Каждый час с %s:00 до %s:59 в %02s мин.", hour, hour, min)
	}

	// 6. Daily at specific time: M H * * *
	if isNum(min) && isNum(hour) && dom == "*" && mon == "*" && dow == "*" {
		mVal, _ := strconv.Atoi(min)
		hVal, _ := strconv.Atoi(hour)
		return fmt.Sprintf("Ежедневно в %02d:%02d (Daily at %02d:%02d)", hVal, mVal, hVal, mVal)
	}

	// 7. Weekdays at specific time: M H * * 1-5
	if isNum(min) && isNum(hour) && dom == "*" && mon == "*" && (dow == "1-5" || dow == "mon-fri") {
		mVal, _ := strconv.Atoi(min)
		hVal, _ := strconv.Atoi(hour)
		return fmt.Sprintf("По будням в %02d:%02d (Weekdays at %02d:%02d)", hVal, mVal, hVal, mVal)
	}

	// 8. Weekends at specific time: M H * * 6,0 or 6,7
	if isNum(min) && isNum(hour) && dom == "*" && mon == "*" && (dow == "6,0" || dow == "0,6" || dow == "6,7" || dow == "sat,sun") {
		mVal, _ := strconv.Atoi(min)
		hVal, _ := strconv.Atoi(hour)
		return fmt.Sprintf("По выходным в %02d:%02d (Weekends at %02d:%02d)", hVal, mVal, hVal, mVal)
	}

	// 9. Specific day of week: M H * * DOW
	if isNum(min) && isNum(hour) && dom == "*" && mon == "*" && isNum(dow) {
		mVal, _ := strconv.Atoi(min)
		hVal, _ := strconv.Atoi(hour)
		dNamesRU := []string{"воскресеньям", "понедельникам", "вторникам", "средам", "четвергам", "пятницам", "субботам", "воскресеньям"}
		dNamesEN := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
		dVal, _ := strconv.Atoi(dow)
		if dVal >= 0 && dVal < len(dNamesRU) {
			return fmt.Sprintf("По %s в %02d:%02d (Weekly on %s)", dNamesRU[dVal], hVal, mVal, dNamesEN[dVal])
		}
	}

	// 10. Specific day of month: M H DOM * *
	if isNum(min) && isNum(hour) && isNum(dom) && mon == "*" && dow == "*" {
		mVal, _ := strconv.Atoi(min)
		hVal, _ := strconv.Atoi(hour)
		return fmt.Sprintf("Каждое %s-е число в %02d:%02d (Monthly on day %s)", dom, hVal, mVal, dom)
	}

	// Fallback general description
	return fmt.Sprintf("Cron: %s (Мин:%s Ч:%s Дн:%s Мес:%s ДнНед:%s)", s, min, hour, dom, mon, dow)
}

func isNum(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.Atoi(s)
	return err == nil
}

// NextRun calculates the next occurrence after 'from' for a cron schedule
func NextRun(schedule string, from time.Time) time.Time {
	s := strings.TrimSpace(schedule)
	if strings.EqualFold(s, "@reboot") {
		return time.Time{} // Reboot has no deterministic clock schedule
	}

	// Convert macro to 5-part cron
	switch strings.ToLower(s) {
	case "@yearly", "@annually":
		s = "0 0 1 1 *"
	case "@monthly":
		s = "0 0 1 * *"
	case "@weekly":
		s = "0 0 * * 0"
	case "@daily", "@midnight":
		s = "0 0 * * *"
	case "@hourly":
		s = "0 * * * *"
	}

	parts := strings.Fields(s)
	if len(parts) != 5 {
		return time.Time{}
	}

	minMatcher, err1 := parseCronField(parts[0], 0, 59)
	hourMatcher, err2 := parseCronField(parts[1], 0, 23)
	domMatcher, err3 := parseCronField(parts[2], 1, 31)
	monMatcher, err4 := parseCronFieldNamed(parts[3], 1, 12, monthNames)
	dowMatcher, err5 := parseCronFieldNamed(parts[4], 0, 7, dowNames) // 0 and 7 = Sun

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
		return time.Time{}
	}

	// Advance starting point to next minute
	t := from.Truncate(time.Minute).Add(time.Minute)

	// Search up to 5 years forward (or ~2.6 million minutes)
	limit := from.AddDate(5, 0, 0)
	for t.Before(limit) {
		// Month check
		if !monMatcher[int(t.Month())] {
			// Advance to beginning of next month
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
			continue
		}

		// Day check (Day of Month & Day of Week)
		domMatch := domMatcher[t.Day()]
		dow := int(t.Weekday())
		dowMatch := dowMatcher[dow] || (dow == 0 && dowMatcher[7])

		// Vixie cron semantics: if either DOM or DOW starts with "*" both must match,
		// otherwise (both restricted) matching either one is enough.
		isDomStar := strings.HasPrefix(parts[2], "*")
		isDowStar := strings.HasPrefix(parts[4], "*")
		var dayMatches bool
		if isDomStar || isDowStar {
			dayMatches = domMatch && dowMatch
		} else {
			dayMatches = domMatch || dowMatch
		}

		if !dayMatches {
			// Advance to next day at 00:00
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			continue
		}

		// Hour check
		if !hourMatcher[t.Hour()] {
			// Advance to start of next hour
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
			continue
		}

		// Minute check
		if !minMatcher[t.Minute()] {
			t = t.Add(time.Minute)
			continue
		}

		// All match!
		return t
	}

	return time.Time{}
}

var (
	monthNames = map[string]int{
		"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
		"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
	}
	dowNames = map[string]int{
		"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6,
	}
)

// parseCronField parses numeric cron field expressions like "*", "*/5", "1,2,3", "10-20", "9-17/2"
func parseCronField(expr string, min, max int) (map[int]bool, error) {
	return parseCronFieldNamed(expr, min, max, nil)
}

// parseCronFieldNamed parses a cron field, additionally accepting symbolic names
// (e.g. "jan", "mon-fri") from the given names table.
func parseCronFieldNamed(expr string, min, max int, names map[string]int) (map[int]bool, error) {
	parseVal := func(v string) (int, error) {
		if n, ok := names[strings.ToLower(v)]; ok {
			return n, nil
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", v)
		}
		return n, nil
	}

	res := make(map[int]bool)
	for _, part := range strings.Split(expr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty element in %q", expr)
		}

		// Optional step suffix: "*/5", "10-20/2"
		step := 1
		if base, stepStr, hasStep := strings.Cut(part, "/"); hasStep {
			n, err := strconv.Atoi(stepStr)
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("invalid step: %s", part)
			}
			step = n
			part = base
		}

		start, end := min, max
		switch {
		case part == "*":
			// full range
		case strings.Contains(part, "-"):
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			s, err1 := parseVal(rangeParts[0])
			e, err2 := parseVal(rangeParts[1])
			if err1 != nil || err2 != nil || s > e {
				return nil, fmt.Errorf("invalid range numbers: %s", part)
			}
			start, end = s, e
		default:
			v, err := parseVal(part)
			if err != nil {
				return nil, err
			}
			start = v
			if step == 1 {
				end = v
			}
		}

		for i := start; i <= end; i += step {
			if i >= min && i <= max {
				res[i] = true
			}
		}
	}

	return res, nil
}
