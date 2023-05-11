package datetime

import (
	"encoding/json"
	"testing"

	"ara.sh/iabdaccounting/bedrock/ptr"
	"github.com/stretchr/testify/require"
)

func TestDateTimeUnmarshalGQL(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name        string
		input       any
		expected    DateTime
		errContains *string
	}{
		{
			name:     "int64",
			input:    int64(12098341),
			expected: DateTime(12098341),
		},
		{
			name:     "int",
			input:    int(12098341),
			expected: DateTime(12098341),
		},
		{
			name:     "uint64",
			input:    uint64(12098341),
			expected: DateTime(12098341),
		},
		{
			name:     "json.Number",
			input:    json.Number("12098341"),
			expected: DateTime(12098341),
		},
		{
			name:        "unacceptable type",
			input:       "12098341",
			errContains: ptr.Of("DateTime.UnmarshalGQL received a"),
		},
		{
			name:        "floating point json.Number",
			input:       json.Number("1219873.4813"),
			errContains: ptr.Of("unable to convert to int64"),
		},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			dt := new(DateTime)
			err := dt.UnmarshalGQL(td.input)
			if td.errContains != nil {
				require.ErrorContains(t, err, *td.errContains)
			} else {
				require.NoError(t, err)
				require.Equal(t, td.expected, *dt)
			}
		})
	}
}
