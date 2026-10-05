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
	monMatcher, err4 := parseCronField(parts[3], 1, 12)
	dowMatcher, err5 := parseCronField(parts[4], 0, 7) // 0 and 7 = Sun

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

		// In standard cron, if both DOM and DOW are specified (not *), it is an OR condition
		dayMatches := false
		isDomStar := parts[2] == "*"
		isDowStar := parts[4] == "*"

		if isDomStar && isDowStar {
			dayMatches = true
		} else if isDomStar {
			dayMatches = dowMatch
		} else if isDowStar {
			dayMatches = domMatch
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

// parseCronField parses cron field expressions like "*", "*/5", "1,2,3", "10-20"
func parseCronField(expr string, min, max int) (map[int]bool, error) {
	res := make(map[int]bool)
	for _, part := range strings.Split(expr, ",") {
		part = strings.TrimSpace(part)
		if part == "*" {
			for i := min; i <= max; i++ {
				res[i] = true
			}
			continue
		}

		if strings.HasPrefix(part, "*/") {
			stepStr := strings.TrimPrefix(part, "*/")
			step, err := strconv.Atoi(stepStr)
			if err != nil || step <= 0 {
				return nil, fmt.Errorf("invalid step: %s", part)
			}
			for i := min; i <= max; i += step {
				res[i] = true
			}
			continue
		}

		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])
			if err1 != nil || err2 != nil || start > end {
				return nil, fmt.Errorf("invalid range numbers: %s", part)
			}
			for i := start; i <= end; i++ {
				if i >= min && i <= max {
					res[i] = true
				}
			}
			continue
		}

		val, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid number: %s", part)
		}
		if val >= min && val <= max {
			res[val] = true
		}
	}

	return res, nil
}
