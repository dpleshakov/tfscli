package workitem

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// fakeClient stands in for apiclient.Client and records what it was asked for.
type fakeClient struct {
	body []byte
	err  error

	path  string
	query url.Values
}

func (c *fakeClient) Get(_ context.Context, path string, query url.Values) ([]byte, error) {
	c.path, c.query = path, query
	if c.err != nil {
		return nil, c.err
	}
	return c.body, nil
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("cannot read fixture %s: %v", name, err)
	}
	return body
}

func get(t *testing.T, client *fakeClient, fields []string) *WorkItem {
	t.Helper()
	wi, err := Get(context.Background(), client, "MyProject", 12345, fields)
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	return wi
}

func field(t *testing.T, wi *WorkItem, name string) Field {
	t.Helper()
	for _, f := range wi.Fields {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("field %s is missing from the work item", name)
	return Field{}
}

func TestGetRequestsDocumentedPath(t *testing.T) {
	client := &fakeClient{body: fixture(t, "bug.json")}

	get(t, client, nil)

	if want := "MyProject/_apis/wit/workitems/12345"; client.path != want {
		t.Errorf("requested path %q, want %q", client.path, want)
	}
	if got := client.query.Get("fields"); got != "" {
		t.Errorf("requested fields=%q, want no fields parameter", got)
	}
	if _, ok := client.query["api-version"]; ok {
		t.Error("query carries api-version; it is added by apiclient")
	}
}

func TestGetPassesRequestedFields(t *testing.T) {
	client := &fakeClient{body: fixture(t, "bug.json")}

	get(t, client, []string{"System.Title", "System.State", "System.Description"})

	if want := "System.Title,System.State,System.Description"; client.query.Get("fields") != want {
		t.Errorf("requested fields=%q, want %q", client.query.Get("fields"), want)
	}
}

func TestGetReadsIdentityAndRevision(t *testing.T) {
	client := &fakeClient{body: fixture(t, "bug.json")}

	wi := get(t, client, nil)

	if wi.ID != 12345 {
		t.Errorf("ID = %d, want 12345", wi.ID)
	}
	if wi.Rev != 7 {
		t.Errorf("Rev = %d, want 7", wi.Rev)
	}
}

func TestGetTagsFieldKinds(t *testing.T) {
	client := &fakeClient{body: fixture(t, "bug.json")}
	wi := get(t, client, nil)

	tests := []struct {
		name  string
		kind  FieldKind
		value any
	}{
		{
			name:  "System.Title",
			kind:  FieldPlain,
			value: "Payment confirmation email is not sent for partial refunds",
		},
		{
			name:  "System.Description",
			kind:  FieldHTML,
			value: "<div>Customers who receive a <b>partial</b> refund never get the confirmation email.</div>",
		},
		{
			name:  "Microsoft.VSTS.TCM.SystemInfo",
			kind:  FieldHTML,
			value: "<div>Server: PROD-APP-02<br/>Build: 4.18.2231</div>",
		},
		{
			name:  "System.AssignedTo",
			kind:  FieldIdentity,
			value: Identity{DisplayName: "Jane Doe", UniqueName: "COMPANY\\j.doe"},
		},
		{
			// Fractional seconds and a plain Z offset are both RFC 3339.
			name:  "System.CreatedDate",
			kind:  FieldDateTime,
			value: "2026-06-14T09:12:33.117Z",
		},
		{
			name:  "System.ChangedDate",
			kind:  FieldDateTime,
			value: "2026-07-01T15:48:02Z",
		},
		{
			// Numbers keep the text TFS sent rather than going through float64.
			name:  "Microsoft.VSTS.Scheduling.RemainingWork",
			kind:  FieldPlain,
			value: "4.5",
		},
		{
			name:  "System.CommentCount",
			kind:  FieldPlain,
			value: "3",
		},
		{
			name:  "Custom.RequiresRegression",
			kind:  FieldPlain,
			value: "true",
		},
		{
			// "3 - Medium" is not a timestamp and must stay plain.
			name:  "Microsoft.VSTS.Common.Severity",
			kind:  FieldPlain,
			value: "3 - Medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := field(t, wi, tt.name)
			if f.Kind != tt.kind {
				t.Errorf("kind = %q, want %q", f.Kind, tt.kind)
			}
			if f.Value != tt.value {
				t.Errorf("value = %#v, want %#v", f.Value, tt.value)
			}
		})
	}
}

