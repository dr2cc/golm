package report

import "testing"

func TestTrackerSequence(t *testing.T) {
	var got []Step
	tr := NewTracker(func(s Step) { got = append(got, s) },
		Step{ID: "a", Title: "A"}, Step{ID: "b", Title: "B"}, Step{ID: "c", Title: "C"})

	tr.Start("a")
	tr.Done("a", Fail, "boom")
	tr.SkipRest("prev failed")

	want := []Step{
		{"a", "A", Pending, ""}, {"b", "B", Pending, ""}, {"c", "C", Pending, ""},
		{"a", "A", Running, ""}, {"a", "A", Fail, "boom"},
		{"b", "B", Skipped, "prev failed"}, {"c", "C", Skipped, "prev failed"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// В CLI получатель не задан — Tracker не должен паниковать.
func TestTrackerNilEmit(t *testing.T) {
	tr := NewTracker(nil, Step{ID: "a"})
	tr.Start("a")
	tr.Done("a", OK, "")
	tr.SkipRest("")
}
