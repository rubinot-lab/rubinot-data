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

const eventsNestedHTMLFixture = `
<html>
  <body>
    <span class="flex items-center gap-2">Fevereiro 2026</span>
    <table class="w-full table-fixed border-collapse text-xs">
      <tr><th>Seg</th><th>Ter</th><th>Qua</th><th>Qui</th><th>Sex</th><th>Sab</th><th>Dom</th></tr>
      <tr>
        <td>
          <div>14</div>
          <div onmouseover="tooltip(this)">
            <div class="calendar-event"><span>A Piece of Cake</span></div>
            <div class="calendar-event"><span>*Castle</span></div>
          </div>
        </td>
        <td>
          <div>15</div>
          <div onmouseover="tooltip(this)">
            <div class="calendar-event"><span>Gaz'Haragoth</span></div>
            <div class="calendar-event"><span>*Castle</span></div>
            <div class="calendar-event"><span>*Skill Event</span></div>
          </div>
        </td>
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

func TestParseEventsHTMLNestedDivs(t *testing.T) {
	result, err := parseEventsHTML(eventsNestedHTMLFixture)
	if err != nil {
		t.Fatalf("expected no parse error, got %v", err)
	}
	if len(result.Days) != 2 {
		t.Fatalf("expected 2 days, got %d", len(result.Days))
	}

	day14 := result.Days[0]
	if day14.Day != 14 {
		t.Fatalf("expected day 14, got %d", day14.Day)
	}
	if len(day14.Events) != 2 {
		t.Fatalf("expected 2 events on day 14, got %d: %v", len(day14.Events), day14.Events)
	}
	if day14.Events[0] != "A Piece of Cake" {
		t.Fatalf("expected 'A Piece of Cake', got %q", day14.Events[0])
	}
	if day14.Events[1] != "Castle" {
		t.Fatalf("expected 'Castle', got %q", day14.Events[1])
	}

	day15 := result.Days[1]
	if day15.Day != 15 {
		t.Fatalf("expected day 15, got %d", day15.Day)
	}
	if len(day15.Events) != 3 {
		t.Fatalf("expected 3 events on day 15, got %d: %v", len(day15.Events), day15.Events)
	}
	if day15.Events[0] != "Gaz'Haragoth" {
		t.Fatalf("expected 'Gaz'Haragoth', got %q", day15.Events[0])
	}
	if day15.Events[1] != "Castle" {
		t.Fatalf("expected 'Castle', got %q", day15.Events[1])
	}
	if day15.Events[2] != "Skill Event" {
		t.Fatalf("expected 'Skill Event', got %q", day15.Events[2])
	}

	expectedAllEvents := map[string]bool{
		"A Piece of Cake": true,
		"Castle":          true,
		"Gaz'Haragoth":    true,
		"Skill Event":     true,
	}
	if len(result.AllEvents) != len(expectedAllEvents) {
		t.Fatalf("expected %d unique events, got %d: %v", len(expectedAllEvents), len(result.AllEvents), result.AllEvents)
	}
	for _, event := range result.AllEvents {
		if !expectedAllEvents[event] {
			t.Fatalf("unexpected event %q in AllEvents", event)
		}
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
