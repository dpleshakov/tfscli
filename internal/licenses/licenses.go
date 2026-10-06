package licenses

import (
	_ "embed" // for go:embed of the Go license
	"io/fs"
	"path"
	"sort"
	"strings"
)

// goLicense is the license of the Go standard library and runtime, a copy of
// LICENSE from the Go distribution.
//
//go:embed go.LICENSE
var goLicense string

// goComponent names the Go standard library and runtime in the output.
const goComponent = "Go standard library and runtime"

// separator frames the name of each component above its license texts.
var separator = strings.Repeat("=", 80)

// Text returns the license information of this binary: the list of the
// components it includes, followed by the license texts of each.
func Text() string {
	set, err := fs.Sub(platformFS, path.Join(platformDir, "third-party"))
	if err != nil {
		set = nil
	}
	return text(goLicense, set)
}

// component is one entry of the output: a name and its license files.
type component struct {
	name  string
	files []string
}

// text assembles the output from the Go license and set, the third-party
// licenses saved by go-licenses: one directory per package path, holding the
// license files of its module. A nil or empty set means a build the release
// did not make.
func text(goLicense string, set fs.FS) string {
	third := thirdParty(set)

	var b strings.Builder
	b.WriteString("tfscli includes the following components:\n\n")
	b.WriteString("  " + goComponent + "\n")
	for _, c := range third {
		b.WriteString("  " + c.name + "\n")
	}
	if len(third) == 0 {
		b.WriteString("\nThis build does not carry the licenses of the third-party modules it\n" +
			"includes: they are embedded only in the release builds, available at\n" +
			"https://github.com/dpleshakov/tfscli/releases.\n")
	}

	writeComponent(&b, goComponent, goLicense)
	for _, c := range third {
		var texts []string
		for _, f := range c.files {
			data, err := fs.ReadFile(set, f)
			if err != nil {
				continue
			}
			texts = append(texts, string(data))
		}
		writeComponent(&b, c.name, strings.Join(texts, "\n"))
	}
	return b.String()
}

// thirdParty lists the components of set, sorted by name, with their files.
// A set that cannot be read yields none.
func thirdParty(set fs.FS) []component {
	if set == nil {
		return nil
	}
	byDir := map[string]*component{}
	err := fs.WalkDir(set, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		dir := path.Dir(p)
		if byDir[dir] == nil {
			byDir[dir] = &component{name: dir}
		}
		byDir[dir].files = append(byDir[dir].files, p)
		return nil
	})
	if err != nil {
		return nil
	}

	list := make([]component, 0, len(byDir))
	for _, c := range byDir {
		list = append(list, *c)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].name < list[j].name })
	return list
}

// writeComponent appends the framed name of a component and its license text.
func writeComponent(b *strings.Builder, name, license string) {
	b.WriteString("\n" + separator + "\n" + name + "\n" + separator + "\n\n")
	b.WriteString(strings.TrimRight(license, "\n") + "\n")
}
