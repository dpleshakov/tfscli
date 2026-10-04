package cli

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

// wiqlResponse is a Query By Wiql response for a tree query, with the url of
// every element that the printer leaves out.
const wiqlResponse = `{
  "queryType": "tree",
  "queryResultType": "workItemLink",
  "asOf": "2026-10-04T10:15:00.483Z",
  "columns": [
    {"referenceName": "System.Id", "name": "ID", "url": "https://tfs.company.com/DefaultCollection/_apis/wit/fields/System.Id"},
    {"referenceName": "System.Title", "name": "Title", "url": "https://tfs.company.com/DefaultCollection/_apis/wit/fields/System.Title"}
  ],
  "workItemRelations": [
    {"rel": null, "source": null, "target": {"id": 297, "url": "https://tfs.company.com/DefaultCollection/_apis/wit/workItems/297"}},
    {"rel": "System.LinkTypes.Hierarchy-Forward", "source": {"id": 297, "url": "https://tfs.company.com/DefaultCollection/_apis/wit/workItems/297"}, "target": {"id": 299, "url": "https://tfs.company.com/DefaultCollection/_apis/wit/workItems/299"}},
    {"rel": "System.LinkTypes.Hierarchy-Forward", "source": {"id": 297, "url": "https://tfs.company.com/DefaultCollection/_apis/wit/workItems/297"}, "target": {"id": 300, "url": "https://tfs.company.com/DefaultCollection/_apis/wit/workItems/300"}}
  ]
}`

const testWiql = "SELECT [System.Id] FROM WorkItems WHERE [System.State] = 'Active'"

func wiqlArgs(extra ...string) []string {
	return append([]string{"wit", "wiql", "query-by-wiql", "--query", testWiql}, extra...)
}

func TestWiqlQueryByWiqlPrintsMarkdown(t *testing.T) {
	s := newServer(t, http.StatusOK, wiqlResponse)

	stdout, stderr, code := execute(t, s, wiqlArgs("-p", "MyProject")...)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing", stderr)
	}
	want := strings.Join([]string{
		"# WIQL query (tree, as of 2026-10-04T10:15:00.483Z)",
		"",
		"Columns: System.Id,System.Title",
		"Relations:",
		"- 297",
		"- 297 -> 299 (System.LinkTypes.Hierarchy-Forward)",
		"- 297 -> 300 (System.LinkTypes.Hierarchy-Forward)",
		"",
	}, "\n")
	if stdout != want {
		t.Errorf("stdout:\n%s\nwant:\n%s", stdout, want)
	}
}

func TestWiqlQueryByWiqlRequest(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		extra   []string
		rawPath string
		query   url.Values
	}{
		{
			name:    "no project at all",
			rawPath: "/DefaultCollection/_apis/wit/wiql",
			query:   url.Values{},
		},
		{
			name:    "project",
			extra:   []string{"-p", "MyProject"},
			rawPath: "/DefaultCollection/MyProject/_apis/wit/wiql",
			query:   url.Values{},
		},
		{
			name:    "project and team with spaces",
			extra:   []string{"-p", "My Project", "--team", "Web Team"},
			rawPath: "/DefaultCollection/My%20Project/Web%20Team/_apis/wit/wiql",
			query:   url.Values{},
		},
		{
			name:    "team with the project from the environment",
			env:     map[string]string{"TFSCLI_PROJECT": "MyProject"},
			extra:   []string{"--team", "Web"},
			rawPath: "/DefaultCollection/MyProject/Web/_apis/wit/wiql",
			query:   url.Values{},
		},
		{
			name:    "top",
			extra:   []string{"-p", "MyProject", "--top", "20"},
			rawPath: "/DefaultCollection/MyProject/_apis/wit/wiql",
			query:   url.Values{"$top": {"20"}},
		},
		{
			name:    "time precision",
			extra:   []string{"-p", "MyProject", "--time-precision"},
			rawPath: "/DefaultCollection/MyProject/_apis/wit/wiql",
			query:   url.Values{"timePrecision": {"true"}},
		},
		{
			// Given explicitly, false is still sent.
			name:    "time precision false",
			extra:   []string{"-p", "MyProject", "--time-precision=false"},
			rawPath: "/DefaultCollection/MyProject/_apis/wit/wiql",
			query:   url.Values{"timePrecision": {"false"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, http.StatusOK, wiqlResponse)
			isolate(t)
			for name, value := range tt.env {
				t.Setenv(name, value)
			}
			t.Setenv("TFSCLI_AUTH", authFor(s))

			var out, errOut strings.Builder
			code := run(testBuild, wiqlArgs(tt.extra...), noTerminal{}, &out, &errOut)

			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, errOut.String())
			}
			if s.method != http.MethodPost {
				t.Errorf("method = %s, want POST", s.method)
			}
			if s.rawPath != tt.rawPath {
				t.Errorf("requested path %q, want %q", s.rawPath, tt.rawPath)
			}
			if !reflect.DeepEqual(s.query, tt.query) {
				t.Errorf("query = %v, want %v", s.query, tt.query)
			}
			want := `{"query":"SELECT [System.Id] FROM WorkItems WHERE [System.State] = 'Active'"}`
			if s.sent != want {
				t.Errorf("body = %s, want %s", s.sent, want)
			}
		})
	}
}