func TestGetKeepsResponseOrderWhenNoFieldsRequested(t *testing.T) {
	client := &fakeClient{body: fixture(t, "bug.json")}

	wi := get(t, client, nil)

	if len(wi.Fields) != 23 {
		t.Fatalf("got %d fields, want the 23 fields of the fixture", len(wi.Fields))
	}
	want := []string{"System.AreaPath", "System.TeamProject", "System.IterationPath", "System.WorkItemType"}
	for i, name := range want {
		if wi.Fields[i].Name != name {
			t.Errorf("field %d is %q, want %q", i, wi.Fields[i].Name, name)
		}
	}
}

func TestGetOrdersFieldsAsRequested(t *testing.T) {
	client := &fakeClient{body: []byte(`{
		"id": 12345,
		"rev": 7,
		"fields": {
			"System.State": "Active",
			"System.Title": "Payment confirmation email is not sent for partial refunds",
			"System.WorkItemType": "Bug"
		}
	}`)}

	// The caller's order, not the response order, and case as typed on the
	// command line.
	wi := get(t, client, []string{"system.title", "System.State"})

	want := []string{"System.Title", "System.State", "System.WorkItemType"}
	for i, name := range want {
		if wi.Fields[i].Name != name {
			t.Errorf("field %d is %q, want %q", i, wi.Fields[i].Name, name)
		}
	}
}

func TestGetPropagatesClientError(t *testing.T) {
	want := &tfserr.Error{Category: tfserr.NotFound, Message: "resource not found", HTTPStatus: 404}
	client := &fakeClient{err: want}

	_, err := Get(context.Background(), client, "MyProject", 12345, nil)

	if !errors.Is(err, error(want)) {
		t.Fatalf("Get() returned %v, want the client error unchanged", err)
	}
}

