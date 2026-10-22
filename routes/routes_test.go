package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	voting "github.com/aosanya/mwanachama-backend-voting"
	"github.com/aosanya/mwanachama-backend-voting/routes"
)

type callerKey struct{}

func callerFrom(ctx context.Context) string {
	id, _ := ctx.Value(callerKey{}).(string)
	return id
}

func newManager(t *testing.T) voting.PollManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	s, err := voting.SpecFor("routestest")
	if err != nil {
		t.Fatal(err)
	}
	if err := voting.Provision(db, s); err != nil {
		t.Fatal(err)
	}
	pm, err := voting.NewPollManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	return pm
}

func newServer(t *testing.T, caller string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for _, rt := range routes.RoutesFor(newManager(t), routes.Mount{Caller: callerFrom}) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), callerKey{}, caller)))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func post(t *testing.T, srv *httptest.Server, path string, body any) (int, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	res, err := http.Post(srv.URL+path, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestNoRouteIsAnonymous(t *testing.T) {
	split := routes.Split(newManager(t), routes.Mount{})
	if len(split.Anonymous) != 0 {
		t.Fatalf("%d routes answer without a caller; every address is gated", len(split.Anonymous))
	}
	if len(routes.AnonymousActions) != 0 {
		t.Fatalf("AnonymousActions = %v, want none", routes.AnonymousActions)
	}
}

func aPoll(createdBy string) map[string]any {
	return map[string]any{
		"question":   "Which day suits the meeting?",
		"mode":       "one",
		"group_ids":  []string{"g-1"},
		"opens_at":   "2026-11-01T08:00:00Z",
		"closes_at":  "2026-11-08T08:00:00Z",
		"created_by": createdBy,
		"state":      "closed",
	}
}

func TestTheCreatorComesFromTheCallerNotTheBody(t *testing.T) {
	srv := newServer(t, "actor-1")
	code, poll := post(t, srv, "/polls", aPoll("someone-else"))
	if code != http.StatusCreated || poll["created_by"] != "actor-1" || poll["state"] != "draft" {
		t.Fatalf("create poll = %d %v", code, poll)
	}
	code, choice := post(t, srv, "/polls/"+poll["id"].(string)+"/choices", map[string]any{"label": "Monday", "poll_id": "elsewhere"})
	if code != http.StatusCreated || choice["poll_id"] != poll["id"] || choice["position"] != float64(1) {
		t.Fatalf("add choice = %d %v", code, choice)
	}
}

func TestAPollWithNoCallerIsUnauthorized(t *testing.T) {
	code, _ := post(t, newServer(t, ""), "/polls", aPoll(""))
	if code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", code)
	}
}

func TestAnUnknownPollAnswersNotFound(t *testing.T) {
	res, err := http.Get(newServer(t, "actor-1").URL + "/polls/nope")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
}
