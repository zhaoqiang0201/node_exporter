//go:build go1.18
// +build go1.18

package version

import "runtime/debug"

var computedRevision string
var computedTags string

func getRevision() string {
	if Revision != "" {
		return Revision
	}
	return computedRevision
}

func getTags() string {
	return computedTags
}

func init() {
	computedRevision, computedTags = computeRevision()
}

func computeRevision() (string, string) {
	var (
		rev      = "unknown"
		tags     = "unknown"
		modified bool
	)
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return rev, tags
	}
	for _, v := range buildInfo.Settings {
		if v.Key == "vcs.revision" {
			rev = v.Value
		}
		if v.Key == "vcs.modified" {
			if v.Value == "true" {
				modified = true
			}
		}
		if v.Key == "-tags" {
			tags = v.Value
		}
	}
	if modified {
		return rev + "-modified", tags
	}
	return rev, tags
}
