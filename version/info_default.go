//go:build !go1.18
// +build !go1.18

package version

func getRevision() string {
	return Revision
}

func getTags() string {
	return "unknown" // Not available prior to Go 1.18
}
