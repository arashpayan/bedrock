package model

type Money int64

func (m Money) IsNegative() bool {
	return m < 0
}

func (m Money) IsPositive() bool {
	return m > 0
}
