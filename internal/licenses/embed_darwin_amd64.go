package licenses

import "embed"

// platformFS holds the third-party licenses of darwin/amd64, written into
// embed/darwin-amd64/third-party by the release build.
//
//go:embed embed/darwin-amd64
var platformFS embed.FS

// platformDir is the directory of platformFS that holds the set.
const platformDir = "embed/darwin-amd64"
