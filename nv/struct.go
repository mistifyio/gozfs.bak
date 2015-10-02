package nv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
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

func (p pair) headerSize() int {
	return 4 + 4 + 4 + align4(len(p.Name)) + 4 + 4
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
	case BOOLEAN_ARRAY, INT8_ARRAY, UINT8_ARRAY, INT16_ARRAY, UINT16_ARRAY, INT32_ARRAY, UINT32_ARRAY:
		valSize = 4 + int(p.NElements*4)
	case INT64_ARRAY, UINT64_ARRAY:
		valSize = 4 + int(p.NElements*8)
	case STRING:
		valSize = 4 + len(p.data.(string)) + 1
	case NVLIST, NVLIST_ARRAY:
		valSize = len(p.data.([]byte))
	case STRING_ARRAY:
		slice := p.data.([]string)
		for i := range slice {
			valSize += align4(4 + len(slice[i]) + 1)
		}
	}
	return p.headerSize() + align4(valSize)
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
				return err
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
	fmt.Fprintf(os.Stderr, "encodeItem: field: %#v\n", field)
	fmt.Println("encodeItem: name:", name, "CanSet:", field.CanSet())

	p := pair{
		Name:      name,
		NElements: 1,
	}

	switch t := field.Kind(); t {
	case reflect.String:
		p.Type = STRING
	case reflect.Uint64:
		p.Type = UINT64
	case reflect.Int32, reflect.Int:
		p.Type = INT32
	case reflect.Struct:
		p.Type = NVLIST
	default:
		panic(fmt.Sprint("unknown type:", t))
	}

	fmt.Println("encodeItem: field.Type().String():", field.Type().String())
	if field.Type().String() == "nv.mVal" {
		p.Type = field.FieldByName("Type").Interface().(dataType)
		p.Name = field.FieldByName("Name").Interface().(string)
		p.data = field.FieldByName("Value").Interface()
	} else {
		fmt.Println("name:", name, "CanAddr:", field.CanAddr())
		if !field.CanSet() {
			panic("can't Set")
			return nil, nil
		}
		p.data = field.Interface()
	}
	value := p.data
	fmt.Fprintf(os.Stderr, "encode: name: %#v, p: %#v, value: %v\n", name, p, value)
	fmt.Fprintln(os.Stderr, "p.UNKNOWN:", int(UNKNOWN))

	if p.Type == UNKNOWN || p.Type > DOUBLE {
		fmt.Println("returning error")
		return nil, fmt.Errorf("invalid Type '%v'", p.Type)
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
			if err := encode(vbuf, reflect.ValueOf(l)); err != nil {
				fmt.Fprintln(os.Stderr, "error")
				return nil, err
			}
		}
		p.data = vbuf.Bytes()
	}

	//fmt.Fprintln(os.Stderr, "vbuf before:", vbuf.Bytes())
	if vbuf.Len() == 0 {
		_, err := xdr.NewEncoder(vbuf).Encode(value)
		if err != nil {
			return nil, err
		}
	}

	psize := p.decodedSize()
	p.DecodedSize = uint32(psize)
	p.EncodedSize = uint32(p.encodedSize())
	fmt.Fprintln(os.Stderr, "p.size:", psize)
	fmt.Fprintln(os.Stderr, "vbuf.Len():", vbuf.Len(), "vbuf:", vbuf.Bytes())

	pbuf := &bytes.Buffer{}
	_, err := xdr.NewEncoder(pbuf).Encode(p)
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

