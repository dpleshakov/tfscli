package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dpleshakov/tfscli/internal/wiql"
	"github.com/dpleshakov/tfscli/internal/workitem"
)

func TestPrintWorkItemRendersEachKind(t *testing.T) {
	wi := &workitem.WorkItem{
		ID:  12345,
		Rev: 7,
		Fields: []workitem.Field{
			{Name: "System.WorkItemType", Kind: workitem.FieldPlain, Value: "Bug"},
			{Name: "System.AssignedTo", Kind: workitem.FieldIdentity, Value: workitem.Identity{
				DisplayName: "Jane Doe",
				UniqueName:  `COMPANY\j.doe`,
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
		`System.AssignedTo: Jane Doe <COMPANY\j.doe>`,
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
			identity: workitem.Identity{DisplayName: "Jane Doe", UniqueName: `COMPANY\j.doe`},
			want:     `Jane Doe <COMPANY\j.doe>`,
		},
		{
			name:     "display name only",
			identity: workitem.Identity{DisplayName: "Jane Doe"},
			want:     "Jane Doe",
		},
		{
			// Some on-prem identities carry only the account name.
			name:     "unique name only",
			identity: workitem.Identity{UniqueName: `COMPANY\j.doe`},
			want:     `COMPANY\j.doe`,
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

func batchMarkdown(t *testing.T, batch *workitem.Batch) string {
	t.Helper()
	var buf bytes.Buffer
	if err := printWorkItems(&buf, batch); err != nil {
		t.Fatalf("printWorkItems() returned error: %v", err)
	}
	return buf.String()
}

func titled(id, rev int, title string) *workitem.WorkItem {
	return &workitem.WorkItem{ID: id, Rev: rev, Fields: []workitem.Field{
		{Name: "System.Title", Kind: workitem.FieldPlain, Value: title},
		{Name: "System.Description", Kind: workitem.FieldHTML, Value: "<p>" + title + " in detail.</p>"},
	}}
}

func TestPrintWorkItemsSeparatesWorkItemsByABlankLine(t *testing.T) {
	batch := &workitem.Batch{WorkItems: []*workitem.WorkItem{
		titled(299, 7, "JavaScript implementation"),
		titled(297, 1, "Customer can sign in"),
	}}

	want := strings.Join([]string{
		"# Work item 299 (rev 7)",
		"",
		"System.Title: JavaScript implementation",
		"",
		"## System.Description",
		"",
		"JavaScript implementation in detail.",
		"",
		"# Work item 297 (rev 1)",
		"",
		"System.Title: Customer can sign in",
		"",
		"## System.Description",
		"",
		"Customer can sign in in detail.",
		"",
	}, "\n")

	if got := batchMarkdown(t, batch); got != want {
		t.Errorf("printWorkItems() wrote:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintWorkItemsListsMissingAfterReturned(t *testing.T) {
	batch := &workitem.Batch{
		WorkItems: []*workitem.WorkItem{{ID: 297, Rev: 1, Fields: []workitem.Field{
			{Name: "System.Title", Kind: workitem.FieldPlain, Value: "Customer can sign in"},
		}}},
		Missing: []int{301, 298},
	}

	want := strings.Join([]string{
		"# Work item 297 (rev 1)",
		"",
		"System.Title: Customer can sign in",
		"",
		"# Work item 301 (not returned: it does not exist, or the PAT has no access to it)",
		"",
		"# Work item 298 (not returned: it does not exist, or the PAT has no access to it)",
		"",
	}, "\n")

	if got := batchMarkdown(t, batch); got != want {
		t.Errorf("printWorkItems() wrote:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintWorkItemsWithOnlyMissing(t *testing.T) {
	batch := &workitem.Batch{Missing: []int{298}}

	want := "# Work item 298 (not returned: it does not exist, or the PAT has no access to it)\n"

	if got := batchMarkdown(t, batch); got != want {
		t.Errorf("printWorkItems() wrote %q, want %q", got, want)
	}
}

func TestPrintWorkItemsMatchesPrintWorkItemForOne(t *testing.T) {
	wi := titled(297, 1, "Customer can sign in")

	if got, want := batchMarkdown(t, &workitem.Batch{WorkItems: []*workitem.WorkItem{wi}}), markdown(t, wi); got != want {
		t.Errorf("printWorkItems() wrote:\n%s\nprintWorkItem() wrote:\n%s", got, want)
	}
}

func wiqlMarkdown(t *testing.T, result *wiql.Result) string {
	t.Helper()
	var buf bytes.Buffer
	if err := printWiqlResult(&buf, result); err != nil {
		t.Fatalf("printWiqlResult() returned error: %v", err)
	}
	return buf.String()
}

func TestPrintWiqlResultFlat(t *testing.T) {
	result := &wiql.Result{
		QueryType: "flat",
		AsOf:      "2026-10-04T10:15:00.483Z",
		Columns:   []string{"System.Id", "System.Title", "System.State"},
		WorkItems: []int{300, 297, 299},
	}

	want := strings.Join([]string{
		"# WIQL query (flat, as of 2026-10-04T10:15:00.483Z)",
		"",
		"Columns: System.Id,System.Title,System.State",
		"Work items: 300,297,299",
		"",
	}, "\n")

	if got := wiqlMarkdown(t, result); got != want {
		t.Errorf("printWiqlResult() wrote:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintWiqlResultLink(t *testing.T) {
	result := &wiql.Result{
		QueryType: "oneHop",
		AsOf:      "2026-10-04T10:15:00Z",
		Columns:   []string{"System.Id"},
		Link:      true,
		Relations: []wiql.Relation{
			{Target: 297},
			{Source: 297, HasSource: true, Target: 299, Rel: "System.LinkTypes.Related"},
			{Source: 297, HasSource: true, Target: 300},
		},
	}

	want := strings.Join([]string{
		"# WIQL query (oneHop, as of 2026-10-04T10:15:00Z)",
		"",
		"Columns: System.Id",
		"Relations:",
		"- 297",
		"- 297 -> 299 (System.LinkTypes.Related)",
		"- 297 -> 300",
		"",
	}, "\n")

	if got := wiqlMarkdown(t, result); got != want {
		t.Errorf("printWiqlResult() wrote:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintWiqlResultEmpty(t *testing.T) {
	tests := []struct {
		name   string
		result *wiql.Result
		want   string
	}{
		{
			name:   "flat",
			result: &wiql.Result{QueryType: "flat", AsOf: "2026-10-04T10:15:00Z", Columns: []string{"System.Id"}},
			want:   "# WIQL query (flat, as of 2026-10-04T10:15:00Z)\n\nColumns: System.Id\nWork items: none\n",
		},
		{
			name:   "link",
			result: &wiql.Result{QueryType: "tree", AsOf: "2026-10-04T10:15:00Z", Columns: []string{"System.Id"}, Link: true},
			want:   "# WIQL query (tree, as of 2026-10-04T10:15:00Z)\n\nColumns: System.Id\nRelations: none\n",
		},
		{
			name:   "nothing but the lists",
			result: &wiql.Result{},
			want:   "# WIQL query\n\nColumns: none\nWork items: none\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wiqlMarkdown(t, tt.result); got != tt.want {
				t.Errorf("printWiqlResult() wrote:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

// TestPrintWiqlResultMatchesPlan checks the example of a link query recorded
// in the tasks file byte for byte.
func TestPrintWiqlResultMatchesPlan(t *testing.T) {
	result := &wiql.Result{
		QueryType: "tree",
		AsOf:      "2026-10-04T10:15:00Z",
		Columns:   []string{"System.Id", "System.Title"},
		Link:      true,
		Relations: []wiql.Relation{
			{Target: 297},
			{Source: 297, HasSource: true, Target: 299, Rel: "System.LinkTypes.Hierarchy-Forward"},
			{Source: 297, HasSource: true, Target: 300, Rel: "System.LinkTypes.Hierarchy-Forward"},
		},
	}

	want := "# WIQL query (tree, as of 2026-10-04T10:15:00Z)\n" +
		"\n" +
		"Columns: System.Id,System.Title\n" +
		"Relations:\n" +
		"- 297\n" +
		"- 297 -> 299 (System.LinkTypes.Hierarchy-Forward)\n" +
		"- 297 -> 300 (System.LinkTypes.Hierarchy-Forward)\n"

	if got := wiqlMarkdown(t, result); got != want {
		t.Errorf("printWiqlResult() wrote:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatAsOf(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{value: "2026-10-04T10:15:00.4836721Z", want: "2026-10-04T10:15:00.4836721Z"},
		{value: "2026-10-04T10:15:00Z", want: "2026-10-04T10:15:00Z"},
		{value: "2026-10-04T13:15:00.25+03:00", want: "2026-10-04T10:15:00.25Z"},
		{value: "yesterday", want: "yesterday"},
	}
	for _, tt := range tests {
		if got := formatAsOf(tt.value); got != tt.want {
			t.Errorf("formatAsOf(%q) = %q, want %q", tt.value, got, tt.want)
		}
	}
}
