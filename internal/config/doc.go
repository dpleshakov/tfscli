// Package config loads ~/.tfscli/config.json, applies environment
// (TFSCLI_*) and flag overrides, and validates that required fields are
// set for the command being run. It never logs the PAT.
package config
