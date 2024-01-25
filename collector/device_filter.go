package collector

import (
	"regexp"
)

type deviceFilter struct {
	ignorePattern *regexp.Regexp
	acceptPattern *regexp.Regexp
}

func newDeviceFilter(ignoredPattern, acceptPattern string) (f deviceFilter) {
	if ignoredPattern != "" {
		f.ignorePattern = regexp.MustCompile(ignoredPattern)
	}

	if acceptPattern != "" {
		f.acceptPattern = regexp.MustCompile(acceptPattern)
	}

	return
}

// ignored returns whether the device should be ignored
func (f *deviceFilter) ignored(name string) bool {
	return (f.ignorePattern != nil && f.ignorePattern.MatchString(name)) ||
		(f.acceptPattern != nil && !f.acceptPattern.MatchString(name))
}
