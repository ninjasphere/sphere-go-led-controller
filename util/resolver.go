package util

var imageDir = "images/"

// resolves a relative image path to a fully qialified path.
func ResolveImagePath(relativePath string) string {
	return imageDir + "/" + relativePath
}

// the directory used to resolve the
func SetImageDir(newDir string) {
	imageDir = newDir
}
