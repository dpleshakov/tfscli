package apiclient

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// resourceLocation is the part of an ApiResourceLocation, one element of the
// response to OPTIONS on _apis, that version selection reads.
type resourceLocation struct {
	ID string `json:"id"`
	// MaxVersion is the highest version the server supports for the
	// location, such as "7.1".
	MaxVersion string `json:"maxVersion"`
	// ReleasedVersion is the latest version that is not a preview; "0.0"
	// when the location has only preview versions.
	ReleasedVersion string `json:"releasedVersion"`
	// ResourceVersion is the revision of the preview at MaxVersion.
	ResourceVersion int `json:"resourceVersion"`
}

// chooseVersion picks the api-version for location from body, the response
// to OPTIONS on _apis: the released version of the location, or, when it has
// none, the latest preview. That is the version the server itself chooses for
// a request without one, so sending it does not change the response. On
// failure version is empty and reason says why, in words fit for a log line
// or an error message.
func chooseVersion(body []byte, location string) (version, reason string) {
	var payload struct {
		Value []resourceLocation `json:"value"`
	}
	// Some servers prefix JSON with a byte order mark.
	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))
	if err := json.Unmarshal(body, &payload); err != nil || payload.Value == nil {
		return "", "the server's list of API versions could not be read"
	}

	for _, loc := range payload.Value {
		if !strings.EqualFold(loc.ID, location) {
			continue
		}
		if isReleased(loc.ReleasedVersion) {
			return loc.ReleasedVersion, ""
		}
		if loc.MaxVersion == "" {
			return "", "the server lists no version for this resource"
		}
		version = loc.MaxVersion + "-preview"
		if loc.ResourceVersion > 0 {
			version += "." + strconv.Itoa(loc.ResourceVersion)
		}
		return version, ""
	}
	return "", "the server does not list this resource among its API versions"
}

// isReleased reports whether v names a released version. The server writes
// "0.0" for a location that has only preview versions.
func isReleased(v string) bool {
	return v != "" && strings.Trim(v, "0.") != ""
}