func DecodeStruct(data []byte, target interface{}) (err error) {
	// Catch any panics from reflection
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			if rs, ok := r.(string); ok {
				err = errors.New(rs)
			} else {
				err = r.(error)
			}
		}
	}()

	b := bytes.NewReader(data)

	// Validate data encoding
	enc := encoding{}
	if err := binary.Read(b, binary.BigEndian, &enc); err != nil {
		return err
	}
	if enc.Encoding > 1 {
		return fmt.Errorf("invalid encoding: %v", enc.Encoding)
	}
	if enc.Endianess > 1 {
		return fmt.Errorf("invalid endianess: %v", enc.Endianess)
	}
	if enc.Reserved1 != 0 {
		return fmt.Errorf("unexpected reserved1 value: %v", enc.Reserved1)
	}
	if enc.Reserved2 != 0 {
		return fmt.Errorf("unexpected reserved2 value: %v", enc.Reserved2)
	}

	// Validate target
	targetV := reflect.ValueOf(target)
	if targetV.Kind() != reflect.Ptr {
		return fmt.Errorf("cannot decode into non-pointer: %v", reflect.TypeOf(targetV).String())
	}
	if targetV.IsNil() {
		return fmt.Errorf("cannot decode into nil")
	}

	return decodeListStruct(b, reflect.Indirect(targetV))
}

