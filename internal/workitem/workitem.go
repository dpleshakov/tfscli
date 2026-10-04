package workitem

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// APIClient is the part of the API client this package uses: Get for Get Work
// Item and List, Post for Get Work Items Batch.
type APIClient interface {
	Get(ctx context.Context, path string, query url.Values) ([]byte, error)
	Post(ctx context.Context, path string, query url.Values, body any) ([]byte, error)
}

// FieldKind tells the printer how to render a field value.
type FieldKind string

// The kinds a field value can take: FieldPlain prints as it arrived,
// FieldHTML goes through the markdown conversion, FieldIdentity is a person,
// and FieldDateTime is a timestamp printed in a normalized form.
const (
	FieldPlain    FieldKind = "plain"
	FieldHTML     FieldKind = "html"
	FieldIdentity FieldKind = "identity"
	FieldDateTime FieldKind = "datetime"
)

// Identity is the value of a field naming a person. TFS sends more keys than
// these two — image URLs, descriptors, internal ids — and none of them are
// worth printing.
type Identity struct {
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
}

// Field is one work item field with the kind it should be rendered as. Value
// is an Identity for FieldIdentity and a string for every other kind: field
// values that are not JSON strings — numbers, booleans, nested objects — keep
// the exact text TFS sent, so nothing is lost or reformatted here.
type Field struct {
	Name  string
	Kind  FieldKind
	Value any
}

// WorkItem is a single work item as returned by Get Work Item.
type WorkItem struct {
	ID     int
	Rev    int
	Fields []Field
}

// BatchRequest holds the parameters shared by List and Get Work Items Batch.
// Empty Fields, AsOf, and ErrorPolicy are not sent, so the server's defaults
// apply; AsOf and ErrorPolicy are passed through without validation.
type BatchRequest struct {
	IDs         []int
	Fields      []string
	AsOf        string
	ErrorPolicy string
}

// Batch is the result of reading several work items. WorkItems are in the
// order the server returned them. Missing are the requested IDs, in request
// order and without duplicates, that the server did not return: with
// errorPolicy=omit it answers null in place of a work item that does not exist
// or cannot be read, and the response order is not guaranteed to match the
// request, so the IDs are found by comparison rather than by position.
type Batch struct {
	WorkItems []*WorkItem
	Missing   []int
}

// htmlFields are the fields TFS stores as HTML. There is nothing in the
// response that marks them — the value is a JSON string like any other — so
// the set has to be known in advance.
var htmlFields = map[string]bool{
	"system.description":                       true,
	"microsoft.vsts.tcm.reprosteps":            true,
	"microsoft.vsts.tcm.systeminfo":            true,
	"microsoft.vsts.common.acceptancecriteria": true,
}

// Get retrieves one work item. An empty fields slice asks TFS for every field.
// Values are returned as received; markdown conversion belongs to the printer.
func Get(ctx context.Context, client APIClient, project string, id int, fields []string) (*WorkItem, error) {
	query := url.Values{}
	if len(fields) > 0 {
		query.Set("fields", strings.Join(fields, ","))
	}

	body, err := client.Get(ctx, fmt.Sprintf("%s/_apis/wit/workitems/%d", project, id), query)
	if err != nil {
		return nil, err
	}
	return parse(body, fields)
}

// List retrieves several work items with Work Items - List, a GET carrying the
// IDs in the query string.
func List(ctx context.Context, client APIClient, project string, req BatchRequest) (*Batch, error) {
	query := url.Values{}
	query.Set("ids", joinIDs(req.IDs))
	if len(req.Fields) > 0 {
		query.Set("fields", strings.Join(req.Fields, ","))
	}
	if req.AsOf != "" {
		query.Set("asOf", req.AsOf)
	}
	if req.ErrorPolicy != "" {
		query.Set("errorPolicy", req.ErrorPolicy)
	}

	body, err := client.Get(ctx, project+"/_apis/wit/workitems", query)
	if err != nil {
		return nil, err
	}
	return parseBatch(body, req)
}

// GetBatch retrieves several work items with Get Work Items Batch, a POST
// carrying the parameters in a JSON body. It exists for requests whose query
// string would be too long for List, and needs Azure DevOps Server 2019 or
// later.
func GetBatch(ctx context.Context, client APIClient, project string, req BatchRequest) (*Batch, error) {
	// Omitted parameters are left out of the body rather than sent empty, so
	// that the server's defaults apply as they do for List.
	body := struct {
		IDs         []int    `json:"ids"`
		Fields      []string `json:"fields,omitempty"`
		AsOf        string   `json:"asOf,omitempty"`
		ErrorPolicy string   `json:"errorPolicy,omitempty"`
	}{req.IDs, req.Fields, req.AsOf, req.ErrorPolicy}

	resp, err := client.Post(ctx, project+"/_apis/wit/workitemsbatch", nil, body)
	if err != nil {
		return nil, err
	}
	return parseBatch(resp, req)
}

