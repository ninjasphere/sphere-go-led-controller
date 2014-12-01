package util

// resolves a relative image path to a fully qialified path.
func ResolveImagePath(relativePath string) string {
	return "images/" + relativePath
}
