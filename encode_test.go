package nv

import (
	"fmt"
	"reflect"
	"testing"
)

func TestEncode(t *testing.T) {
	for _, s := range good {
		l, err := Decode(s)
		if err != nil {
			t.Fatal(err)
		}
		b, err := Encode(l)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(b, s) {
			fmt.Printf("%+v\n", l)
			t.Fatalf("encoded does not match payload\nwant:|%v|\n got:|%v|\n", s, b)
		}
	}
}
