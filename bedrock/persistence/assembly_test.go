package persistence

import (
	"context"
	"testing"
	"time"

	"ara.sh/iabdaccounting/bedrock/model"
	"github.com/stretchr/testify/require"
)

func TestSetAssemblyInfo(t *testing.T) {
	t.Parallel()

	db := newDB(t)
	golden := model.Assembly{
		Name:     "Bahá'ís of Thousand Oaks",
		Timezone: *time.Local,
	}
	err := db.SetAssemblyInfo(context.Background(), golden)
	require.NoError(t, err)

	actual, err := db.AssemblyInfo(context.Background())
	require.NoError(t, err)
	require.NotNil(t, actual)
	require.Equal(t, golden, *actual)
}
