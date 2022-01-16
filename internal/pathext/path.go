package pathext

import "strings"

func IsRoot(path string, trim bool) bool {
	if trim && len(strings.TrimSpace(path)) == 0 {
		return true
	}

	return path == "" || path == "." || path == "/" || path == "./"
}
