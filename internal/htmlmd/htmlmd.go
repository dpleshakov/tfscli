package htmlmd

import (
	"fmt"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// Convert renders an HTML fragment as markdown using the library defaults.
// TFS-specific rules are registered here once real HTML samples have been
// collected; until then the conversion is unconfigured on purpose.
func Convert(html string) (string, error) {
	md, err := htmltomarkdown.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("convert html to markdown: %w", err)
	}
	return md, nil
}
