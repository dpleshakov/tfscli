package wiql

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"testing"

	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// fakeClient stands in for apiclient.Client and records what it was asked for.
type fakeClient struct {
	body []byte
	err  error

	location string
	path     string
	query    url.Values
	sent     any
}

func (c *fakeClient) Post(_ context.Context, location, path string, query url.Values, body any) ([]byte, error) {
	c.location, c.path, c.query, c.sent = location, path, query, body
	if c.err != nil {
		return nil, c.err
	}
	return c.body, nil
}

const flatResponse = `{
	"queryType": "flat",
	"queryResultType": "workItem",
	"asOf": "2026-10-04T10:15:00Z",
	"columns": [
		{"referenceName": "System.Id", "name": "ID", "url": "https://tfs/_apis/wit/fields/System.Id"},
		{"referenceName": "System.Title", "name": "Title", "url": "https://tfs/_apis/wit/fields/System.Title"}
	],
	"workItems": [
		{"id": 300, "url": "https://tfs/_apis/wit/workItems/300"},
		{"id": 297, "url": "https://tfs/_apis/wit/workItems/297"},
		{"id": 299, "url": "https://tfs/_apis/wit/workItems/299"}
	]
}`

const treeResponse = `{
	"queryType": "tree",
	"queryResultType": "workItemLink",
	"asOf": "2026-10-04T10:15:00Z",
	"columns": [
		{"referenceName": "System.Id", "name": "ID"},
		{"referenceName": "System.Title", "name": "Title"}
	],
	"workItemRelations": [
		{"rel": null, "source": null, "target": {"id": 297, "url": "https://tfs/_apis/wit/workItems/297"}},
		{"rel": "System.LinkTypes.Hierarchy-Forward", "source": {"id": 297}, "target": {"id": 299}},
		{"rel": "System.LinkTypes.Hierarchy-Forward", "source": {"id": 297}, "target": {"id": 300}}
	]
}`

func query(t *testing.T, client *fakeClient, req Request) *Result {
	t.Helper()
	result, err := QueryByWiql(context.Background(), client, req)
	if err != nil {
		t.Fatalf("QueryByWiql() returned error: %v", err)
	}
	return result
}

func TestQueryByWiqlRequestsPath(t *testing.T) {
	tests := []struct {
		name    string
		project string
		team    string
		want    string
	}{
		{name: "collection level", want: "_apis/wit/wiql"},
		{name: "project", project: "MyProject", want: "MyProject/_apis/wit/wiql"},
		{name: "project and team", project: "MyProject", team: "Web", want: "MyProject/Web/_apis/wit/wiql"},
		{name: "team with a space", project: "My Project", team: "Web Team", want: "My%20Project/Web%20Team/_apis/wit/wiql"},
		{name: "percent signs", project: "100% Done", team: "A%20B", want: "100%25%20Done/A%2520B/_apis/wit/wiql"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{body: []byte(flatResponse)}

			query(t, client, Request{Project: tt.project, Team: tt.team, Query: "SELECT [System.Id] FROM WorkItems"})

			if client.path != tt.want {
				t.Errorf("requested path %q, want %q", client.path, tt.want)
			}
			if want := "1a9c53f7-f243-4447-b110-35ef023636e4"; client.location != want {
				t.Errorf("location = %q, want %q", client.location, want)
			}
		})
	}
}

