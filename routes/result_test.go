package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, srv *httptest.Server, path string) (int, string) {
	t.Helper()
	res, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	raw, _ := json.Marshal(out)
	return res.StatusCode, string(raw)
}

func TestAMemberReadsTheCountOnlyOncePublishedToTheirGroup(t *testing.T) {
	srv := newAnsweringServer(t, "actor-1", []string{"g-1"})
	pollID, choiceID := anOpenPoll(t, srv)
	if code, out := post(t, srv, "/polls/"+pollID+"/answers", map[string]any{"votes": map[string]int{choiceID: 1}}); code != http.StatusCreated {
		t.Fatalf("answer = %d %v", code, out)
	}
	code, body := get(t, srv, "/polls/"+pollID+"/count")
	if code != http.StatusOK || !strings.Contains(body, `"answered":1`) || strings.Contains(body, "actor-1") {
		t.Fatalf("holders' count = %d %s", code, body)
	}
	if code, body := get(t, srv, "/polls/"+pollID+"/result"); code != http.StatusNotFound {
		t.Fatalf("unpublished result = %d %s", code, body)
	}
	if code, out := post(t, srv, "/polls/"+pollID+"/result", map[string]any{"group_ids": []string{"g-2"}, "published_by": "someone-else"}); code != http.StatusOK || out["published_by"] != "actor-1" {
		t.Fatalf("publish = %d %v", code, out)
	}
	if code, body := get(t, srv, "/polls/"+pollID+"/result"); code != http.StatusNotFound {
		t.Fatalf("result published elsewhere = %d %s", code, body)
	}
	post(t, srv, "/polls/"+pollID+"/result", map[string]any{"group_ids": []string{"g-1"}})
	code, body = get(t, srv, "/polls/"+pollID+"/result")
	if code != http.StatusOK || !strings.Contains(body, `"votes":1`) || strings.Contains(body, "actor-1") {
		t.Fatalf("published result = %d %s", code, body)
	}
}

func TestAResultIsPublishedToAtLeastOneGroup(t *testing.T) {
	srv := newAnsweringServer(t, "actor-1", []string{"g-1"})
	pollID, _ := anOpenPoll(t, srv)
	if code, out := post(t, srv, "/polls/"+pollID+"/result", map[string]any{"group_ids": []string{}}); code != http.StatusBadRequest {
		t.Fatalf("publish to none = %d %v", code, out)
	}
}
