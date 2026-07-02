// Package apiclient wraps net/http for TFS REST API calls: URL building,
// PAT authorization, TLS configuration, request logging via a RoundTripper,
// and classification of HTTP outcomes into stable tfserr categories.
package apiclient