func TestQueryByWiqlSendsOnlyGivenParameters(t *testing.T) {
	const wiql = "SELECT [System.Id] FROM WorkItems WHERE [System.State] = 'Active'"
	tests := []struct {
		name  string
		req   Request
		query url.Values
	}{
		{
			name:  "query only",
			req:   Request{Query: wiql},
			query: url.Values{},
		},
		{
			name:  "top",
			req:   Request{Query: wiql, Top: new(50)},
			query: url.Values{"$top": {"50"}},
		},
		{
			name:  "time precision",
			req:   Request{Query: wiql, TimePrecision: new(true)},
			query: url.Values{"timePrecision": {"true"}},
		},
		{
			name:  "time precision false",
			req:   Request{Query: wiql, TimePrecision: new(false)},
			query: url.Values{"timePrecision": {"false"}},
		},
		{
			name:  "top and time precision",
			req:   Request{Query: wiql, Top: new(5), TimePrecision: new(true)},
			query: url.Values{"$top": {"5"}, "timePrecision": {"true"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{body: []byte(flatResponse)}

			query(t, client, tt.req)

			if !reflect.DeepEqual(client.query, tt.query) {
				t.Errorf("query = %v, want %v", client.query, tt.query)
			}
			sent, err := json.Marshal(client.sent)
			if err != nil {
				t.Fatalf("cannot encode the body sent: %v", err)
			}
			want, _ := json.Marshal(map[string]string{"query": wiql})
			if string(sent) != string(want) {
				t.Errorf("body = %s, want %s", sent, want)
			}
		})
	}
}

func TestQueryByWiqlReadsFlatResult(t *testing.T) {
	result := query(t, &fakeClient{body: []byte(flatResponse)}, Request{Query: "q"})

	want := &Result{
		QueryType: "flat",
		AsOf:      "2026-10-04T10:15:00Z",
		Columns:   []string{"System.Id", "System.Title"},
		WorkItems: []int{300, 297, 299},
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestQueryByWiqlReadsLinkResult(t *testing.T) {
	result := query(t, &fakeClient{body: []byte(treeResponse)}, Request{Query: "q"})

	want := &Result{
		QueryType: "tree",
		AsOf:      "2026-10-04T10:15:00Z",
		Columns:   []string{"System.Id", "System.Title"},
		Link:      true,
		Relations: []Relation{
			{Target: 297},
			{Source: 297, HasSource: true, Target: 299, Rel: "System.LinkTypes.Hierarchy-Forward"},
			{Source: 297, HasSource: true, Target: 300, Rel: "System.LinkTypes.Hierarchy-Forward"},
		},
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestQueryByWiqlReadsEmptyResults(t *testing.T) {
	tests := []struct {
		name string
		body string
		link bool
	}{
		{name: "flat", body: `{"queryType": "flat", "queryResultType": "workItem", "columns": [], "workItems": []}`},
		{name: "flat without the list", body: `{"queryType": "flat", "queryResultType": "workItem"}`},
		{name: "link", body: `{"queryType": "oneHop", "queryResultType": "workItemLink", "workItemRelations": []}`, link: true},
		{name: "link without the result type", body: `{"queryType": "tree"}`, link: true},
		{name: "link known only by its relations", body: `{"workItemRelations": []}`, link: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := query(t, &fakeClient{body: []byte(tt.body)}, Request{Query: "q"})

			if result.Link != tt.link {
				t.Errorf("Link = %v, want %v", result.Link, tt.link)
			}
			if len(result.WorkItems) != 0 || len(result.Relations) != 0 {
				t.Errorf("result = %+v, want no work items and no relations", result)
			}
		})
	}
}

func TestQueryByWiqlPropagatesClientError(t *testing.T) {
	want := &tfserr.Error{Category: tfserr.Config, Message: "TF51005: The query references a field that does not exist.", HTTPStatus: 400}
	client := &fakeClient{err: want}

	_, err := QueryByWiql(context.Background(), client, Request{Query: "q"})

	if !errors.Is(err, error(want)) {
		t.Fatalf("QueryByWiql() returned %v, want the client error unchanged", err)
	}
}

func TestQueryByWiqlRejectsMalformedResponse(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
	}{
		{name: "not JSON", body: "<html>Gateway timeout</html>"},
		{name: "not an object", body: `[1, 2]`},
		{name: "work items is not an array", body: `{"queryType": "flat", "workItems": {}}`},
		{name: "work item without id", body: `{"queryType": "flat", "workItems": [{"url": "x"}]}`},
		{name: "relation without target", body: `{"queryType": "tree", "workItemRelations": [{"rel": null, "source": null}]}`},
		{name: "relation source without id", body: `{"queryType": "tree", "workItemRelations": [{"source": {}, "target": {"id": 1}}]}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := QueryByWiql(context.Background(), &fakeClient{body: []byte(tt.body)}, Request{Query: "q"})

			var te *tfserr.Error
			if !errors.As(err, &te) {
				t.Fatalf("QueryByWiql() returned %v, want a *tfserr.Error", err)
			}
			if te.Category != tfserr.Server {
				t.Errorf("category = %q, want %q", te.Category, tfserr.Server)
			}
		})
	}
}
