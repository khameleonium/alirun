package cron

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestFindJobExactID(t *testing.T) {
	var sb strings.Builder
	for i := 1; i <= 12; i++ {
		fmt.Fprintf(&sb, "0 %d * * * /bin/echo job%d\n", i, i)
	}
	jobs, err := ParseCrontab(sb.String(), "crontab (user)", false)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		"cron-12":                   "/bin/echo job12",
		"[cron-12] /bin/echo job12": "/bin/echo job12",
		"CRON-11":                   "/bin/echo job11",
		"cron-1":                    "/bin/echo job1",
		"job10":                     "/bin/echo job10",
		"[cron-2] /bin/echo job2":   "/bin/echo job2",
	}
	for target, want := range cases {
		j := findJob(jobs, target)
		if j == nil || j.Command != want {
			t.Errorf("findJob(%q) = %v, want %q", target, j, want)
		}
	}
	if j := findJob(jobs, "cron-99"); j != nil {
		t.Errorf("findJob(cron-99) must not match anything, got %q", j.Command)
	}
}

func TestParseCronFieldRangeStepAndNames(t *testing.T) {
	got, err := parseCronField("9-17/2", 0, 23)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range []int{9, 11, 13, 15, 17} {
		if !got[h] {
			t.Errorf("hour %d expected in 9-17/2", h)
		}
	}
	if got[10] || len(got) != 5 {
		t.Errorf("unexpected set for 9-17/2: %v", got)
	}

	dows, err := parseCronFieldNamed("MON-FRI", 0, 7, dowNames)
	if err != nil || len(dows) != 5 || !dows[1] || !dows[5] {
		t.Errorf("MON-FRI parsed wrong: %v %v", dows, err)
	}

	// Jobs with range+step must be recognised (they used to vanish from the list)
	jobs, _ := ParseCrontab("0 9-17/2 * * * /bin/echo x\n", "crontab (user)", false)
	if len(jobs) != 1 {
		t.Fatalf("range-step job not parsed: %v", jobs)
	}

	from := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC) // Wednesday
	next := NextRun("0 9 * * mon-fri", from)
	if want := time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC); !next.Equal(want) {
		t.Errorf("NextRun mon-fri = %v, want %v", next, want)
	}
}
