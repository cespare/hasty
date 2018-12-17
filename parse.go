// Package hasty provides functions that are useful for quick'n'dirty coding
// (small "script" programs, puzzle solving, etc).
//
// Not for use in production code.
package hasty

import (
	"encoding"
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"sync"
)

// MustParse is like Parse except that it panics
// instead of returning a non-nil error.
func MustParse(data []byte, v interface{}, r *regexp.Regexp) {
	if err := Parse(data, v, r); err != nil {
		panic(err)
	}
}

// Parse uses r to parse data and loads it into v.
//
// The target v must be a pointer to a value of struct type. The exported struct
// fields correspond to named capture groups of r. The following types are
// supported:
//
// - If the field implements encoding.TextUnmarshaler, then that is used.
//
// FIXME: finish documenting
func Parse(data []byte, v interface{}, r *regexp.Regexp) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return errors.New("hasty: Parse requires pointer target")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return errors.New("hasty: Parse target must be a pointer to struct value")
	}
	rt := rv.Type()

	var p *parser
	if v, ok := parserCache.Load(rt); ok {
		p = v.(*parser)
	} else {
		var err error
		p, err = newParser(rt)
		if err != nil {
			return err
		}
		parserCache.LoadOrStore(rt, p)
	}

	return p.parse(data, rv, r)
}

var parserCache sync.Map

// FIXME: document
var ErrNoMatch = errors.New("hasty: provided regular expression did not match data")

type parser struct {
	byName map[string]*fieldParser
}

type fieldParser struct {
	i    int
	utyp unmarshalerType
	typ  reflect.Type
}

type unmarshalerType int

const (
	notUnmarshaler unmarshalerType = iota
	unmarshalerVal
	unmarshalerPtr
)

var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

func newParser(rt reflect.Type) (*parser, error) {
	p := &parser{byName: make(map[string]*fieldParser)}
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			// Field isn't exported.
			continue
		}
		fp := &fieldParser{i: i}
		p.byName[field.Name] = fp
		if field.Type.Implements(textUnmarshalerType) {
			fp.utyp = unmarshalerVal
			continue
		}
		if reflect.PtrTo(field.Type).Implements(textUnmarshalerType) {
			fp.utyp = unmarshalerPtr
			continue
		}
		switch field.Type.Kind() {
		case reflect.Slice:
			if field.Type.Elem().Kind() != reflect.Uint8 {
				return nil, fmt.Errorf("hasty: unsupported type: %s", field.Type)
			}
		case reflect.String,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		default:
			return nil, fmt.Errorf("hasty: unsupported type: %s", field.Type)
		}
		fp.typ = field.Type
	}
	return p, nil
}

func (p *parser) parse(data []byte, rv reflect.Value, r *regexp.Regexp) error {
	matches := r.FindSubmatch(data)
	if matches == nil {
		return ErrNoMatch
	}
	names := r.SubexpNames()
	for i := 1; i < len(matches); i++ {
		match := matches[i]
		name := names[i]
		fp, ok := p.byName[name]
		if !ok {
			return fmt.Errorf("hasty: no target field for capture group %q", name)
		}
		field := rv.Field(fp.i)
		if fp.utyp != notUnmarshaler {
			var tu encoding.TextUnmarshaler
			switch fp.utyp {
			case unmarshalerVal:
				tu = field.Interface().(encoding.TextUnmarshaler)
			case unmarshalerPtr:
				tu = field.Addr().Interface().(encoding.TextUnmarshaler)
			}
			if err := tu.UnmarshalText(match); err != nil {
				return fmt.Errorf("hasty: %s.UnmarshalText returned error: %s", name, err)
			}
			continue
		}
		switch fp.typ.Kind() {
		case reflect.Slice:
			field.SetBytes(append([]byte(nil), match...))
		case reflect.String:
			field.SetString(string(match))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(string(match), 10, 64)
			if err != nil {
				return fmt.Errorf("hasty: cannot parse %q as integer", match)
			}
			minMax := minMaxIntVals[fp.typ.Bits()]
			if n < minMax[0] || n > minMax[1] {
				return fmt.Errorf("hasty: cannot represent %d as %s", n, fp.typ)
			}
			field.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := strconv.ParseUint(string(match), 10, 64)
			if err != nil {
				return fmt.Errorf("hasty: cannot parse %q as unsigned integer", match)
			}
			max := maxUintVals[fp.typ.Bits()]
			if n > max {
				return fmt.Errorf("hasty: cannot represent %d as %s", n, fp.typ)
			}
			field.SetUint(n)
		}
	}
	return nil
}

var minMaxIntVals = map[int][2]int64{
	8:  {math.MinInt8, math.MaxInt8},
	16: {math.MinInt16, math.MaxInt16},
	32: {math.MinInt32, math.MaxInt32},
	64: {math.MinInt64, math.MaxInt64},
}

var maxUintVals = map[int]uint64{
	8:  math.MaxUint8,
	16: math.MaxUint16,
	32: math.MaxUint32,
	64: math.MaxUint64,
}
