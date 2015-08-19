package nv

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

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
	UINT64_ARRAY
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

type List struct {
	header
	Pairs []pair
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
		l.Pairs = append(l.Pairs, p)

		var end uint64
		err := binary.Read(r, binary.BigEndian, &end)
		if err != nil {
			return List{}, err
		}
		if end == 0 {
			break
		}
		_, err = r.Seek(-8, 1)
		if err != nil {
			return List{}, err
		}
	}
	return List(l), nil
}
