package routes_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	voting "github.com/aosanya/mwanachama-backend-voting"
	"github.com/aosanya/mwanachama-backend-voting/routes"
)

func serve(t *testing.T, mount routes.Mount, caller string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for _, rt := range routes.RoutesFor(newManager(t), mount) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), callerKey{}, caller)))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestBuildReturnsEveryDeclaredAddressWithAnAction(t *testing.T) {
	built, err := routes.Build(newManager(t))
	if err != nil {
		t.Fatal(err)
	}
	shape, err := routes.Table.Shape()
	if err != nil {
		t.Fatal(err)
	}
	if len(built) == 0 || len(built) != len(shape) {
		t.Fatalf("built %d routes, declared %d", len(built), len(shape))
	}
	seen := map[string]bool{}
	for _, rt := range built {
		if !strings.HasPrefix(rt.Action, "voting.") {
			t.Fatalf("route %q carries action %q, want a voting.* action", rt.Pattern(""), rt.Action)
		}
		if seen[rt.Action] {
			t.Fatalf("action %q is bound to two routes", rt.Action)
		}
		seen[rt.Action] = true
	}
}

func TestBuildForAnswersTheSameAddressesAsBuild(t *testing.T) {
	pm := newManager(t)
	plain, err := routes.Build(pm)
	if err != nil {
		t.Fatal(err)
	}
	withCaller, err := routes.BuildFor(pm, routes.Mount{Caller: callerFrom})
	if err != nil {
		t.Fatal(err)
	}
	if len(plain) != len(withCaller) {
		t.Fatalf("Build = %d routes, BuildFor = %d", len(plain), len(withCaller))
	}
	for i := range plain {
		if plain[i].Pattern("") != withCaller[i].Pattern("") {
			t.Fatalf("route %d: %q != %q", i, plain[i].Pattern(""), withCaller[i].Pattern(""))
		}
	}
}

func TestAnAuthorizerRefusalAnswersForbiddenAndNothingIsCreated(t *testing.T) {
	var mu sync.Mutex
	var asked []string
	deny := func(_ context.Context, action string) error {
		mu.Lock()
		defer mu.Unlock()
		asked = append(asked, action)
		return errors.New("no")
	}
	srv := serve(t, routes.Mount{Caller: callerFrom, Authorize: deny}, "actor-1")
	code, _ := post(t, srv, "/polls", aPoll(""))
	if code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", code)
	}
	if len(asked) != 1 || asked[0] != "voting.poll.create" {
		t.Fatalf("authorizer was asked %v, want [voting.poll.create]", asked)
	}
	res, err := http.Get(srv.URL + "/polls/any")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("read status = %d, want 403 before the poll is even looked up", res.StatusCode)
	}
}

func TestAnAuthorizerThatGrantsOnlyReadsLetsReadsThroughAndRefusesWrites(t *testing.T) {
	readOnly := func(_ context.Context, action string) error {
		if strings.HasSuffix(action, ".read") {
			return nil
		}
		return errors.New("no")
	}
	srv := serve(t, routes.Mount{Caller: callerFrom, Authorize: readOnly}, "actor-1")
	code, _ := post(t, srv, "/polls", aPoll(""))
	if code != http.StatusForbidden {
		t.Fatalf("create status = %d, want 403", code)
	}
	res, err := http.Get(srv.URL + "/polls/nope")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("read status = %d, want 404 from the poll lookup", res.StatusCode)
	}
}

func TestAPollWithNoQuestionIsRefusedAsInvalid(t *testing.T) {
	srv := newServer(t, "actor-1")
	body := aPoll("")
	body["question"] = ""
	code, _ := post(t, srv, "/polls", body)
	if code != http.StatusBadRequest && code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 400 or 422", code)
	}
}

func TestEverySentinelIsAKnownVotingError(t *testing.T) {
	known := []error{
		voting.ErrPollNotFound, voting.ErrInvalidPoll, voting.ErrInvalidChoice, voting.ErrNoCreator,
		voting.ErrPollNotDraft, voting.ErrNoAnswerer, voting.ErrOutsideWindow, voting.ErrNotInPollGroups,
		voting.ErrInvalidAnswer, voting.ErrAlreadyAnswered, voting.ErrNoPublisher,
		voting.ErrInvalidPublication, voting.ErrNoReader, voting.ErrAlreadyPublished,
	}
	if len(routes.Sentinels) != len(known) {
		t.Fatalf("%d sentinels mapped, %d errors declared", len(routes.Sentinels), len(known))
	}
	for name, err := range routes.Sentinels {
		found := false
		for _, k := range known {
			if errors.Is(err, k) {
				found = true
			}
		}
		if !found || !strings.HasPrefix(name, "Err") {
			t.Fatalf("sentinel %q is not one of the module's errors", name)
		}
	}
}
