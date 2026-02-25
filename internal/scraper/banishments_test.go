package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchBanishmentsFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/bans")
		assertQuery(t, r, "world", "15")
		assertQuery(t, r, "page", "2")
		writeJSON(w, map[string]any{
			"bans": []map[string]any{{
				"account_id":     1,
				"account_name":   "acc",
				"main_character": "Hero",
				"reason":         "Rule 2B",
				"banned_at":      "1772043027",
				"expires_at":     "-1",
				"banned_by":      "GM",
				"is_permanent":   true,
			}},
			"totalCount":  367,
			"totalPages":  8,
			"currentPage": 2,
		})
	}))
	defer api.Close()

	fs := newFlareSolverrJSONServer(t, nil)
	defer fs.Close()

	result, _, err := FetchBanishments(context.Background(), baseURLOf(api), "Belaria", 15, 2, testFetchOptions(fs.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Page != 2 || result.TotalBans != 367 {
		t.Fatalf("unexpected page payload: %+v", result)
	}
	if len(result.Entries) != 1 || !result.Entries[0].IsPermanent {
		t.Fatalf("unexpected entries: %+v", result.Entries)
	}
}
