package persistence

import "errors"

var ErrAssemblyFileTooNew = errors.New("assembly file is newer than the program version")
