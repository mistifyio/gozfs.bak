package nv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

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

func align4(n int) int {
	return (n + 3) & ^3
}

func align8(n int) int {
	return (n + 7) & ^7
}

func (p pair) encodedSize() int {
	valSize := 0
	switch p.Type {
	case BYTE, INT8, UINT8, INT16, UINT16, BOOLEAN_VALUE, INT32, UINT32:
		valSize = 4
	case INT64, UINT64, HRTIME, DOUBLE:
		valSize = 8
	case BYTE_ARRAY:
		valSize = int(p.NElements * 1)
	case INT8_ARRAY, UINT8_ARRAY, INT16_ARRAY, UINT16_ARRAY, INT32_ARRAY, UINT32_ARRAY:
		valSize = 4 + int(p.NElements*4)
	case INT64_ARRAY, UINT64_ARRAY:
		valSize = 4 + int(p.NElements*8)
	case STRING:
		valSize = 4 + len(p.data.(string)) + 1
	case NVLIST:
		valSize = len(p.data.([]byte))
	case NVLIST_ARRAY:
		valSize = len(p.data.([]byte))
	case BOOLEAN_ARRAY:
		valSize = 4 + int(p.NElements*4)
	case STRING_ARRAY:
		slice := p.data.([]string)
		for i := range slice {
			valSize += align4(4 + len(slice[i]) + 1)
		}
	}
	return 4 + 4 + 4 + align4(len(p.Name)) + 4 + 4 + align4(valSize)
}

func (p pair) decodedSize() int {
	// typedef struct nvpair {
	// 	int32_t nvp_size;	/* size of this nvpair */
	// 	int16_t	nvp_name_sz;	/* length of name string */
	// 	int16_t	nvp_reserve;	/* not used */
	// 	int32_t	nvp_value_elem;	/* number of elements for array types */
	// 	data_type_t nvp_type;	/* type of value */
	// 	/* name string */
	// 	/* aligned ptr array for string arrays */
	// 	/* aligned array of data for value */
	// } nvpair_t;
	nvpair_tSize := 4 + 2 + 2 + 4 + 4 + len(p.Name) + 1

	valSize := 0
	switch p.Type {
	case BYTE, INT8, UINT8:
		valSize = 1
	case INT16, UINT16:
		valSize = 2
	case BOOLEAN_VALUE, INT32, UINT32:
		valSize = 4
	case INT64, UINT64, HRTIME, DOUBLE:
		valSize = 8
	case BYTE_ARRAY, INT8_ARRAY, UINT8_ARRAY:
		valSize = int(p.NElements * 1)
	case INT16_ARRAY, UINT16_ARRAY:
		valSize = int(p.NElements * 2)
	case INT32_ARRAY, UINT32_ARRAY:
		valSize = int(p.NElements * 4)
	case INT64_ARRAY, UINT64_ARRAY:
		valSize = int(p.NElements * 8)
	case STRING:
		valSize = len(p.data.(string)) + 1
	case NVLIST:
		// /* nvlist header */
		// typedef struct nvlist {
		// 	int32_t		nvl_version;
		// 	uint32_t	nvl_nvflag;	/* persistent flags */
		// 	uint64_t	nvl_priv;	/* ptr to private data if not packed */
		// 	uint32_t	nvl_flag;
		// 	int32_t		nvl_pad;	/* currently not used, for alignment */
		// } nvlist_t;
		valSize = 4 + 4 + 8 + 4 + 4
	case NVLIST_ARRAY:
		// value_sz = (uint64_t)nelem * sizeof (uint64_t) +
		//	      (uint64_t)nelem * NV_ALIGN(sizeof (nvlist_t));
		valSize = int(p.NElements) * (8 + align8(4+4+8+4+4))
	case BOOLEAN_ARRAY:
		valSize = 4 + int(p.NElements*4)
	case STRING_ARRAY:
		slice := p.data.([]string)
		for i := range slice {
			valSize += align8(4 + len(slice[i]) + 1)
		}
	}
	fmt.Fprintln(os.Stderr, "value size:", valSize)
	fmt.Fprintln(os.Stderr, "name size:", len(p.Name)+1)
	return align8(nvpair_tSize) + align8(valSize)
}

