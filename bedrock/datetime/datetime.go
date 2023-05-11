package datetime

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type DateTime int64

func Now() DateTime {
	return DateTime(time.Now().UnixMilli())
}

func FromTime(t time.Time) DateTime {
	return DateTime(t.UnixMilli())
}

func (dt DateTime) MarshalGQL(w io.Writer) {
	fmt.Fprintf(w, "%d", dt)
}

func (dt DateTime) MarshalJSON() ([]byte, error) {
	str := fmt.Sprintf("%d", dt)
	return []byte(str), nil
}

func (dt DateTime) Ptr() *DateTime {
	return &dt
}

func (dt DateTime) Time() time.Time {
	seconds := int64(dt / 1000)
	remainderMs := int64(dt % 1000)

	return time.Unix(seconds, remainderMs*1e6)
}

func (dt *DateTime) UnmarshalGQL(v any) error {
	var num int64
	switch t := v.(type) {
	case int:
		num = int64(t)
	case int64:
		num = t
	case json.Number:
		var err error
		num, err = t.Int64()
		if err != nil {
			return fmt.Errorf("DateTime.UnmarshalGQL unable to convert to int64: %w", err)
		}
	case uint64:
		num = int64(t)
	default:
		return fmt.Errorf("DateTime.UnmarshalGQL received a %T", v)
	}

	*dt = DateTime(num)
	return nil
}
