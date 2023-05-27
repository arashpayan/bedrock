package persistence

import (
	"context"
	"testing"

	"ara.sh/iabdaccounting/bedrock/model"
	"ara.sh/iabdaccounting/bedrock/ptr"
	"github.com/stretchr/testify/require"
)

func TestInsertParty(t *testing.T) {
	t.Parallel()

	db := newDB(t)
	in := model.CreatePartyInput{
		Name:            "Scooby Doo",
		EmailAddress:    ptr.Of("scooby.doo@gmail.com"),
		BahaiIDNumber:   ptr.Of("191919"),
		Address:         ptr.Of("123 Main St."),
		TelephoneNumber: ptr.Of("18055551212"),
	}
	actual, err := db.CreateParty(context.Background(), in)
	require.NoError(t, err)
	require.NotNil(t, actual)
	assertValidBase(t, actual.Base)
}