func deref(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

func writeHeader(w io.Writer) error {

	return nil
}

func Encode(i interface{}) ([]byte, error) {
	if i == nil {
		return nil, errors.New("can not encode a nil pointer")
	}

	v := reflect.ValueOf(i)
	if !v.IsValid() {
		return nil, fmt.Errorf("type '%s' is invalid", v.Kind().String())
	}

	var err error
	buff := bytes.NewBuffer(nil)
	if err = binary.Write(buff, binary.BigEndian, encoding{Encoding: 1, Endianess: 1}); err != nil {
		return nil, err
	}

	if err = encode(buff, v); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

func encode(w io.Writer, v reflect.Value) error {
	var err error
	if err = binary.Write(w, binary.BigEndian, header{Flag: UNIQUE_NAME}); err != nil {
		return err
	}

	v = deref(v)
	switch v.Kind() {
	case reflect.Struct:
		_, err = encodeStruct(v, w)
	case reflect.Map:
		keys := make([]string, len(v.MapKeys()))
		for i, k := range v.MapKeys() {
			keys[i] = k.Interface().(string)
		}
		sort.Strings(keys)
		fmt.Println("keys:", keys)

		for _, k := range keys {
			fmt.Printf("Encode: k: %v, v: %#v\n", k, v.MapIndex(reflect.ValueOf(k)))
			_, err = encodeItem(w, k, v.MapIndex(reflect.ValueOf(k)))
			if err != nil {
				break
			}
		}
		err = binary.Write(w, binary.BigEndian, uint64(0))
	default:
		return fmt.Errorf("invalid type '%s', must be a struct", v.Kind().String())
	}

	return err
}

func encodeStruct(v reflect.Value, w io.Writer) (int, error) {
	var err error
	size := 0
	numFields := v.NumField()
	fmt.Fprintf(os.Stderr, "v:%+v\n", v.Interface())
	fmt.Fprintln(os.Stderr, "numFields:", numFields)
	for i := 0; i < numFields; i++ {
		field := v.Field(i)
		fmt.Fprintln(os.Stderr, "field:", field)

		structField := v.Type().Field(i)
		name := structField.Name
		if tag := structField.Tag.Get("nv"); tag != "" {
			tags := strings.Split(tag, ",")
			if len(tags) > 0 && tags[0] != "" {
				name = tags[0]
			}
		}

		encodeItem(w, name, field)
	}

	if err = binary.Write(w, binary.BigEndian, uint64(0)); err != nil {
		return 0, err
	}
	return size + 8, nil
}

func encodeItem(w io.Writer, name string, field reflect.Value) ([]byte, error) {
	var err error
	fmt.Println("name:", name, field.CanAddr())
	if !field.CanAddr() {
		return nil, nil
	}
	p := pair{
		Name:      name,
		NElements: 1,
		data:      field.Interface(),
	}
	value := p.data
	size := 0

	fmt.Fprintf(os.Stderr, "encode: name, value: %v, %v\n", name, value)

	switch t := field.Kind(); t {
	case reflect.String:
		p.Type = STRING
	case reflect.Uint64:
		p.Type = UINT64
	case reflect.Int32:
		p.Type = INT32
	case reflect.Struct:
		p.Type = NVLIST
		size = 24
	default:
		panic(fmt.Sprint("unknown type:", t))
	}

	if field.Type().String() == "nv.mVal" {
		p.Type = field.FieldByName("Type").Interface().(dataType)
		p.Name = field.FieldByName("Name").Interface().(string)
		p.data = field.FieldByName("Value").Interface()
		value = p.data
	}

	if p.Type == UNKNOWN || p.Type > DOUBLE {
		return nil, fmt.Errorf("invalid Type '%v'", field.Kind())
	}

	vbuf := &bytes.Buffer{}
	fmt.Fprintln(os.Stderr, "encode: p.Type:", p.Type)
	switch p.Type {
	case BYTE:
		value = int8(value.(uint8))
	case UINT8:
		value = int(int8(value.(uint8)))
	case BYTE_ARRAY:
		p.NElements = uint32(len(value.([]byte)))
		n := int(p.NElements)
		arrType := reflect.ArrayOf(n, reflect.TypeOf(byte(0)))
		arr := reflect.New(arrType).Elem()
		for i, b := range value.([]byte) {
			arr.Index(i).SetUint(uint64(b))
		}
		value = arr.Interface()
	case BOOLEAN_ARRAY:
		p.NElements = uint32(len(value.([]bool)))
	case INT8_ARRAY:
		p.NElements = uint32(len(value.([]int8)))
	case INT16_ARRAY:
		p.NElements = uint32(len(value.([]int16)))
	case INT32_ARRAY:
		p.NElements = uint32(len(value.([]int32)))
	case INT64_ARRAY:
		p.NElements = uint32(len(value.([]int64)))
	case UINT8_ARRAY:
		// this one is weird since UINT8s are encoded as char
		// aka int32s... :(
		p.NElements = uint32(len(value.([]uint8)))
		n := int(p.NElements)
		sliceType := reflect.SliceOf(reflect.TypeOf(int32(0)))
		slice := reflect.MakeSlice(sliceType, n, n)
		for i, b := range value.([]uint8) {
			slice.Index(i).SetInt(int64(int8(b)))
		}
		value = slice.Interface()
	case UINT16_ARRAY:
		p.NElements = uint32(len(value.([]uint16)))
	case UINT32_ARRAY:
		p.NElements = uint32(len(value.([]uint32)))
	case UINT64_ARRAY:
		p.NElements = uint32(len(value.([]uint64)))
	case STRING_ARRAY:
		p.NElements = uint32(len(value.([]string)))
		arrType := reflect.ArrayOf(int(p.NElements), reflect.TypeOf(""))
		arr := reflect.New(arrType).Elem()
		for i, b := range value.([]string) {
			arr.Index(i).SetString(b)
		}
		value = arr.Interface()
	case NVLIST:
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "::: recursing :::")
		if err := encode(vbuf, reflect.ValueOf(value)); err != nil {
			fmt.Fprintln(os.Stderr, "error")
			return nil, err
		}
		p.data = vbuf.Bytes()
		fmt.Println("vbuf.Len():", vbuf.Len())
		fmt.Fprintln(os.Stderr, "::: finished  :::")
		fmt.Fprintln(os.Stderr)
	case NVLIST_ARRAY:
		p.NElements = uint32(len(value.([]mList)))
		for _, l := range value.([]mList) {
			if err = encode(vbuf, reflect.ValueOf(l)); err != nil {
				fmt.Fprintln(os.Stderr, "error")
				return nil, err
			}
		}
		p.data = vbuf.Bytes()
	}

	//fmt.Fprintln(os.Stderr, "vbuf before:", vbuf.Bytes())
	if vbuf.Len() == 0 {
		_, err = xdr.NewEncoder(vbuf).Encode(value)
		if err != nil {
			return nil, err
		}
		size = vbuf.Len()
	}

	psize := p.decodedSize()
	p.DecodedSize = uint32(psize)
	p.EncodedSize = uint32(p.encodedSize())
	fmt.Fprintln(os.Stderr, "p.size:", psize)
	fmt.Fprintln(os.Stderr, "vbuf.Len():", vbuf.Len(), "vbuf:", vbuf.Bytes())

	pbuf := &bytes.Buffer{}
	_, err = xdr.NewEncoder(pbuf).Encode(p)
	if err != nil {
		return nil, err
	}

	_, err = pbuf.WriteTo(w)
	if err != nil {
		return nil, err
	}
	_, err = vbuf.WriteTo(w)
	if err != nil {
		return nil, err
	}
	size += pbuf.Len() + vbuf.Len()
	fmt.Println("encoded size?:", p.encodedSize())
	fmt.Println("decoded size:", p.decodedSize())
	fmt.Fprintln(os.Stderr)

	return nil, nil
}

type mVal struct {
	Name  string
	Type  dataType
	Value interface{}
}
type mList map[string]mVal

func Decode(buf []byte) (mList, error) {
	b := bytes.NewReader(buf)

	enc := encoding{}
	err := binary.Read(b, binary.BigEndian, &enc)
	if err != nil {
		return nil, err
	}

	if enc.Encoding > 1 {
		return nil, fmt.Errorf("invalid encoding: %v", enc.Encoding)
	}
	if enc.Endianess > 1 {
		return nil, fmt.Errorf("invalid endianess: %v", enc.Endianess)
	}
	if enc.Reserved1 != 0 {
		return nil, fmt.Errorf("unexpected reserved1 value: %v", enc.Reserved1)
	}
	if enc.Reserved2 != 0 {
		return nil, fmt.Errorf("unexpected reserved2 value: %v", enc.Reserved2)
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

func decodeList(r io.ReadSeeker) (mList, error) {
	var h header
	err := binary.Read(r, binary.BigEndian, &h)
	if err != nil {
		return nil, err
	}

	if h.Version != 0 {
		return nil, fmt.Errorf("unexpected version: %v", h.Version)
	}
	if h.Flag < UNIQUE_NAME || h.Flag > UNIQUE_NAME_TYPE {
		return nil, fmt.Errorf("unexpected Flag: %v", h.Flag)
	}

	m := mList{}
	for {
		end, err := isEnd(r)
		if err != nil {
			return nil, err
		}
		if end {
			break
		}

		p := pair{}
		_, err = xdr.Unmarshal(r, &p)
		if err != nil {
			return nil, err
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
			arr := make([]mList, 0, p.NElements)
			for i := uint32(0); i < p.NElements; i++ {
				var m mList
				m, err = decodeList(r)
				if err != nil {
					break
				}
				arr = append(arr, m)
			}
			v = arr
		default:
			return nil, fmt.Errorf("unknown type: %v", p.Type)
		}
		if err != nil {
			return nil, err
		}

		m[p.Name] = mVal{
			Name:  p.Name,
			Type:  p.Type,
			Value: v,
		}

		//p.data = v
		//pp := Pair{pair: p, Value: v}
		//l.Pairs = append(l.Pairs, pp)

	}
	return m, nil
}
