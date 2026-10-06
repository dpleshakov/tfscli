//go:build !((linux || windows || darwin) && (amd64 || arm64))

package licenses

import "embed"

// platformFS is empty on a platform the release does not build: no set of
// third-party licenses is ever generated for it.
var platformFS embed.FS

// platformDir names no directory of the empty platformFS.
const platformDir = "embed/other"
