package shell

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpandPath(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("test doesn't work on windows")
	}

	for _, in := range []string{"~", "~/", "foo.log"} {
		actual, err := ExpandPath(in)
		require.NoError(t, err)
		require.Truef(t, filepath.IsAbs(actual), "expanded path is not absolute: %s", actual)
	}
}
