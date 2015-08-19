package nv

const enc_dec = "\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00"

//                                  name length          ┌╴name...fill    beginning of type
const enc_dec_name_typ = enc_dec + "\x00\x00\x00\x01" + "0\x00\x00\x00" + "\x00\x00\x00"

var bad = []struct {
	payload []byte
	err     string
}{
	{[]byte{}, "EOF"},
	{[]byte("\x00"), "unexpected EOF"},
	{[]byte("\x00\x00\x00\x00"), "EOF"},
	{[]byte("\x02\x00\x00\x00"), "invalid encoding: 2"},
	{[]byte("\x01\x02\x00\x00"), "invalid endianess: 2"},
	{[]byte("\x01\x01\x01\x00"), "unexpected reserved1 value: 1"},
	{[]byte("\x01\x01\x00\x01"), "unexpected reserved2 value: 1"},
	{[]byte("\x01\x01\x00\x00\x00\x00\x00\x00"), "unexpected EOF"},
	{[]byte("\x01\x01\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00"), "unexpected version: 1"},
	{[]byte("\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"), "unexpected Flag: flag(0)"},
	{[]byte("\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x03"), "unexpected Flag: flag(3)"},
	{[]byte("\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00"), "xdr:DecodeUint: unexpected EOF while decoding 4 bytes - read: '[0]'"},
	/*
		type pair struct {
			EncodedSize uint32
			DecodedSize uint32
			Name        string
			Type        dataType
			NElements   uint32
			data        interface{}
		}
	*/
	{[]byte(enc_dec_name_typ + "\x00" + "\x00\x00\x00\x01\x00\x00\x00\x00"), "unknown type: UNKNOWN"},
	{[]byte(enc_dec_name_typ + string(byte(DOUBLE+1)) + "\x00\x00\x00\x01\x00\x00\x00\x00"), "unknown type: dataType(28)"},
	{[]byte(enc_dec_name_typ + string(byte(BYTE)) + "\x00\x00\x00\x01\x00\x00\x00\x00"), "EOF"},
	{[]byte(enc_dec_name_typ + string(byte(BYTE)) + "\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00"), "unexpected EOF"},
	{[]byte(enc_dec_name_typ + string(byte(BYTE)) + "\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01"), "xdr:DecodeUint: EOF while decoding 4 bytes - read: '[]'"},
	{[]byte(enc_dec_name_typ + string(byte(STRING)) + "\x00\x00\x00\x00\xff\xff\xff\xff\x00\x00\x00\x00\x00\x00\x00\x00"), "xdr:DecodeString: data exceeds max slice limit - read: '4294967295'"},
	{[]byte(enc_dec_name_typ + string(byte(NVLIST)) + "\x00\x00\x00\x01" + string([]byte(enc_dec_name_typ + "\x00" + "\x00\x00\x00\x01\x00\x00\x00\x00")[4:]) + "\x00\x00\x00\x00\x00\x00\x00\x00"), "unknown type: UNKNOWN"},
	{[]byte(enc_dec_name_typ + string(byte(NVLIST_ARRAY)) + "\x00\x00\x00\x01" + string([]byte(enc_dec_name_typ + "\x00" + "\x00\x00\x00\x01\x00\x00\x00\x00")[4:]) + "\x00\x00\x00\x00\x00\x00\x00\x00"), "unknown type: UNKNOWN"},
}
