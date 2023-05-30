package shell

import (
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

/*
Expands a path by
1. Replacing '~' by the user's home directory (*nix)
2. If the path is relative, make it absolute by using the working directory
*/
func ExpandPath(path string) (string, error) {
	if runtime.GOOS == "windows" {
		return path, nil
	}

	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	dir := usr.HomeDir

	path = strings.Replace(path, "~", dir, -1)
	if !filepath.IsAbs(path) {
		return filepath.Abs(path)
	}

	return path, nil
}
