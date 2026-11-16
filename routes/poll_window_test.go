package routes

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func windowBody(opens, closes string) string {
	return fmt.Sprintf(`{"question":"Which street?","mode":"one","per_group":"once","group_ids":["ward-a"],"opens_at":%q,"closes_at":%q}`, opens, closes)
}

func TestAPollWindowThatIsNotARealTimestampAnswers400NamingTheField(t *testing.T) {
	l := newLiveVoting(t, "clerk", nil)
	for _, bad := range []string{"2026-02-30T00:00:00Z", "2026-11-08T24:00:00Z", "2016-12-31T23:59:60Z", "2026-11-08", "2026-11-08T00:00:00", "2026-11-08t00:00:00z", " 2026-11-08T00:00:00Z", "2026-11-08T00:00:00Z "} {
		for field, body := range map[string]string{
			"opens_at":  windowBody(bad, "2026-11-09T00:00:00Z"),
			"closes_at": windowBody("2026-11-01T00:00:00Z", bad),
		} {
			rec := l.call("POST", "/polls", body)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), field) {
				t.Errorf("%s %q answered %d %s, want 400 naming %s", field, bad, rec.Code, rec.Body.String(), field)
			}
		}
	}
}

func TestAPollWindowClosingBeforeItOpensByInstantAnswers400(t *testing.T) {
	l := newLiveVoting(t, "clerk", nil)
	rec := l.call("POST", "/polls", windowBody("2026-11-08T00:00:00Z", "2026-11-08T12:00:00+14:00"))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "closes_at falls after opens_at") {
		t.Fatalf("answered %d %s, want 400 closes_at falls after opens_at", rec.Code, rec.Body.String())
	}
}

func TestAPollWindowOfTenRealHoursAnswers201AndComesBackInUTC(t *testing.T) {
	l := newLiveVoting(t, "clerk", nil)
	rec := l.call("POST", "/polls", windowBody("2026-11-08T05:00:00+14:00", "2026-11-08T00:00:00Z"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("answered %d %s, want 201", rec.Code, rec.Body.String())
	}
	got := decodeObject(t, rec)
	if got["opens_at"] != "2026-11-07T15:00:00Z" || got["closes_at"] != "2026-11-08T00:00:00Z" {
		t.Fatalf("window %v to %v", got["opens_at"], got["closes_at"])
	}
}

func TestAPollWindowWhoseUTCYearIsOutOfRangeAnswers400NamingTheField(t *testing.T) {
	l := newLiveVoting(t, "clerk", nil)
	cases := map[string]struct{ body, field string }{
		"closes_at in year 10000": {windowBody("2026-11-08T00:00:00Z", "9999-12-31T23:00:00-02:00"), "closes_at"},
		"opens_at in year -1":     {windowBody("0000-01-01T00:00:00+02:00", "2026-11-09T00:00:00Z"), "opens_at"},
	}
	for name, c := range cases {
		rec := l.call("POST", "/polls", c.body)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid poll: "+c.field+" falls outside the years 0000 to 9999 in UTC") {
			t.Errorf("%s answered %d %s, want 400 naming %s", name, rec.Code, rec.Body.String(), c.field)
		}
	}
}

func TestAPollWindowEndingOnTheLastSecondOfYear9999Answers201(t *testing.T) {
	l := newLiveVoting(t, "clerk", nil)
	rec := l.call("POST", "/polls", windowBody("2026-11-08T00:00:00Z", "9999-12-31T23:59:59Z"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("answered %d %s, want 201", rec.Code, rec.Body.String())
	}
}
