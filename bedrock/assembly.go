package bedrock

import (
	"ara.sh/iabdaccounting/bedrock/persistence"
)

type Assembly struct {
	db persistence.Database
}

func OpenAssembly(path string) (*Assembly, error) {
	db, err := persistence.Open(path)
	if err != nil {
		return nil, err
	}
	return &Assembly{db: *db}, nil
}

// func (a *Assembly) CreateAccount(in)
