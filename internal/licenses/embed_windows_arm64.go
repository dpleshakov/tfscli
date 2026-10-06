package licenses

import "embed"

// platformFS holds the third-party licenses of windows/arm64, written into
// embed/windows-arm64/third-party by the release build.
//
//go:embed embed/windows-arm64
var platformFS embed.FS

// platformDir is the directory of platformFS that holds the set.
const platformDir = "embed/windows-arm64"
