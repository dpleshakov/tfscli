package wiql

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// APIClient is the part of the API client this package uses: Post for Query
// By Wiql.
type APIClient interface {
	Post(ctx context.Context, path string, query url.Values, body any) ([]byte, error)
}

// Request holds the parameters of Query By Wiql. Project and Team are the
// optional path segments: an empty Project runs the query at collection
// level, and Team is used only together with a Project, which the caller
// ensures. Top and TimePrecision are sent only when they are not nil, so that
// the server's defaults apply otherwise.
type Request struct {
	Project       string
	Team          string
	Query         string
	Top           *int
	TimePrecision *bool
}

// Relation is one element of the result of a link query. A relation without a
// source is a root of the result; its Rel is empty.
type Relation struct {
	// Source is the ID of the source work item; it is meaningful only when
	// HasSource is true.
	Source    int
	HasSource bool
	Target    int
	Rel       string
}

// Result is what Query By Wiql returned. A flat query fills WorkItems; a link
// query (tree or oneHop) fills Relations instead, and Link tells the two
// apart even when the list is empty. Both lists keep the response order.
type Result struct {
	QueryType string
	AsOf      string
	// Columns are the reference names of the columns the query selected.
	Columns   []string
	Link      bool
	WorkItems []int
	Relations []Relation
}

// QueryByWiql runs a WIQL query with Query By Wiql. The query text is passed
// to the server as is.
func QueryByWiql(ctx context.Context, client APIClient, req Request) (*Result, error) {
	query := url.Values{}
	if req.Top != nil {
		query.Set("$top", strconv.Itoa(*req.Top))
	}
	if req.TimePrecision != nil {
		query.Set("timePrecision", strconv.FormatBool(*req.TimePrecision))
	}
	body := struct {
		Query string `json:"query"`
	}{req.Query}

	resp, err := client.Post(ctx, path(req), query, body)
	if err != nil {
		return nil, err
	}
	return parse(resp)
}

// path builds {project}/{team}/_apis/wit/wiql, leaving out the segments that
// are not given. The project and the team are escaped, since the API client
// takes the path as already escaped.
func path(req Request) string {
	p := "_apis/wit/wiql"
	if req.Team != "" {
		p = url.PathEscape(req.Team) + "/" + p
	}
	if req.Project != "" {
		p = url.PathEscape(req.Project) + "/" + p
	}
	return p
}

type reference struct {
	ID *int `json:"id"`
}

func parse(body []byte) (*Result, error) {
	var payload struct {
		QueryType       string `json:"queryType"`
		QueryResultType string `json:"queryResultType"`
		AsOf            string `json:"asOf"`
		Columns         []struct {
			ReferenceName string `json:"referenceName"`
		} `json:"columns"`
		WorkItems         []reference `json:"workItems"`
		WorkItemRelations *[]struct {
			Rel    string     `json:"rel"`
			Source *reference `json:"source"`
			Target *reference `json:"target"`
		} `json:"workItemRelations"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, malformed(err)
	}

	result := &Result{QueryType: payload.QueryType, AsOf: payload.AsOf}
	for _, c := range payload.Columns {
		result.Columns = append(result.Columns, c.ReferenceName)
	}

	// The result type says whether the query was a link query; the query
	// type and the presence of relations back it up for a response that
	// leaves the result type out.
	result.Link = payload.QueryResultType == "workItemLink" ||
		payload.QueryType == "tree" || payload.QueryType == "oneHop" ||
		payload.WorkItemRelations != nil

	for _, wi := range payload.WorkItems {
		if wi.ID == nil {
			return nil, malformed(fmt.Errorf("a work item has no id"))
		}
		result.WorkItems = append(result.WorkItems, *wi.ID)
	}
	if payload.WorkItemRelations != nil {
		for _, r := range *payload.WorkItemRelations {
			if r.Target == nil || r.Target.ID == nil {
				return nil, malformed(fmt.Errorf("a relation has no target"))
			}
			rel := Relation{Target: *r.Target.ID, Rel: r.Rel}
			if r.Source != nil {
				if r.Source.ID == nil {
					return nil, malformed(fmt.Errorf("a relation source has no id"))
				}
				rel.Source, rel.HasSource = *r.Source.ID, true
			}
			result.Relations = append(result.Relations, rel)
		}
	}
	return result, nil
}

// malformed reports a response that does not have the documented shape. It is
// the server's output, so it lands in the server category.
func malformed(cause error) error {
	return &tfserr.Error{
		Category: tfserr.Server,
		Message:  "TFS returned a response that could not be parsed",
		Cause:    cause,
	}
}
