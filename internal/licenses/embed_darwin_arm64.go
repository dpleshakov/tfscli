package licenses

import "embed"

// platformFS holds the third-party licenses of darwin/arm64, written into
// embed/darwin-arm64/third-party by the release build.
//
//go:embed embed/darwin-arm64
var platformFS embed.FS

// platformDir is the directory of platformFS that holds the set.
const platformDir = "embed/darwin-arm64"
