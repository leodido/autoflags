package values

import (
	"strconv"
	"time"

	"github.com/spf13/pflag"
)

// stringValue implements pflag.Value for a string.
type stringValue struct {
	s *string
}

func NewString(p *string) *stringValue {
	return &stringValue{s: p}
}

func (s *stringValue) String() string {
	return *s.s
}

func (s *stringValue) Set(val string) error {
	*s.s = val
	return nil
}

func (s *stringValue) Type() string {
	return "string"
}

var _ pflag.Value = (*stringValue)(nil)

// intValue implements pflag.Value for an int.
type intValue struct {
	i *int
}

func NewInt(p *int) *intValue {
	return &intValue{i: p}
}

func (i *intValue) String() string {
	return strconv.Itoa(*i.i)
}

func (i *intValue) Set(val string) error {
	v, err := strconv.Atoi(val)
	if err != nil {
		return err
	}
	*i.i = v
	return nil
}

func (i *intValue) Type() string {
	return "int"
}

var _ pflag.Value = (*intValue)(nil)

// durationValue implements pflag.Value for a time.Duration.
type durationValue struct {
	d *time.Duration
}

// NewDuration creates a new durationValue.
func NewDuration(val time.Duration, p *time.Duration) *durationValue {
	*p = val // Set the initial value
	return &durationValue{d: p}
}

func (d *durationValue) Set(s string) error {
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	// Only assign on success
	*d.d = v

	return nil
}

func (d *durationValue) Type() string {
	return "duration"
}

func (d *durationValue) String() string {
	return d.d.String()
}

var _ pflag.Value = (*durationValue)(nil)
