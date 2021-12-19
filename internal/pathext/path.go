package pathext

func IsRoot(path string) bool {
	return path == "" || path == "." || path == "/" || path == "./"
}
