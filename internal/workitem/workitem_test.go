package workitem

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
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
			value: Identity{DisplayName: "Anna Ivanova", UniqueName: "COMPANY\\a.ivanova"},
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