func TestGetRejectsMalformedResponse(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
	}{
		{name: "not JSON", body: "<html>Gateway timeout</html>"},
		{name: "fields is not an object", body: `{"id": 12345, "rev": 7, "fields": []}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{body: []byte(tt.body)}

			_, err := Get(context.Background(), client, "MyProject", 12345, nil)

			var te *tfserr.Error
			if !errors.As(err, &te) {
				t.Fatalf("Get() returned %v, want a *tfserr.Error", err)
			}
			if te.Category != tfserr.Server {
				t.Errorf("category = %q, want %q", te.Category, tfserr.Server)
			}
		})
	}
}

func TestGetAcceptsResponseWithoutFields(t *testing.T) {
	client := &fakeClient{body: []byte(`{"id": 12345, "rev": 7}`)}

	wi := get(t, client, nil)

	if len(wi.Fields) != 0 {
		t.Errorf("got %d fields, want none", len(wi.Fields))
	}
}

func list(t *testing.T, client *fakeClient, req BatchRequest) *Batch {
	t.Helper()
	batch, err := List(context.Background(), client, "MyProject", req)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}
	return batch
}

func ids(batch *Batch) []int {
	var got []int
	for _, wi := range batch.WorkItems {
		got = append(got, wi.ID)
	}
	return got
}

const threeItems = `{
	"count": 3,
	"value": [
		{"id": 297, "rev": 1, "fields": {"System.Title": "Customer can sign in"}},
		{"id": 299, "rev": 7, "fields": {"System.Title": "JavaScript implementation"}},
		{"id": 300, "rev": 1, "fields": {"System.Title": "Unit testing"}}
	]
}`

func TestListSendsOnlyGivenParameters(t *testing.T) {
	tests := []struct {
		name string
		req  BatchRequest
		want url.Values
	}{
		{
			name: "ids only",
			req:  BatchRequest{IDs: []int{297, 299, 300}},
			want: url.Values{"ids": {"297,299,300"}},
		},
		{
			name: "fields",
			req:  BatchRequest{IDs: []int{297}, Fields: []string{"System.Title", "System.State"}},
			want: url.Values{"ids": {"297"}, "fields": {"System.Title,System.State"}},
		},
		{
			name: "asOf",
			req:  BatchRequest{IDs: []int{297}, AsOf: "2014-12-29T20:49:22.103Z"},
			want: url.Values{"ids": {"297"}, "asOf": {"2014-12-29T20:49:22.103Z"}},
		},
		{
			name: "errorPolicy",
			req:  BatchRequest{IDs: []int{297}, ErrorPolicy: "omit"},
			want: url.Values{"ids": {"297"}, "errorPolicy": {"omit"}},
		},
		{
			name: "everything",
			req: BatchRequest{
				IDs:         []int{297, 299},
				Fields:      []string{"System.Title"},
				AsOf:        "2014-12-29T20:49:22.103Z",
				ErrorPolicy: "fail",
			},
			want: url.Values{
				"ids":         {"297,299"},
				"fields":      {"System.Title"},
				"asOf":        {"2014-12-29T20:49:22.103Z"},
				"errorPolicy": {"fail"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{body: []byte(threeItems)}

			list(t, client, tt.req)

			if want := "MyProject/_apis/wit/workitems"; client.path != want {
				t.Errorf("requested path %q, want %q", client.path, want)
			}
			if got, want := client.query.Encode(), tt.want.Encode(); got != want {
				t.Errorf("query = %q, want %q", got, want)
			}
		})
	}
}

func TestListReturnsEveryWorkItem(t *testing.T) {
	client := &fakeClient{body: []byte(threeItems)}

	batch := list(t, client, BatchRequest{IDs: []int{297, 299, 300}})

	if got, want := ids(batch), []int{297, 299, 300}; !slices.Equal(got, want) {
		t.Errorf("work items %v, want %v", got, want)
	}
	if len(batch.Missing) != 0 {
		t.Errorf("missing %v, want none", batch.Missing)
	}
	if got := field(t, batch.WorkItems[1], "System.Title").Value; got != "JavaScript implementation" {
		t.Errorf("title of 299 = %#v, want the value from the response", got)
	}
	if batch.WorkItems[1].Rev != 7 {
		t.Errorf("rev of 299 = %d, want 7", batch.WorkItems[1].Rev)
	}
}

func TestListKeepsResponseOrder(t *testing.T) {
	client := &fakeClient{body: []byte(threeItems)}

	batch := list(t, client, BatchRequest{IDs: []int{300, 299, 297}})

	if got, want := ids(batch), []int{297, 299, 300}; !slices.Equal(got, want) {
		t.Errorf("work items %v, want the response order %v", got, want)
	}
	if len(batch.Missing) != 0 {
		t.Errorf("missing %v, want none: every requested ID was returned", batch.Missing)
	}
}

func TestListReportsOmittedWorkItemsAsMissing(t *testing.T) {
	client := &fakeClient{body: []byte(`{
		"count": 4,
		"value": [
			{"id": 297, "rev": 1, "fields": {"System.Title": "Customer can sign in"}},
			null,
			{"id": 300, "rev": 1, "fields": {"System.Title": "Unit testing"}},
			null
		]
	}`)}

	// 298 is asked for twice and must be reported once.
	batch := list(t, client, BatchRequest{IDs: []int{301, 297, 298, 300, 298}, ErrorPolicy: "omit"})

	if got, want := ids(batch), []int{297, 300}; !slices.Equal(got, want) {
		t.Errorf("work items %v, want %v", got, want)
	}
	if want := []int{301, 298}; !slices.Equal(batch.Missing, want) {
		t.Errorf("missing %v, want %v in request order", batch.Missing, want)
	}
}

func TestListOrdersFieldsAsRequested(t *testing.T) {
	client := &fakeClient{body: []byte(`{
		"count": 1,
		"value": [
			{"id": 297, "rev": 1, "fields": {"System.State": "New", "System.Title": "Customer can sign in"}}
		]
	}`)}

	batch := list(t, client, BatchRequest{IDs: []int{297}, Fields: []string{"System.Title", "System.State"}})

	if got := batch.WorkItems[0].Fields[0].Name; got != "System.Title" {
		t.Errorf("first field is %q, want System.Title as requested", got)
	}
}

func TestListPropagatesClientError(t *testing.T) {
	want := &tfserr.Error{Category: tfserr.NotFound, Message: "resource not found", HTTPStatus: 404}
	client := &fakeClient{err: want}

	_, err := List(context.Background(), client, "MyProject", BatchRequest{IDs: []int{297}})

	if !errors.Is(err, error(want)) {
		t.Fatalf("List() returned %v, want the client error unchanged", err)
	}
}

func TestListRejectsMalformedResponse(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
	}{
		{name: "not JSON", body: "<html>Gateway timeout</html>"},
		{name: "no value", body: `{"count": 0}`},
		{name: "value is not an array", body: `{"count": 1, "value": {}}`},
		{name: "element is not an object", body: `{"count": 1, "value": [42]}`},
		{name: "fields is not an object", body: `{"count": 1, "value": [{"id": 297, "rev": 1, "fields": []}]}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{body: []byte(tt.body)}

			_, err := List(context.Background(), client, "MyProject", BatchRequest{IDs: []int{297}})

			var te *tfserr.Error
			if !errors.As(err, &te) {
				t.Fatalf("List() returned %v, want a *tfserr.Error", err)
			}
			if te.Category != tfserr.Server {
				t.Errorf("category = %q, want %q", te.Category, tfserr.Server)
			}
		})
	}
}
