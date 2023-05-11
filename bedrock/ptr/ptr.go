package ptr

func Bool(val bool) *bool {
	return &val
}

func False() *bool {
	b := false
	return &b
}

func Of[T any](val T) *T {
	return &val
}

func String(val string) *string {
	return &val
}

func True() *bool {
	b := true
	return &b
}

func Uint64(val uint64) *uint64 {
	return &val
}
