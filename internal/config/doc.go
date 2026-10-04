// Package config resolves the credential — the server URL, collection, and PAT, from
// $XDG_DATA_HOME/tfscli/auth.json or TFSCLI_AUTH — and loads the optional
// $XDG_CONFIG_HOME/tfscli/config.json, applies environment (TFSCLI_*) and flag
// overrides, and validates that required fields are set for the command being
// run. It never logs the PAT.
package config
