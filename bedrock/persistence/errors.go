package persistence

import "errors"

var ErrAssemblyFileTooNew = errors.New("assembly file is newer than the program version")
var ErrReceiptRequiresLineItems = errors.New("a receipt requires at least 1 line item")
var ErrTotalIsZero = errors.New("total is zero")
