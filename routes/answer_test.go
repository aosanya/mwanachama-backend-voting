package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aosanya/mwanachama-backend-voting/routes"
)

func newAnsweringServer(t *testing.T, caller string, groups []string) *httptest.Server {
	t.Helper()
	pm := newManager(t)
	pm.UseGroups(func(context.Context, string) ([]string, error) { return groups, nil })
	mux := http.NewServeMux()
	for _, rt := range routes.RoutesFor(pm, routes.Mount{Caller: callerFrom}) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), callerKey{}, caller)))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func anOpenPoll(t *testing.T, srv *httptest.Server) (string, string) {
	t.Helper()
	body := aPoll("")
	body["opens_at"], body["closes_at"] = "2020-01-01T00:00:00Z", "2099-01-01T00:00:00Z"
	body["choices"] = []map[string]any{{"label": "Monday"}, {"label": "Tuesday"}}
	code, poll := post(t, srv, "/polls", body)
	if code != http.StatusCreated {
		t.Fatalf("create poll = %d %v", code, poll)
	}
	first := poll["choices"].([]any)[0].(map[string]any)["id"].(string)
	return poll["id"].(string), first
}

func TestTheAnswererComesFromTheCallerNotTheBody(t *testing.T) {
	srv := newAnsweringServer(t, "actor-1", []string{"g-1"})
	pollID, choiceID := anOpenPoll(t, srv)
	code, part := post(t, srv, "/polls/"+pollID+"/answers", map[string]any{
		"answered_by": "someone-else",
		"poll_id":     "elsewhere",
		"votes":       map[string]int{choiceID: 1},
	})
	if code != http.StatusCreated || part["answered_by"] != "actor-1" || part["poll_id"] != pollID {
		t.Fatalf("answer = %d %v", code, part)
	}
	if _, has := part["votes"]; has {
		t.Fatalf("the answer carries what was chosen: %v", part)
	}
	code, _ = post(t, srv, "/polls/"+pollID+"/answers", map[string]any{"votes": map[string]int{choiceID: 1}})
	if code != http.StatusConflict {
		t.Fatalf("second answer status = %d, want 409", code)
	}
}

func TestACallerOutsideThePollsGroupsAnswersNotFound(t *testing.T) {
	srv := newAnsweringServer(t, "actor-1", []string{"g-9"})
	pollID, choiceID := anOpenPoll(t, srv)
	code, _ := post(t, srv, "/polls/"+pollID+"/answers", map[string]any{"votes": map[string]int{choiceID: 1}})
	if code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", code)
	}
}

func TestAnAnswerWithNoCallerIsUnauthorized(t *testing.T) {
	srv := newAnsweringServer(t, "", []string{"g-1"})
	code, _ := post(t, srv, "/polls/any/answers", map[string]any{"votes": map[string]int{"c": 1}})
	if code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", code)
	}
}
