package ui

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var dollarsRegEx = regexp.MustCompile(`^\$?\d*\.?\d{0,2}$`)

func DollarValidator(s string) error {
	if !dollarsRegEx.MatchString(s) {
		return errors.New("not a dollars amount")
	}
	return nil
}

func ParseDayDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	return time.Parse("2006-01-02", s)
}
