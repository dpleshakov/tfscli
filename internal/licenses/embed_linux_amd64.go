package licenses

import "embed"

// platformFS holds the third-party licenses of linux/amd64, written into
// embed/linux-amd64/third-party by the release build.
//
//go:embed embed/linux-amd64
var platformFS embed.FS

// platformDir is the directory of platformFS that holds the set.
const platformDir = "embed/linux-amd64"
