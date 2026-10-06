package licenses

import "embed"

// platformFS holds the third-party licenses of windows/amd64, written into
// embed/windows-amd64/third-party by the release build.
//
//go:embed embed/windows-amd64
var platformFS embed.FS

// platformDir is the directory of platformFS that holds the set.
const platformDir = "embed/windows-amd64"