func TestWiqlQueryByWiqlRejectsBadInputBeforeCalling(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "no --query",
			args: []string{"wit", "wiql", "query-by-wiql", "-p", "MyProject"},
			want: `Error [config]: no WIQL query given (pass it with --query, e.g. --query "SELECT [System.Id] FROM WorkItems WHERE [System.State] = 'Active'")` + "\n",
		},
		{
			name: "blank --query",
			args: []string{"wit", "wiql", "query-by-wiql", "-p", "MyProject", "--query", "  "},
			want: `Error [config]: no WIQL query given (pass it with --query, e.g. --query "SELECT [System.Id] FROM WorkItems WHERE [System.State] = 'Active'")` + "\n",
		},
		{
			name: "query as an argument",
			args: []string{"wit", "wiql", "query-by-wiql", "-p", "MyProject", "SELECT [System.Id] FROM WorkItems"},
			want: `Error [config]: query-by-wiql takes no arguments; pass the query with --query, e.g. --query "SELECT [System.Id] FROM WorkItems"` + "\n",
		},
		{
			name: "query split into arguments",
			args: []string{"wit", "wiql", "query-by-wiql", "SELECT", "[System.Id]", "FROM", "WorkItems"},
			want: `Error [config]: query-by-wiql takes no arguments; pass the query with --query, e.g. --query "SELECT [System.Id] FROM WorkItems"` + "\n",
		},
		{
			name: "team without a project",
			args: wiqlArgs("--team", "Web"),
			want: `Error [config]: --team needs a project, and the project is not set (pass -p, set TFSCLI_PROJECT, or add "project" to the config file)` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, http.StatusOK, wiqlResponse)

			stdout, stderr, code := execute(t, s, tt.args...)

			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if stderr != tt.want {
				t.Errorf("stderr = %q, want %q", stderr, tt.want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want nothing", stdout)
			}
			if s.calls != 0 {
				t.Errorf("the server was called %d times, want no call at all", s.calls)
			}
		})
	}
}

func TestWiqlQueryByWiqlReportsASyntaxErrorAsConfig(t *testing.T) {
	body := `{"$id":"1","message":"TF51006: The query statement is missing a FROM clause. The error is caused by «WHERE».","typeKey":"InvalidQueryTextException","errorCode":0}`
	s := newServer(t, http.StatusBadRequest, body)

	stdout, stderr, code := execute(t, s, wiqlArgs("-p", "MyProject")...)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	want := "Error [config]: TF51006: The query statement is missing a FROM clause. The error is caused by «WHERE». (HTTP 400)\n"
	if stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing", stdout)
	}
}
