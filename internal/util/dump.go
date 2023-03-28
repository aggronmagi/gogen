// Copyright 2014 The godump Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

type variable struct {
	// Output dump string
	Out string

	// Indent counter
	indent int64
}

func (v *variable) dump(val reflect.Value, name string, ignore bool) {

	v.indent++
	defer func() {
		v.indent--
	}()

	if val.IsValid() && val.CanInterface() {
		typ := val.Type()

		switch typ.Kind() {
		case reflect.Array, reflect.Slice:
			if val.IsNil() {
				return
			}
			if !ignore {
				v.printType(name, val.Interface())
			}
			l := val.Len()
			for i := 0; i < l; i++ {
				v.printIndex(i)
				v.dump(val.Index(i), strconv.Itoa(i), true)
			}
		case reflect.Map:
			if val.IsNil() {
				return
			}
			v.printType(name, val.Interface())
			//l := val.Len()
			keys := val.MapKeys()
			for _, k := range keys {
				v.dump(val.MapIndex(k), k.Interface().(string), true)
			}
		case reflect.Ptr:
			if !val.IsNil() {
				if !ignore {
					v.printType(name, val.Interface())
				}
				v.dump(val.Elem(), name, true)
			}
		case reflect.Struct:
			if !ignore {
				v.printType(name, val.Interface())
			}
			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				v.dump(val.FieldByIndex([]int{i}), field.Name, false)
			}
		default:
			v.printValue(name, val.Interface())
		}
	} else {
		v.printValue(name, "")
	}
}

func (v *variable) printType(name string, vv interface{}) {
	v.printIndent()
	v.Out = fmt.Sprintf("%s%s(%T)\n", v.Out, name, vv)
}

func (v *variable) printValue(name string, vv interface{}) {
	v.printIndent()
	v.Out = fmt.Sprintf("%s%s(%T) %#v\n", v.Out, name, vv, vv)
}

func (v *variable) printIndex(index int) {
	v.printIndent()
	v.Out = fmt.Sprintf("%s- %d -:\n", v.Out, index)
}

func (v *variable) printIndent() {
	var i int64
	for i = 0; i < v.indent; i++ {
		v.Out = fmt.Sprintf("%s  ", v.Out)
	}
}

// Dump print to standard out the value that is passed as the argument with indentation.
// Pointers are dereferenced.
func Dump(v interface{}, tips ...string) {
	val := reflect.ValueOf(v)
	dump := &variable{indent: -1}
	dump.dump(val, "  ", false)
	if frame, ok := getCallerFrame(1); ok {
		if len(dump.Out) == 0 {
			dump.Out = "nil"
		}
		fmt.Printf("%s:%d:\n%s = %s\n", frame.File, frame.Line, strings.Join(tips, "\n"), dump.Out)
	} else {
		fmt.Printf("%s\n", dump.Out)
	}

}

// Sdump return the value that is passed as the argument with indentation.
// Pointers are dereferenced.
func Sdump(v interface{}, name string) string {
	val := reflect.ValueOf(v)
	dump := &variable{indent: -1}
	dump.dump(val, name, false)
	return dump.Out
}

// getCallerFrame gets caller frame. The argument skip is the number of stack
// frames to ascend, with 0 identifying the caller of getCallerFrame. The
// boolean ok is false if it was not possible to recover the information.
//
// Note: This implementation is similar to runtime.Caller, but it returns the whole frame.
// copy from zap
func getCallerFrame(skip int) (frame runtime.Frame, ok bool) {
	const skipOffset = 2 // skip getCallerFrame and Callers

	pc := make([]uintptr, 1)
	numFrames := runtime.Callers(skip+skipOffset, pc[:])
	if numFrames < 1 {
		return
	}

	frame, _ = runtime.CallersFrames(pc).Next()
	return frame, frame.PC != 0
}