func decodeListStruct(r io.ReadSeeker, target reflect.Value) error {
	// Validate data header
	var h header
	if err := binary.Read(r, binary.BigEndian, &h); err != nil {
		return err
	}
	if h.Version != 0 {
		return fmt.Errorf("unexpected version: %v", h.Version)
	}
	if h.Flag < UNIQUE_NAME || h.Flag > UNIQUE_NAME_TYPE {
		return fmt.Errorf("unexpected Flag: %v", h.Flag)
	}

	// Make sure the target is not a pointer. If it is, initilize and
	// dereference.
	if target.Kind() == reflect.Ptr {
		target.Set(reflect.New(target.Type().Elem()))
		target = reflect.Indirect(target)
	}

	// Maps and structs have slightly different handling
	isMap := (target.Kind() == reflect.Map)

	if isMap {
		// Initialize the map. Can't add keys without this.
		target.Set(reflect.MakeMap(target.Type()))
	}

	// For structs, create the lookup table for field name/tag -> field index
	var targetFieldIndexMap map[string]int
	if !isMap {
		targetFieldIndexMap = fieldIndexMap(target)
	}

	// Start decoding data
	for {
		// Done when there's no more data or an error has occured
		if end, err := isEnd(r); end || err != nil {
			return err
		}

		// Get next encoded data pair
		// Note: This just gets the data pair metadata. The actual value data
		// is left on the reader, to be pulled off by the decoder.
		dataPair := pair{}
		if _, err := xdr.Unmarshal(r, &dataPair); err != nil {
			return err
		}

		var targetMapKey reflect.Value
		if isMap {
			// Map keys are set with reflect.Values rather than strings
			targetMapKey = reflect.ValueOf(dataPair.Name)
		}

		// If not dealing with a map, look up the corresponding struct field
		var targetField reflect.Value
		if !isMap {
			targetFieldIndex, ok := targetFieldIndexMap[dataPair.Name]
			// If there's no corresponding struct field, skip the data and move
			// on to the next data pair
			if !ok {
				r.Seek(int64(dataPair.EncodedSize-uint32(dataPair.headerSize())), 1)
				continue
			} else {
				targetField = target.Field(targetFieldIndex)
			}
		}

		var err error
		dec := newDecoder(r)
		switch dataPair.Type {
		case BOOLEAN_VALUE:
			v, err := dec.DecodeBool()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.SetBool(v)
				}
			}
		case BYTE:
			v, err := dec.DecodeByte()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.SetUint(uint64(v))
				}
			}
		case INT8:
			v, err := dec.DecodeInt8()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.SetInt(int64(v))
				}
			}
		case INT16:
			v, err := dec.DecodeInt16()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.SetInt(int64(v))
				}
			}
		case INT32:
			v, err := dec.DecodeInt32()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.SetInt(int64(v))
				}
			}
		case INT64:
			v, err := dec.DecodeInt64()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.SetInt(v)
				}
			}
		case UINT8:
			v, err := dec.DecodeUint8()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.SetUint(uint64(v))
				}
			}
		case UINT16:
			v, err := dec.DecodeUint16()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.SetUint(uint64(v))
				}
			}
		case UINT32:
			v, err := dec.DecodeUint32()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.SetUint(uint64(v))
				}
			}
		case UINT64:
			v, err := dec.DecodeUint64()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.SetUint(uint64(v))
				}
			}
		case HRTIME:
			v, err := dec.DecodeHRTime()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					// TODO: CONFIRM THIS IS OK
					targetField.SetInt(int64(v))
				}
			}
		case DOUBLE:
			v, err := dec.DecodeFloat64()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.SetFloat(v)
				}
			}
		case BOOLEAN_ARRAY:
			v, err := dec.DecodeBoolArray()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.Set(reflect.ValueOf(v))
				}
			}
		case BYTE_ARRAY:
			if _, err = r.Seek(-4, 1); err == nil {
				v, err := dec.DecodeByteArray()
				if err == nil {
					if isMap {
						target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
					} else {
						targetField.SetBytes(v)
					}
				}
			}
		case INT8_ARRAY:
			v, err := dec.DecodeInt8Array()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.Set(reflect.ValueOf(v))
				}
			}
		case INT16_ARRAY:
			v, err := dec.DecodeInt16Array()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.Set(reflect.ValueOf(v))
				}
			}
		case INT32_ARRAY:
			v, err := dec.DecodeInt32Array()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.Set(reflect.ValueOf(v))
				}
			}
		case INT64_ARRAY:
			v, err := dec.DecodeInt64Array()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.Set(reflect.ValueOf(v))
				}
			}
		case UINT8_ARRAY:
			v, err := dec.DecodeUint8Array()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.Set(reflect.ValueOf(v))
				}
			}
		case UINT16_ARRAY:
			v, err := dec.DecodeUint16Array()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.Set(reflect.ValueOf(v))
				}
			}
		case UINT32_ARRAY:
			v, err := dec.DecodeUint32Array()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.Set(reflect.ValueOf(v))
				}
			}
		case UINT64_ARRAY:
			v, err := dec.DecodeUint64Array()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.Set(reflect.ValueOf(v))
				}
			}
		case STRING:
			v, err := dec.DecodeString()
			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
				} else {
					targetField.SetString(v)
				}
			}
		case STRING_ARRAY:
			if _, err = r.Seek(-4, 1); err == nil {
				v, err := dec.DecodeStringArray()
				if err == nil {
					if isMap {
						target.SetMapIndex(targetMapKey, reflect.ValueOf(v))
					} else {
						targetField.Set(reflect.ValueOf(v))
					}
				}
			}
		case NVLIST:
			if isMap {
				elem := reflect.Indirect(reflect.New(target.Type().Elem()))
				err = decodeListStruct(r, elem)
				if err == nil {
					target.SetMapIndex(targetMapKey, elem)
				}
			} else {
				elem := reflect.Indirect(reflect.New(targetField.Type()))
				err = decodeListStruct(r, elem)
				if err == nil {
					targetField.Set(elem)
				}
			}
		case NVLIST_ARRAY:
			var sliceType reflect.Type
			if isMap {
				sliceType = target.Type().Elem()
			} else {
				sliceType = targetField.Type()
			}
			arr := reflect.MakeSlice(sliceType, 0, int(dataPair.NElements))
			for i := uint32(0); i < dataPair.NElements; i++ {
				elem := reflect.Indirect(reflect.New(sliceType.Elem()))
				err = decodeListStruct(r, elem)
				if err != nil {
					break
				}
				arr = reflect.Append(arr, elem)
			}

			if err == nil {
				if isMap {
					target.SetMapIndex(targetMapKey, arr)
				} else {
					targetField.Set(arr)
				}
			}
		default:
			return fmt.Errorf("unknown type: %v", dataPair.Type)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// fieldIndexMap creates a map of field names, with optional tag name overrides,
// to their index
func fieldIndexMap(v reflect.Value) map[string]int {
	vFieldIndexMap := make(map[string]int)
	for i := 0; i < v.NumField(); i++ {
		vField := v.Field(i)
		// Skip fields that can't be set (e.g. unexported)
		if !vField.CanSet() {
			continue
		}
		vTypeField := v.Type().Field(i)
		dataFieldName := vTypeField.Name
		if tagFieldName := vTypeField.Tag.Get("nv"); tagFieldName != "" {
			dataFieldName = tagFieldName
		}
		vFieldIndexMap[dataFieldName] = i
	}
	return vFieldIndexMap
}
