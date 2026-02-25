package scraper

import (
	"context"
	"testing"
)

const eventsHTMLFixture = `
<html>
  <body>
    <span class="flex items-center gap-2">Fevereiro 2026</span>
    <table class="w-full table-fixed border-collapse text-xs">
      <tr><th>Seg</th><th>Ter</th><th>Qua</th><th>Qui</th><th>Sex</th><th>Sab</th><th>Dom</th></tr>
      <tr>
        <td><div>24</div><div>Castle</div></td>
        <td><div>25</div><div>*Skill Event</div></td>
      </tr>
    </table>
  </body>
</html>
`

func TestParseEventsHTML(t *testing.T) {
	result, err := parseEventsHTML(eventsHTMLFixture)
	if err != nil {
		t.Fatalf("expected no parse error, got %v", err)
	}
	if len(result.Days) != 2 {
		t.Fatalf("expected 2 days, got %d", len(result.Days))
	}
	if result.Days[0].Day != 24 || len(result.AllEvents) == 0 {
		t.Fatalf("unexpected events payload: %+v", result)
	}
}

func TestFetchEventsSchedule(t *testing.T) {
	fs := newFlareSolverrJSONServer(t, func(targetURL string) string {
		if targetURL != "https://www.rubinot.com.br/events" {
			t.Fatalf("unexpected target URL %s", targetURL)
		}
		return eventsHTMLFixture
	})
	defer fs.Close()

	result, sourceURL, err := FetchEventsSchedule(context.Background(), "https://www.rubinot.com.br", 0, 0, testFetchOptions(fs.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sourceURL != "https://www.rubinot.com.br/events" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if len(result.Days) != 2 {
		t.Fatalf("expected 2 days, got %d", len(result.Days))
	}
}
