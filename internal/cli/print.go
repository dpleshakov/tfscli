package cli

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dpleshakov/tfscli/internal/htmlmd"
	"github.com/dpleshakov/tfscli/internal/tfserr"
	"github.com/dpleshakov/tfscli/internal/workitem"
)

// printWorkItem writes wi as markdown: short values as "Name: value" lines,
// prose and anything else spanning several lines as a "## Name" section. Field
// order is the one workitem.Get established and is not rearranged here.
//
// The document is assembled in memory first, so that a field that fails to
// convert leaves nothing half-written on stdout.
func printWorkItem(w io.Writer, wi *workitem.WorkItem) error {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# Work item %d (rev %d)\n", wi.ID, wi.Rev)

	afterLine := false // the previous field printed as a "Name: value" line
	for _, f := range wi.Fields {
		text, err := render(f)
		if err != nil {
			return err
		}

		if isSection(f, text) {
			fmt.Fprintf(&buf, "\n## %s\n\n%s\n", f.Name, text)
			afterLine = false
			continue
		}
		if !afterLine {
			buf.WriteString("\n")
		}
		fmt.Fprintf(&buf, "%s\n", strings.TrimRight(f.Name+": "+text, " "))
		afterLine = true
	}

	_, err := w.Write(buf.Bytes())
	return err
}

// render turns one field value into the text to print, per its kind.
func render(f workitem.Field) (string, error) {
	switch f.Kind {
	case workitem.FieldHTML:
		html, ok := f.Value.(string)
		if !ok {
			return fmt.Sprint(f.Value), nil
		}
		md, err := htmlmd.Convert(html)
		if err != nil {
			return "", &tfserr.Error{
				Category: tfserr.Server,
				Message:  fmt.Sprintf("cannot convert field %s from HTML to markdown", f.Name),
				Cause:    err,
			}
		}
		return strings.TrimSpace(md), nil

	case workitem.FieldIdentity:
		identity, ok := f.Value.(workitem.Identity)
		if !ok {
			return fmt.Sprint(f.Value), nil
		}
		return formatIdentity(identity), nil

	case workitem.FieldDateTime:
		text, ok := f.Value.(string)
		if !ok {
			return fmt.Sprint(f.Value), nil
		}
		return formatTimestamp(text), nil

	default:
		return fmt.Sprint(f.Value), nil
	}
}

// isSection decides between an inline line and a heading with a block under
// it. HTML fields hold prose and always get a section; other fields get one
// once their value stops fitting on a single line.
func isSection(f workitem.Field, text string) bool {
	if text == "" {
		return false
	}
	return f.Kind == workitem.FieldHTML || strings.Contains(text, "\n")
}

// formatIdentity prints the two parts of an identity that identify a person.
// The image URLs, descriptors, and internal ids TFS sends alongside them are
// noise for a reader and are dropped by the workitem package already.
func formatIdentity(identity workitem.Identity) string {
	switch {
	case identity.DisplayName == "":
		return identity.UniqueName
	case identity.UniqueName == "":
		return identity.DisplayName
	default:
		return fmt.Sprintf("%s <%s>", identity.DisplayName, identity.UniqueName)
	}
}

// formatTimestamp drops sub-second precision and keeps the instant in UTC:
// rendering it in the local zone would make the same work item print
// differently depending on the machine that ran the command. A value that does
// not parse is printed unchanged rather than replaced by an error.
func formatTimestamp(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return t.UTC().Format(time.RFC3339)
}
