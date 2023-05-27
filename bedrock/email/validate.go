package email

import (
	"errors"
	"strings"
)

func Validate(str string) error {
	if len(str) > 254 {
		return errors.New("too long")
	}
	parts := strings.Split(str, "@")
	if len(parts) != 2 {
		return errors.New("address doesn't have a user and domain separated by an '@'")
	}
	if parts[0] == "" {
		return errors.New("invalid local component")
	}
	domainParts := strings.Split(parts[1], ".")
	if len(domainParts) < 2 {
		return errors.New("invalid domain")
	}
	tld := domainParts[len(domainParts)-1]
	if len(tld) < 2 {
		return errors.New("invalid tld")
	}
	for _, dp := range domainParts {
		if len(dp) == 0 {
			return errors.New("invalid domain name")
		}
	}

	return nil
}
