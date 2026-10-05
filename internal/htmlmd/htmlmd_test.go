package htmlmd

import "testing"

// The expectations below record the library defaults, which is all v1 uses.
// They are a smoke check that the wrapper is wired to the converter — not an
// endorsement of any particular rendering. TFS-specific rules and golden tests
// on real samples arrive with the html-quirks entry in docs/backlog.md.
func TestConvert(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "emphasis",
			html: "<p>Text with <b>bold</b> and <i>italic</i>.</p>",
			want: "Text with **bold** and *italic*.",
		},
		{
			name: "unordered list",
			html: "<ul><li>first</li><li>second</li></ul>",
			want: "- first\n- second",
		},
		{
			name: "ordered list",
			html: "<ol><li>first</li><li>second</li></ol>",
			want: "1. first\n2. second",
		},
		{
			name: "link",
			html: `<p>See <a href="https://tfs.company.com/wi/12345">work item 12345</a>.</p>`,
			want: "See [work item 12345](https://tfs.company.com/wi/12345).",
		},
		{
			name: "paragraphs are separated by a blank line",
			html: "<p>Line one</p><p>Line two</p>",
			want: "Line one\n\nLine two",
		},
		{
			name: "code block",
			html: "<pre><code>go test ./...</code></pre>",
			want: "```\ngo test ./...\n```",
		},
		{
			name: "plain text passes through",
			html: "Nothing to convert here",
			want: "Nothing to convert here",
		},
		{
			name: "empty input",
			html: "",
			want: "",
		},
		{
			// TFS stores rich-text fields as fragments and does not guarantee
			// well-formed markup; the converter must not fail on them.
			name: "unclosed tag",
			html: "<p>Unclosed <b>bold",
			want: "Unclosed **bold**",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Convert(tt.html)
			if err != nil {
				t.Fatalf("Convert(%q) returned error: %v", tt.html, err)
			}
			if got != tt.want {
				t.Errorf("Convert(%q) = %q, want %q", tt.html, got, tt.want)
			}
		})
	}
}
