package testdata

// TestType test type comment 1
// second line comment
//go:generate gogen stringer -t TestType
type TestType int // comment 2

const (
	// T1 comment 11
	T1 TestType = iota // line suffix comment 1
	// T2 comment 22
	T2
	T3 // T3 line suffix comment 1
)

// TestString comment for test string 1
type TestString string // comment for test string 2

// TestFunc func doc
func TestFunc(in int) (out int) {
	return
}

// NewTestType new func 1
func NewTestType(v int) TestType {
	return TestType(v)
}

// NewTestType2 new func 2
func NewTestType2(v int) (TestType, error) {
	return TestType(v), nil
}

type v struct {
}

func (*v) NewTestFunc() TestType {
	return NewTestType(1)
}

func XXFD() {
}
