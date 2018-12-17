package hasty

import (
	"reflect"
	"regexp"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	type stringInt struct {
		S string
		x string
		I int
	}
	type ints struct {
		I   int
		I8  int8
		I16 int16
		I32 int32
		I64 int64
	}
	type uints struct {
		U   int
		U8  int8
		U16 int16
		U32 int32
		U64 int64
	}
	type myString string
	type stringBytes struct {
		S myString
		B []byte
	}
	type unmarshaler struct {
		S string
		T time.Time
	}
	for _, tt := range []struct {
		in   string
		v    interface{}
		r    string
		want interface{}
	}{
		{
			"[foo, 3]",
			new(stringInt),
			`^\[(?P<S>\w+), (?P<I>\d+)\]$`,
			&stringInt{S: "foo", I: 3},
		},
		{
			"-1, -2, -3, -4, -5",
			new(ints),
			`^(?P<I>-?\d+), (?P<I8>-?\d+), (?P<I16>-?\d+), (?P<I32>-?\d+), (?P<I64>-?\d+)$`,
			&ints{-1, -2, -3, -4, -5},
		},
		{
			"1, 2, 3, 4, 5",
			new(uints),
			`^(?P<U>\d+), (?P<U8>\d+), (?P<U16>\d+), (?P<U32>\d+), (?P<U64>\d+)$`,
			&uints{1, 2, 3, 4, 5},
		},
		{
			"hello world",
			new(stringBytes),
			`^(?P<S>\w+) (?P<B>\w+)$`,
			&stringBytes{"hello", []byte("world")},
		},
		{
			"date = 2018-12-15T00:00:00Z",
			new(unmarshaler),
			`^(?P<S>\w+)\s*=\s*(?P<T>.+)$`,
			&unmarshaler{"date", time.Date(2018, 12, 15, 0, 0, 0, 0, time.UTC)},
		},
		{
			"a b 1",
			new(stringInt),
			`^(?P<S>\w) (.) (?P<I>\d)$`,
			&stringInt{S: "a", I: 1},
		},
	} {
		r := regexp.MustCompile(tt.r)
		if err := Parse([]byte(tt.in), tt.v, r); err != nil {
			t.Errorf("Parse(%q, %T, %q): %s", tt.in, tt.v, tt.r, err)
			continue
		}
		if !reflect.DeepEqual(tt.v, tt.want) {
			t.Errorf("Parse(%q, %T, %q): got %#v; want %#v",
				tt.in, tt.v, tt.r, tt.v, tt.want)
		}
	}
}
