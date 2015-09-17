package nv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"

	xdr "github.com/davecgh/go-xdr/xdr2"
)

//go:generate stringer -type=flag
type flag uint32

const (
	_ flag = iota
	UNIQUE_NAME
	UNIQUE_NAME_TYPE
)

//go:generate stringer -type=dataType
type dataType uint32

const (
	UNKNOWN dataType = iota
	BOOLEAN
	BYTE
	INT16
	UINT16
	INT32
	UINT32
	INT64
	UINT64
	STRING
	BYTE_ARRAY
	INT16_ARRAY
	UINT16_ARRAY
	INT32_ARRAY
	UINT32_ARRAY
	INT64_ARRAY
	UINT64_ARRAY // 0x10
	STRING_ARRAY
	HRTIME
	NVLIST
	NVLIST_ARRAY
	BOOLEAN_VALUE
	INT8
	UINT8
	BOOLEAN_ARRAY
	INT8_ARRAY
	UINT8_ARRAY
	DOUBLE
)

var goToNV = map[string]dataType{
	"bool":          BOOLEAN_VALUE,
	"boolean_array": BOOLEAN_ARRAY,
	"byte":          BYTE,
	"byte_array":    BYTE_ARRAY,
	"float64":       DOUBLE,
	"int16":         INT16,
	"int16_array":   INT16_ARRAY,
	"int32":         INT32,
	"int32_array":   INT32_ARRAY,
	"int64":         INT64,
	"int64_array":   INT64_ARRAY,
	"int8":          INT8,
	"int8_array":    INT8_ARRAY,
	"nv.List":       NVLIST,
	"nvlist_array":  NVLIST_ARRAY,
	"string":        STRING,
	"string_array":  STRING_ARRAY,
	"time.Time":     HRTIME,
	"uint16":        UINT16,
	"uint16_array":  UINT16_ARRAY,
	"uint32":        UINT32,
	"uint32_array":  UINT32_ARRAY,
	"uint64":        UINT64,
	"uint64_array":  UINT64_ARRAY,
	"uint8":         UINT8,
	"uint8_array":   UINT8_ARRAY,
}

type List struct {
	header
	Pairs []Pair
}

type encoding struct {
	Encoding  uint8
	Endianess uint8
	Reserved1 uint8
	Reserved2 uint8
}

type header struct {
	Version uint32
	Flag    flag
}

type pair struct {
	EncodedSize uint32
	DecodedSize uint32
	Name        string
	Type        dataType
	NElements   uint32
	data        interface{}
}

type Pair struct {
	pair
	Value interface{}
}

