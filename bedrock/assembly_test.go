package bedrock

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAssembly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "file.bedrock")
	ably, err := OpenAssembly(path)
	require.NoError(t, err)
	require.NotNil(t, ably)
}
