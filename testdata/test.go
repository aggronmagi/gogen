package testdata

// TestType test type comment 1
// xxx
// xxx2
// xdxx4
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

func TestFunc(in int) (out int) {
	return
}
