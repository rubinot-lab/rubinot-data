package scraper

import (
	"encoding/json"
	"sort"
	"strings"
)

type SchemaDiff struct {
	Endpoint      string               `json:"endpoint"`
	Status        string               `json:"status"`
	NewFields     []string             `json:"new_fields,omitempty"`
	MissingFields []string             `json:"missing_fields,omitempty"`
	NestedDiffs   map[string]FieldDiff `json:"nested_diffs,omitempty"`
}

type FieldDiff struct {
	NewFields     []string `json:"new_fields,omitempty"`
	MissingFields []string `json:"missing_fields,omitempty"`
}

func CompareSchema(endpoint string, rawJSON []byte) (*SchemaDiff, error) {
	expected, ok := UpstreamSchemas[endpoint]
	if !ok {
		return &SchemaDiff{Endpoint: endpoint, Status: "unknown"}, nil
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &parsed); err != nil {
		return nil, err
	}

	actualKeys := make([]string, 0, len(parsed))
	for k := range parsed {
		actualKeys = append(actualKeys, k)
	}

	diff := &SchemaDiff{
		Endpoint:    endpoint,
		Status:      "match",
		NestedDiffs: make(map[string]FieldDiff),
	}

	diff.NewFields = setDiff(actualKeys, expected.TopLevel)
	diff.MissingFields = setDiff(expected.TopLevel, actualKeys)

	for field, expectedNested := range expected.Nested {
		actualField := strings.TrimSuffix(field, "[0]")
		isArray := strings.HasSuffix(field, "[0]")

		raw, exists := parsed[actualField]
		if !exists {
			continue
		}

		var nestedObj map[string]json.RawMessage
		if isArray {
			var arr []json.RawMessage
			if json.Unmarshal(raw, &arr) != nil || len(arr) == 0 {
				continue
			}
			if json.Unmarshal(arr[0], &nestedObj) != nil {
				continue
			}
		} else {
			if json.Unmarshal(raw, &nestedObj) != nil {
				continue
			}
		}

		nestedKeys := make([]string, 0, len(nestedObj))
		for k := range nestedObj {
			nestedKeys = append(nestedKeys, k)
		}

		nd := FieldDiff{
			NewFields:     setDiff(nestedKeys, expectedNested),
			MissingFields: setDiff(expectedNested, nestedKeys),
		}
		if len(nd.NewFields) > 0 || len(nd.MissingFields) > 0 {
			diff.NestedDiffs[field] = nd
		}
	}

	if len(diff.NewFields) > 0 || len(diff.MissingFields) > 0 || len(diff.NestedDiffs) > 0 {
		diff.Status = "drift"
	}

	return diff, nil
}

func setDiff(a, b []string) []string {
	bSet := make(map[string]bool, len(b))
	for _, v := range b {
		bSet[v] = true
	}
	var result []string
	for _, v := range a {
		if !bSet[v] {
			result = append(result, v)
		}
	}
	sort.Strings(result)
	return result
}
