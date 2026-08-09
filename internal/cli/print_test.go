package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dpleshakov/tfscli/internal/workitem"
)

func TestPrintWorkItemRendersEachKind(t *testing.T) {
	wi := &workitem.WorkItem{
		ID:  12345,
		Rev: 7,
		Fields: []workitem.Field{
			{Name: "System.WorkItemType", Kind: workitem.FieldPlain, Value: "Bug"},
			{Name: "System.AssignedTo", Kind: workitem.FieldIdentity, Value: workitem.Identity{
				DisplayName: "Anna Ivanova",
				UniqueName:  `COMPANY\a.ivanova`,
			}},
			{Name: "System.CreatedDate", Kind: workitem.FieldDateTime, Value: "2026-06-14T09:12:33.117Z"},
			{Name: "Microsoft.VSTS.Scheduling.RemainingWork", Kind: workitem.FieldPlain, Value: "4.5"},
			{Name: "System.Description", Kind: workitem.FieldHTML, Value: "<div>A <b>partial</b> refund sends no email.</div>"},
			{Name: "Microsoft.VSTS.TCM.ReproSteps", Kind: workitem.FieldHTML, Value: "<ol><li>Open the order</li><li>Refund half of it</li></ol>"},
		},
	}

	want := strings.Join([]string{
		"# Work item 12345 (rev 7)",
		"",
		"System.WorkItemType: Bug",
		`System.AssignedTo: Anna Ivanova <COMPANY\a.ivanova>`,
		"System.CreatedDate: 2026-06-14T09:12:33Z",
		"Microsoft.VSTS.Scheduling.RemainingWork: 4.5",
		"",
		"## System.Description",
		"",
		"A **partial** refund sends no email.",
		"",
		"## Microsoft.VSTS.TCM.ReproSteps",
		"",
		"1. Open the order",
		"2. Refund half of it",
		"",
	}, "\n")

	if got := markdown(t, wi); got != want {
		t.Errorf("printWorkItem() wrote:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintWorkItemKeepsFieldOrder(t *testing.T) {
	// A plain field after an HTML one stays after it: the order established by
	// workitem.Get is what --fields asked for.
	wi := &workitem.WorkItem{
		ID:  1,
		Rev: 1,
		Fields: []workitem.Field{
			{Name: "System.Description", Kind: workitem.FieldHTML, Value: "<p>Prose</p>"},
			{Name: "System.State", Kind: workitem.FieldPlain, Value: "Active"},
		},
	}

	want := strings.Join([]string{
		"# Work item 1 (rev 1)",
		"",
		"## System.Description",
		"",
		"Prose",
		"",
		"System.State: Active",
		"",
	}, "\n")

	if got := markdown(t, wi); got != want {
		t.Errorf("printWorkItem() wrote:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintWorkItemSplitsMultilinePlainValues(t *testing.T) {
	wi := &workitem.WorkItem{
		ID:  1,
		Rev: 1,
		Fields: []workitem.Field{
			{Name: "Custom.Notes", Kind: workitem.FieldPlain, Value: "first line\nsecond line"},
		},
	}

	want := strings.Join([]string{
		"# Work item 1 (rev 1)",
		"",
		"## Custom.Notes",
		"",
		"first line",
		"second line",
		"",
	}, "\n")

	if got := markdown(t, wi); got != want {
		t.Errorf("printWorkItem() wrote:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintWorkItemKeepsEmptyValuesInline(t *testing.T) {
	// An empty value is still worth reporting — the field exists — but a
	// heading over nothing is not, and the line carries no trailing space.
	wi := &workitem.WorkItem{
		ID:  1,
		Rev: 1,
		Fields: []workitem.Field{
			{Name: "System.Description", Kind: workitem.FieldHTML, Value: ""},
			{Name: "System.Tags", Kind: workitem.FieldPlain, Value: ""},
		},
	}

	want := strings.Join([]string{
		"# Work item 1 (rev 1)",
		"",
		"System.Description:",
		"System.Tags:",
		"",
	}, "\n")

	if got := markdown(t, wi); got != want {
		t.Errorf("printWorkItem() wrote:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintWorkItemWithoutFields(t *testing.T) {
	want := "# Work item 12345 (rev 7)\n"

	if got := markdown(t, &workitem.WorkItem{ID: 12345, Rev: 7}); got != want {
		t.Errorf("printWorkItem() wrote %q, want %q", got, want)
	}
}

func TestFormatIdentity(t *testing.T) {
	tests := []struct {
		name     string
		identity workitem.Identity
		want     string
	}{
		{
			name:     "both parts",
			identity: workitem.Identity{DisplayName: "Anna Ivanova", UniqueName: `COMPANY\a.ivanova`},
			want:     `Anna Ivanova <COMPANY\a.ivanova>`,
		},
		{
			name:     "display name only",
			identity: workitem.Identity{DisplayName: "Anna Ivanova"},
			want:     "Anna Ivanova",
		},
		{
			// Some on-prem identities carry only the account name.
			name:     "unique name only",
			identity: workitem.Identity{UniqueName: `COMPANY\a.ivanova`},
			want:     `COMPANY\a.ivanova`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatIdentity(tt.identity); got != tt.want {
				t.Errorf("formatIdentity() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "fractional seconds are dropped", value: "2026-06-14T09:12:33.117Z", want: "2026-06-14T09:12:33Z"},
		{name: "already whole seconds", value: "2026-07-01T15:48:02Z", want: "2026-07-01T15:48:02Z"},
		{name: "offset is normalised to UTC", value: "2026-07-01T18:48:02+03:00", want: "2026-07-01T15:48:02Z"},
		{name: "unparseable value passes through", value: "yesterday", want: "yesterday"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatTimestamp(tt.value); got != tt.want {
				t.Errorf("formatTimestamp(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func markdown(t *testing.T, wi *workitem.WorkItem) string {
	t.Helper()
	var buf bytes.Buffer
	if err := printWorkItem(&buf, wi); err != nil {
		t.Fatalf("printWorkItem() returned error: %v", err)
	}
	return buf.String()
}
