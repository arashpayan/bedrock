package email

import (
	"encoding/json"
	"io"
)

type Email string

func (e Email) MarshalGQL(w io.Writer) {
	buf, _ := json.Marshal(string(e))
	w.Write(buf)
}

func (e Email) Ptr() *Email {
	return &e
}

func (e *Email) UnmarshalGQL(v interface{}) error {
	var val Email
	switch t := v.(type) {
	case string:
		val = Email(t)
	case []byte:
		val = Email(t)
	}

	if err := Validate(string(val)); err != nil {
		return err
	}

	*e = val
	return nil
}
