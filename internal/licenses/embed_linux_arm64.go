package licenses

import "embed"

// platformFS holds the third-party licenses of linux/arm64, written into
// embed/linux-arm64/third-party by the release build.
//
//go:embed embed/linux-arm64
var platformFS embed.FS

// platformDir is the directory of platformFS that holds the set.
const platformDir = "embed/linux-arm64"
