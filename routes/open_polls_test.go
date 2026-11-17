package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func getList(t *testing.T, srv *httptest.Server, path string) (int, []map[string]any, string) {
	t.Helper()
	res, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var raw json.RawMessage
	_ = json.NewDecoder(res.Body).Decode(&raw)
	var out []map[string]any
	_ = json.Unmarshal(raw, &out)
	return res.StatusCode, out, string(raw)
}

func TestTheOpenPollsRouteListsThePollsOfTheCallersGroupsWithTheirChoices(t *testing.T) {
	srv := newAnsweringServer(t, "actor-1", []string{"g-1"})
	pollID, _ := anOpenPoll(t, srv)
	code, list, raw := getList(t, srv, "/polls/open")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", code, raw)
	}
	if len(list) != 1 || list[0]["id"] != pollID {
		t.Fatalf("list = %s", raw)
	}
	if choices, _ := list[0]["choices"].([]any); len(choices) != 2 {
		t.Fatalf("choices = %v", list[0]["choices"])
	}
	if list[0]["answered"] != false {
		t.Fatalf("answered = %v, want false", list[0]["answered"])
	}
}

func TestTheOpenPollsRouteIsNotTakenForAPollIDAndMarksAnAnsweredPoll(t *testing.T) {
	srv := newAnsweringServer(t, "actor-1", []string{"g-1"})
	pollID, choiceID := anOpenPoll(t, srv)
	if code, out := post(t, srv, "/polls/"+pollID+"/answers", map[string]any{"votes": map[string]int{choiceID: 1}}); code != http.StatusCreated {
		t.Fatalf("answer = %d %v", code, out)
	}
	code, list, raw := getList(t, srv, "/polls/open")
	if code != http.StatusOK || strings.HasPrefix(raw, "{") {
		t.Fatalf("GET /polls/open = %d %s, shadowed by /polls/{pollID}", code, raw)
	}
	if len(list) != 1 || list[0]["answered"] != true {
		t.Fatalf("list = %s", raw)
	}
}

func TestTheOpenPollsRouteAnswersAnEmptyArrayForACallerInNoGroup(t *testing.T) {
	srv := newAnsweringServer(t, "actor-1", nil)
	anOpenPoll(t, srv)
	code, _, raw := getList(t, srv, "/polls/open")
	if code != http.StatusOK || strings.TrimSpace(raw) != "[]" {
		t.Fatalf("GET /polls/open = %d %s, want 200 []", code, raw)
	}
}

func TestTheOpenPollsRouteNamesNeitherWhoMadeAPollNorWhoPublishedIt(t *testing.T) {
	srv := newAnsweringServer(t, "actor-1", []string{"g-1"})
	anOpenPoll(t, srv)
	_, _, raw := getList(t, srv, "/polls/open")
	if strings.Contains(raw, "created_by") || strings.Contains(raw, "published_by") || strings.Contains(raw, "actor-1") {
		t.Fatalf("respondent list names a member: %s", raw)
	}
}

func TestTheOpenPollsRouteWithNoCallerIsUnauthorized(t *testing.T) {
	srv := newAnsweringServer(t, "", []string{"g-1"})
	code, _, raw := getList(t, srv, "/polls/open")
	if code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401: %s", code, raw)
	}
}
