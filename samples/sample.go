package sample

import "log"

//go:generate gogen import ./testdata -t TestType -t TestString -o gen_td.go

// Google Public DNS provides two distinct DoH APIs at these endpoints
// Using the GET method can reduce latency, as it is cached more effectively.
// RFC 8484 GET requests must have a ?dns= query parameter with a Base64Url encoded DNS message. The GET method is the only method supported for the JSON API.

//go:generate gogen option
func ConfigOptionDeclareWithDefault() interface{} {
	return map[string]interface{}{
		// test comment 1
		// test comment 2
		"TestNil":  nil,   // test comment 3
		"TestBool": false, // test comment 4
		// 这里是函数注释1
		// 这里是函数注释2
		"TestInt":         32,                         // default 32
		"TestInt64":       int64(32),                  // int64 line
		"TestSliceInt":    []int{1, 2, 3},             // slice int
		"TestSliceInt64":  []int64{1, 2, 3},           // slice int64 line
		"TestSliceString": []string{"test1", "test2"}, // slice string
		"TestSliceBool":   []bool{false, true},        // slice bool line comment
		"TestSliceIntNil": []int(nil),                 // TestSliceIntNil line comment
		"TestSliceByte":   []byte(nil),                // TestSliceByte line comment
		// SliceInt Doc
		"TestSliceIntEmpty": []int{},                       // Slice int line comment
		"TestMapIntInt":     map[int]int{1: 1, 2: 2, 3: 3}, // TestMapIntInt line comment
		"TestMapIntString":  map[int]string{1: "test"},     // TestMapIntString line comment
		"TestMapStringInt":  map[string]int{"test": 1},     // TestMapStringInt line comment
		// MapStringString Doc
		"TestMapStringString": map[string]string{"test": "test"}, // MapStringString Line Comment

		"TestString": "Meow",
		// Food Doc
		"Food": (*string)(nil), // Food Line Comment
		// Walk Doc
		"Walk": func() {
			log.Println("Walking")
		}, // Walk Line Comment
		// TestNilFunc
		"TestNilFunc": (func())(nil), // 中文1
		// TestReserved1_
		"TestReserved1_": []byte(nil), // 在调优或者运行阶段，我们可能需要动态查看连接池中的一些指标，
		// 来判断设置的值是否合理，或者检测连接池是否有异常情况出现
		"TestReserved2Inner": 1, // TestReserved2Inner after
	}
}

// HTTP parsing and communication with DNS resolver was successful, and the response body content is a DNS response in either binary or JSON encoding,
// depending on the query endpoint, Accept header and GET parameters.

//go:generate gogen option -f -a
func SpecOptionDeclareWithDefault() interface{} {
	return map[string]interface{}{
		// test comment 5
		// test comment 6
		"TestNil1":  nil,   // test comment 1
		"TestBool1": false, // test comment 2
		// 这里是函数注释3
		// 这里是函数注释4
		"TestInt1":       32,
		"TestNilFunc1":   (func())(nil), // 中文2
		"TestReserved2_": []byte(nil),
		// sql.DB对外暴露出了其运行时的状态db.DBStats，sql.DB在关闭，创建，释放连接时候，会维护更新这个状态。
		// 我们可以通过prometheus来收集连接池状态，然后在grafana面板上配置指标，使指标可以动态的展示。
		"TestReserved2Inner1": 1,
		// Test Append Func
		"SliceOpt": []int32{},
	}
}
