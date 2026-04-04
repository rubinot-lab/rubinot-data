package scraper

import (
	"testing"
)

func TestCompareSchemaDriftDetected(t *testing.T) {
	rawJSON := []byte(`{"worlds": [], "totalOnline": 100, "overallRecord": 200, "overallRecordTime": 300, "newField": "surprise"}`)
	diff, err := CompareSchema("/api/worlds", rawJSON)
	if err != nil {
		t.Fatal(err)
	}
	if diff.Status != "drift" {
		t.Errorf("Status = %q, want drift", diff.Status)
	}
	if len(diff.NewFields) != 1 || diff.NewFields[0] != "newField" {
		t.Errorf("NewFields = %v, want [newField]", diff.NewFields)
	}
	if len(diff.MissingFields) != 0 {
		t.Errorf("MissingFields = %v, want []", diff.MissingFields)
	}
}

func TestCompareSchemaMatch(t *testing.T) {
	rawJSON := []byte(`{"worlds": [], "totalOnline": 100, "overallRecord": 200, "overallRecordTime": 300}`)
	diff, err := CompareSchema("/api/worlds", rawJSON)
	if err != nil {
		t.Fatal(err)
	}
	if diff.Status != "match" {
		t.Errorf("Status = %q, want match", diff.Status)
	}
}

func TestCompareSchemaMissingField(t *testing.T) {
	rawJSON := []byte(`{"worlds": [], "totalOnline": 100}`)
	diff, err := CompareSchema("/api/worlds", rawJSON)
	if err != nil {
		t.Fatal(err)
	}
	if diff.Status != "drift" {
		t.Errorf("Status = %q, want drift", diff.Status)
	}
	if len(diff.MissingFields) != 2 {
		t.Errorf("MissingFields = %v, want 2 items", diff.MissingFields)
	}
}

func TestCompareSchemaNestedDrift(t *testing.T) {
	rawJSON := []byte(`{"worlds": [{"name": "Auroria", "pvpType": "Open PvP", "pvpTypeLabel": "Open PvP", "worldType": "yellow", "locked": false, "playersOnline": 500, "newWorldField": true}], "totalOnline": 100, "overallRecord": 200, "overallRecordTime": 300}`)
	diff, err := CompareSchema("/api/worlds", rawJSON)
	if err != nil {
		t.Fatal(err)
	}
	if diff.Status != "drift" {
		t.Errorf("Status = %q, want drift", diff.Status)
	}
	nd, ok := diff.NestedDiffs["worlds[0]"]
	if !ok {
		t.Fatal("expected nested diff for worlds[0]")
	}
	if len(nd.NewFields) != 1 || nd.NewFields[0] != "newWorldField" {
		t.Errorf("NestedDiffs[worlds[0]].NewFields = %v, want [newWorldField]", nd.NewFields)
	}
}

func TestCompareSchemaUnknownEndpoint(t *testing.T) {
	diff, err := CompareSchema("/api/unknown", []byte(`{"foo": 1}`))
	if err != nil {
		t.Fatal(err)
	}
	if diff.Status != "unknown" {
		t.Errorf("Status = %q, want unknown", diff.Status)
	}
}
