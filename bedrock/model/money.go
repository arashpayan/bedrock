package model

import (
	"errors"
	"math/big"
	"strings"
)

type Money int64

func (m Money) IsNegative() bool {
	return m < 0
}

func (m Money) IsPositive() bool {
	return m > 0
}

func NewMoney(s string) (Money, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	f, _, err := big.ParseFloat(s, 10, 128, big.AwayFromZero)
	if err != nil {
		return 0, err
	}
	f = f.Mul(f, big.NewFloat(100))
	parsed := f.String()
	if strings.Contains(parsed, ".") {
		return 0, errors.New("too many decimals")
	}
	val, _ := f.Int64()
	return Money(val), nil
}
