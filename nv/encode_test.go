package nv

import (
	"reflect"
	"testing"
)

func TestEncodeGood(t *testing.T) {
	for _, s := range good {
		l, err := Decode(s.payload)
		if err != nil {
			t.Fatal(s.name, err)
		}
		b, err := Encode(l)
		if err != nil {
			t.Fatal(s.name, err)
		}
		if !reflect.DeepEqual(b, s.payload) {
			t.Fatalf("%s failed: encoded does not match payload\nwant:|%v|\n got:|%v|\n", s.name, s.payload, b)
		}
	}
}

func TestEncodeBad(t *testing.T) {
	for _, s := range encode_bad {
		_, err := Encode(s.payload)
		if err == nil {
			t.Fatalf("expected an error, wanted:|%s|, for payload: |%v|\n", s.err, s.payload)
		}
		if s.err != err.Error() {
			t.Fatalf("error mismatch, want:|%s|, got:|%s|, payload:|%v|\n",
				s.err, err.Error(), s.payload)
		}
	}
}