func joinIDs(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}

// parseBatch reads the {"count", "value"} envelope of a multi-item response.
// Each element is parsed as a single work item; a null element is a work item
// the server omitted.
func parseBatch(body []byte, req BatchRequest) (*Batch, error) {
	var payload struct {
		Value *[]json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, malformed(err)
	}
	if payload.Value == nil {
		return nil, malformed(fmt.Errorf("value is missing"))
	}

	batch := &Batch{}
	returned := make(map[int]bool, len(*payload.Value))
	for _, raw := range *payload.Value {
		if string(raw) == "null" {
			continue
		}
		wi, err := parse(raw, req.Fields)
		if err != nil {
			return nil, err
		}
		batch.WorkItems = append(batch.WorkItems, wi)
		returned[wi.ID] = true
	}

	for _, id := range req.IDs {
		if !returned[id] {
			batch.Missing = append(batch.Missing, id)
			// Marking it keeps a repeated ID from being reported twice.
			returned[id] = true
		}
	}
	return batch, nil
}

func parse(body []byte, requested []string) (*WorkItem, error) {
	var payload struct {
		ID     int             `json:"id"`
		Rev    int             `json:"rev"`
		Fields json.RawMessage `json:"fields"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, malformed(err)
	}

	wi := &WorkItem{ID: payload.ID, Rev: payload.Rev}
	fields, err := decodeFields(payload.Fields)
	if err != nil {
		return nil, err
	}
	wi.Fields = order(fields, requested)
	return wi, nil
}

// decodeFields walks the "fields" object with a token decoder rather than
// unmarshalling it into a map, because a map loses the order TFS sent and the
// printer has nothing better to fall back on.
func decodeFields(raw json.RawMessage) ([]Field, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, malformed(fmt.Errorf("fields is not a JSON object"))
	}

	var fields []Field
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, malformed(err)
		}
		name, ok := tok.(string)
		if !ok {
			return nil, malformed(fmt.Errorf("field name is not a string"))
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, malformed(err)
		}
		fields = append(fields, classify(name, value))
	}
	return fields, nil
}

// classify assigns a kind by looking at the field name for HTML and at the
// value itself for the rest: identity fields are recognized by their shape and
// dates by being parseable, so custom fields are covered without listing them.
func classify(name string, raw json.RawMessage) Field {
	if identity, ok := asIdentity(raw); ok {
		return Field{Name: name, Kind: FieldIdentity, Value: identity}
	}

	text, isString := asString(raw)
	if !isString {
		// Numbers, booleans, arrays and unrecognized objects print as the
		// server wrote them.
		return Field{Name: name, Kind: FieldPlain, Value: string(raw)}
	}
	switch {
	case htmlFields[strings.ToLower(name)]:
		return Field{Name: name, Kind: FieldHTML, Value: text}
	case isTimestamp(text):
		return Field{Name: name, Kind: FieldDateTime, Value: text}
	default:
		return Field{Name: name, Kind: FieldPlain, Value: text}
	}
}

func asIdentity(raw json.RawMessage) (Identity, bool) {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return Identity{}, false
	}
	if _, ok := keys["displayName"]; !ok {
		return Identity{}, false
	}
	var identity Identity
	if err := json.Unmarshal(raw, &identity); err != nil {
		return Identity{}, false
	}
	return identity, true
}

func asString(raw json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

func isTimestamp(s string) bool {
	_, err := time.Parse(time.RFC3339, s)
	return err == nil
}

// order puts the fields in the order the caller asked for them, since TFS
// answers a --fields request in its own order. Fields that were not requested
// — every field, when the caller asked for none — keep the response order and
// follow the requested ones.
func order(fields []Field, requested []string) []Field {
	if len(requested) == 0 || len(fields) == 0 {
		return fields
	}

	rank := make(map[string]int, len(requested))
	for i, name := range requested {
		key := strings.ToLower(strings.TrimSpace(name))
		if _, seen := rank[key]; !seen {
			rank[key] = i
		}
	}
	rankOf := func(f Field) int {
		if i, ok := rank[strings.ToLower(f.Name)]; ok {
			return i
		}
		return len(requested)
	}

	slices.SortStableFunc(fields, func(a, b Field) int {
		return cmp.Compare(rankOf(a), rankOf(b))
	})
	return fields
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
