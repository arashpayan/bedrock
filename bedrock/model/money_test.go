package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMoney(t *testing.T) {
	t.Parallel()

	for i := 0; i < 100; i++ {
		in := fmt.Sprintf(".%02d", i)
		m, err := NewMoney(in)
		require.NoError(t, err)
		require.EqualValuesf(t, i, m, "input: %s", in)
	}
	for i := 0; i < 100; i++ {
		in := fmt.Sprintf("0.%02d", i)
		m, err := NewMoney(in)
		require.NoError(t, err)
		require.EqualValuesf(t, i, m, "input: %s", in)
	}
	for i := 0; i < 100; i++ {
		in := fmt.Sprintf("1.%02d", i)
		m, err := NewMoney(in)
		require.NoError(t, err)
		require.EqualValuesf(t, 100+i, m, "input: %s", in)
	}
	for i := 0; i < 100; i++ {
		in := fmt.Sprintf("3333.%02d", i)
		m, err := NewMoney(in)
		require.NoError(t, err)
		require.EqualValuesf(t, i+333300, m, "input: %s", in)
	}
}
