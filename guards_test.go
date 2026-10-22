package voting_test

import (
	"path/filepath"
	"testing"

	voting "github.com/aosanya/mwanachama-backend-voting"
	"github.com/aosanya/mwanachama-backend-voting/models"
)

func shippedSpecs(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob("spec/examples/*.voting.json")
	if err != nil || len(paths) < 2 {
		t.Fatalf("want at least two shipped examples, got %v (%v)", paths, err)
	}
	return append([]string{"voting.wakala.json"}, paths...)
}

func TestEveryExampleFitsTheTypes(t *testing.T) {
	for _, p := range shippedSpecs(t) {
		s, err := voting.LoadSpec(p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		db := openDB(t)
		if err := voting.Provision(db, s); err != nil {
			t.Fatalf("%s: Provision: %v", p, err)
		}
		if _, err := voting.NewPollManager(db, s); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
	}
}

func TestShippedExamplesCoexist(t *testing.T) {
	db := openDB(t)
	for _, p := range shippedSpecs(t) {
		s, err := voting.LoadSpec(p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if err := voting.Provision(db, s); err != nil {
			t.Fatalf("%s: Provision into a shared database: %v", p, err)
		}
	}
}

func TestVocabularyMatchesTheBlueprint(t *testing.T) {
	want := map[string][]string{
		"poll.mode":      {models.ModeOne, models.ModeUpTo, models.ModeQuadratic},
		"poll.per_group": {models.PerGroupOnce, models.PerGroupEach},
		"poll.state":     {models.StateDraft, models.StateOpen, models.StateClosed},
	}
	b, err := voting.Blueprint()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, o := range b.Objects {
		for _, f := range o.Fields {
			if len(f.Values) == 0 {
				continue
			}
			key := o.Role + "." + f.Name
			seen[key] = true
			consts, ok := want[key]
			if !ok {
				t.Errorf("%s declares values with no Go constants beside them", key)
				continue
			}
			if len(consts) != len(f.Values) {
				t.Errorf("%s: blueprint %v, constants %v", key, f.Values, consts)
				continue
			}
			for i := range consts {
				if consts[i] != f.Values[i] {
					t.Errorf("%s: blueprint %v, constants %v", key, f.Values, consts)
				}
			}
		}
	}
	for key := range want {
		if !seen[key] {
			t.Errorf("constants for %s match no declared enum", key)
		}
	}
}

func TestRequiredFieldsHaveNoDefault(t *testing.T) {
	for _, p := range shippedSpecs(t) {
		s, err := voting.LoadSpec(p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		for _, o := range s.Objects {
			for _, f := range o.Fields {
				if f.Required && f.Default != "" {
					t.Errorf("%s: %s.%s is required and defaulted", p, o.Role, f.Name)
				}
			}
		}
	}
}

func TestWhoAnsweredIsKeptApartFromWhatWasChosen(t *testing.T) {
	b, err := voting.Blueprint()
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]map[string]bool{}
	for _, o := range b.Objects {
		fields[o.Role] = map[string]bool{}
		for _, f := range o.Fields {
			fields[o.Role][f.Name] = true
		}
	}
	if fields["ballot"]["answered_by"] || fields["ballot"]["answered_at"] {
		t.Error("the ballot carries who answered")
	}
	if fields["participation"]["votes"] {
		t.Error("the participation carries what was chosen")
	}
	if !fields["ballot"]["response_id"] {
		t.Error("the ballot has no response link for its owner to read it by")
	}
}
