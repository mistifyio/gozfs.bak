package nv

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeStructGood(t *testing.T) {
	//for i, test := range good {
	for i, test := range good {
		fmt.Printf("\n\nTEST: %s\n", test.name)

		structs, ok := goodTargetStructs[test.name]
		if !ok {
			continue
		}
		v := reflect.New(reflect.TypeOf(structs.ptr).Elem())
		err := DecodeStruct(test.payload, v.Interface())
		if err != nil {
			fmt.Println(test.name, "failed:", err)
			t.Fail()
		}

		maps, ok := goodTargetMaps[test.name]
		if !ok {
			continue
		}
		m := reflect.New(reflect.TypeOf(maps.ptr).Elem())
		err = DecodeStruct(test.payload, m.Interface())
		if err != nil {
			fmt.Println(test.name, "failed:", err)
			t.Fail()
		}

		e, _ := json.Marshal(structs.out)
		o, _ := json.Marshal(v.Elem().Interface())
		ms, _ := json.Marshal(m.Elem().Interface())
		fmt.Printf("Results:\n\texp: %s\n\tstr: %s\n\tmap: %s\n\n", string(e), string(o), string(ms))
		assert.True(t, assert.ObjectsAreEqualValues(structs.out, v.Elem().Interface()), test.name)
		if i == len(goodTargetStructs) {
			return
		}
	}
}
