package workitem

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// APIClient is the part of the API client this package uses. Post is absent
// because Get Work Item is a GET; it joins the interface when the first POST
// endpoint (batch get, WIQL) arrives.
type APIClient interface {
	Get(ctx context.Context, path string, query url.Values) ([]byte, error)
}

// FieldKind tells the printer how to render a field value.
type FieldKind string

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
// value itself for the rest: identity fields are recognised by their shape and
// dates by being parseable, so custom fields are covered without listing them.
func classify(name string, raw json.RawMessage) Field {
	if identity, ok := asIdentity(raw); ok {
		return Field{Name: name, Kind: FieldIdentity, Value: identity}
	}

	text, isString := asString(raw)
	if !isString {
		// Numbers, booleans, arrays and unrecognised objects print as the
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
