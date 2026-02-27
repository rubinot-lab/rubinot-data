package validation

import "testing"

func TestAllCategories(t *testing.T) {
	validator := testValidator()
	categories := validator.AllCategories()

	expectedIDs := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	if len(categories) != len(expectedIDs) {
		t.Fatalf("expected %d categories, got %d", len(expectedIDs), len(categories))
	}

	for i, expectedID := range expectedIDs {
		if categories[i].ID != expectedID {
			t.Fatalf("expected category index %d to have ID %d, got %d", i, expectedID, categories[i].ID)
		}
	}
}

func TestAllTowns(t *testing.T) {
	validator := testValidator()
	towns := validator.AllTowns()

	expectedIDs := []int{1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 13, 14, 33, 63, 66, 67}
	if len(towns) != len(expectedIDs) {
		t.Fatalf("expected %d towns, got %d", len(expectedIDs), len(towns))
	}

	for i, expectedID := range expectedIDs {
		if towns[i].ID != expectedID {
			t.Fatalf("expected town index %d to have ID %d, got %d", i, expectedID, towns[i].ID)
		}
	}
}
