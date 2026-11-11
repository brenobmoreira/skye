package notes

import "testing"

func TestNext(t *testing.T) {
	cases := []struct {
		r     Repeat
		after string
		want  string
	}{
		{Repeat{Kind: "daily"}, "2026-11-11", "2026-11-12"},
		{Repeat{Kind: "daily"}, "2026-12-31", "2027-01-01"},
		{Repeat{Kind: "weekdays"}, "2026-11-13", "2026-11-16"},
		{Repeat{Kind: "weekdays"}, "2026-11-11", "2026-11-12"},
		{Repeat{Kind: "weekly", Weekday: 5}, "2026-11-11", "2026-11-13"},
		{Repeat{Kind: "weekly", Weekday: 5}, "2026-11-13", "2026-11-20"},
		{Repeat{Kind: "monthly", Day: 31}, "2026-11-11", "2026-11-30"},
		{Repeat{Kind: "monthly", Day: 31}, "2026-11-30", "2026-12-31"},
		{Repeat{Kind: "monthly", Day: 31}, "2027-01-31", "2027-02-28"},
		{Repeat{Kind: "monthly", Day: 29}, "2028-01-31", "2028-02-29"},
		{Repeat{Kind: "monthly", Day: 10}, "2026-11-11", "2026-12-10"},
	}
	for _, c := range cases {
		got, err := Next(c.r, c.after)
		if err != nil {
			t.Fatalf("Next(%+v, %s): %v", c.r, c.after, err)
		}
		if got != c.want {
			t.Errorf("Next(%+v, %s) = %s, want %s", c.r, c.after, got, c.want)
		}
	}
	if _, err := Next(Repeat{Kind: "daily"}, "11/11"); err == nil {
		t.Fatal("invalid date accepted")
	}
}

func TestRepeatValidate(t *testing.T) {
	bad := []Repeat{
		{Kind: "yearly"},
		{Kind: "weekly", Weekday: 7},
		{Kind: "weekly", Weekday: -1},
		{Kind: "monthly", Day: 0},
		{Kind: "monthly", Day: 32},
	}
	for _, r := range bad {
		if r.Validate() == nil {
			t.Errorf("%+v accepted", r)
		}
	}
	good := []Repeat{{Kind: "daily"}, {Kind: "weekdays"}, {Kind: "weekly", Weekday: 0}, {Kind: "monthly", Day: 31}}
	for _, r := range good {
		if err := r.Validate(); err != nil {
			t.Errorf("%+v: %v", r, err)
		}
	}
}