func deref(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

func Encode(i interface{}) ([]byte, error) {
	if i == nil {
		return nil, errors.New("can not encode a nil pointer")
	}

	v := reflect.ValueOf(i)
	if !v.IsValid() {
		return nil, fmt.Errorf("type '%s' is invalid", v.Kind().String())
	}
	v = deref(v)
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("invalid type '%s', must be a struct", v.Kind().String())
	}

	buff := bytes.NewBuffer(nil)

	var err error
	if err = binary.Write(buff, binary.BigEndian, encoding{Encoding: 1, Endianess: 1}); err != nil {
		return nil, err
	}

	if err = encodeList(v, buff); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

func encodeList(v reflect.Value, w io.Writer) error {
	if !v.IsValid() {
		return errors.New("v is invalid")
	}

	if v.Type().String() != "nv.List" {
		return fmt.Errorf("invalid type '%s' expected 'nv.List'", v.Type().String())
	}
	pairs := v.FieldByName("Pairs")
	numPairs := pairs.Len()

	var err error
	if err = binary.Write(w, binary.BigEndian, header{Flag: UNIQUE_NAME}); err != nil {
		return err
	}

	enc := xdr.NewEncoder(w)
	for i := 0; i < numPairs; i++ {
		p := pairs.Index(i)
		pp := p.Interface().(Pair)
		pp.NElements = 1

		if pp.Type == UNKNOWN || pp.Type > DOUBLE {
			return fmt.Errorf("invalid Type '%v'", pp.Type)
		}
		switch pp.Type {
		case BYTE:
			pp.Value = int8(pp.Value.(uint8))
		case UINT8:
			pp.Value = int(int8(pp.Value.(uint8)))
		case BYTE_ARRAY:
			pp.NElements = uint32(len(pp.Value.([]byte)))
			n := int(pp.NElements)
			arrType := reflect.ArrayOf(n, reflect.TypeOf(byte(0)))
			arr := reflect.New(arrType).Elem()
			for i, b := range pp.Value.([]byte) {
				arr.Index(i).SetUint(uint64(b))
			}
			pp.Value = arr.Interface()
		case BOOLEAN_ARRAY:
			pp.NElements = uint32(len(pp.Value.([]bool)))
		case INT8_ARRAY:
			pp.NElements = uint32(len(pp.Value.([]int8)))
		case INT16_ARRAY:
			pp.NElements = uint32(len(pp.Value.([]int16)))
		case INT32_ARRAY:
			pp.NElements = uint32(len(pp.Value.([]int32)))
		case INT64_ARRAY:
			pp.NElements = uint32(len(pp.Value.([]int64)))
		case UINT8_ARRAY:
			// this one is weird since UINT8s are encoded as char
			// aka int32s... :(
			pp.NElements = uint32(len(pp.Value.([]uint8)))
			n := int(pp.NElements)
			sliceType := reflect.SliceOf(reflect.TypeOf(int32(0)))
			slice := reflect.MakeSlice(sliceType, n, n)
			for i, b := range pp.Value.([]uint8) {
				slice.Index(i).SetInt(int64(int8(b)))
			}
			pp.Value = slice.Interface()
		case UINT16_ARRAY:
			pp.NElements = uint32(len(pp.Value.([]uint16)))
		case UINT32_ARRAY:
			pp.NElements = uint32(len(pp.Value.([]uint32)))
		case UINT64_ARRAY:
			pp.NElements = uint32(len(pp.Value.([]uint64)))
		case STRING_ARRAY:
			pp.NElements = uint32(len(pp.Value.([]string)))
			arrType := reflect.ArrayOf(int(pp.NElements), reflect.TypeOf(""))
			arr := reflect.New(arrType).Elem()
			for i, b := range pp.Value.([]string) {
				arr.Index(i).SetString(b)
			}
			pp.Value = arr.Interface()
		case NVLIST:
			if _, err = enc.Encode(pp.pair); err != nil {
				return err
			}
			if err = encodeList(reflect.ValueOf(pp.Value), w); err != nil {
				return err
			}
			continue
		case NVLIST_ARRAY:
			pp.NElements = uint32(len(pp.Value.([]List)))
			if _, err = enc.Encode(pp.pair); err != nil {
				return err
			}
			if pp.NElements == 0 {
				return fmt.Errorf("empty NVLIST_ARRAY")
			}
			for _, l := range pp.Value.([]List) {
				if err = encodeList(reflect.ValueOf(l), w); err != nil {
					return err
				}
			}
			continue
		}

		_, err = enc.Encode(pp)
		if err != nil {
			return err
		}
	}

	return binary.Write(w, binary.BigEndian, uint64(0))
}

func Decode(buf []byte) (List, error) {
	b := bytes.NewReader(buf)

	enc := encoding{}
	err := binary.Read(b, binary.BigEndian, &enc)
	if err != nil {
		return List{}, err
	}

	if enc.Encoding > 1 {
		return List{}, fmt.Errorf("invalid encoding: %v", enc.Encoding)
	}
	if enc.Endianess > 1 {
		return List{}, fmt.Errorf("invalid endianess: %v", enc.Endianess)
	}
	if enc.Reserved1 != 0 {
		return List{}, fmt.Errorf("unexpected reserved1 value: %v", enc.Reserved1)
	}
	if enc.Reserved2 != 0 {
		return List{}, fmt.Errorf("unexpected reserved2 value: %v", enc.Reserved2)
	}

	return decodeList(b)
}

func isEnd(r io.ReadSeeker) (bool, error) {
	var end uint64
	err := binary.Read(r, binary.BigEndian, &end)
	if err != nil {
		return false, err
	}
	if end == 0 {
		return true, nil
	}
	_, err = r.Seek(-8, 1)
	return false, err
}

func decodeList(r io.ReadSeeker) (List, error) {
	l := List{}
	err := binary.Read(r, binary.BigEndian, &l.header)
	if err != nil {
		return List{}, err
	}

	if l.Version != 0 {
		return List{}, fmt.Errorf("unexpected version: %v", l.Version)
	}
	if l.Flag < UNIQUE_NAME || l.Flag > UNIQUE_NAME_TYPE {
		return List{}, fmt.Errorf("unexpected Flag: %v", l.Flag)
	}

	for {
		end, err := isEnd(r)
		if err != nil {
			return List{}, err
		}
		if end {
			break
		}

		p := pair{}
		_, err = xdr.Unmarshal(r, &p)
		if err != nil {
			return List{}, err
		}

		var v interface{}
		dec := newDecoder(r)
		switch p.Type {
		case BOOLEAN_VALUE:
			v, err = dec.DecodeBool()
		case BYTE:
			v, err = dec.DecodeByte()
		case INT8:
			v, err = dec.DecodeInt8()
		case INT16:
			v, err = dec.DecodeInt16()
		case INT32:
			v, err = dec.DecodeInt32()
		case INT64:
			v, err = dec.DecodeInt64()
		case UINT8:
			v, err = dec.DecodeUint8()
		case UINT16:
			v, err = dec.DecodeUint16()
		case UINT32:
			v, err = dec.DecodeUint32()
		case UINT64:
			v, err = dec.DecodeUint64()
		case HRTIME:
			v, err = dec.DecodeHRTime()
		case DOUBLE:
			v, err = dec.DecodeFloat64()
		case BOOLEAN_ARRAY:
			v, err = dec.DecodeBoolArray()
		case BYTE_ARRAY:
			if _, err = r.Seek(-4, 1); err == nil {
				v, err = dec.DecodeByteArray()
			}
		case INT8_ARRAY:
			v, err = dec.DecodeInt8Array()
		case INT16_ARRAY:
			v, err = dec.DecodeInt16Array()
		case INT32_ARRAY:
			v, err = dec.DecodeInt32Array()
		case INT64_ARRAY:
			v, err = dec.DecodeInt64Array()
		case UINT8_ARRAY:
			v, err = dec.DecodeUint8Array()
		case UINT16_ARRAY:
			v, err = dec.DecodeUint16Array()
		case UINT32_ARRAY:
			v, err = dec.DecodeUint32Array()
		case UINT64_ARRAY:
			v, err = dec.DecodeUint64Array()
		case STRING:
			v, err = dec.DecodeString()
		case STRING_ARRAY:
			if _, err = r.Seek(-4, 1); err == nil {
				v, err = dec.DecodeStringArray()
			}
		case NVLIST:
			v, err = decodeList(r)
		case NVLIST_ARRAY:
			arr := make([]List, 0, p.NElements)
			for i := uint32(0); i < p.NElements; i++ {
				var list List
				list, err = decodeList(r)
				if err != nil {
					break
				}
				arr = append(arr, list)
			}
			v = arr
		default:
			return List{}, fmt.Errorf("unknown type: %v", p.Type)
		}
		if err != nil {
			return List{}, err
		}

		p.data = v
		pp := Pair{pair: p, Value: v}
		l.Pairs = append(l.Pairs, pp)

	}
	return List(l), nil
}
