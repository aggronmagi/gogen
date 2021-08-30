package sample

import "github.com/aggronmagi/gogen/gen"

type MInterface interface {
	Value() (int, error)
}

type MTest struct {
}

func (m MTest) TestF1(int32, int32) (string, error) {
	return "", nil
}

func (m MTest) TestF2(v1 int32, f1 int32) (string, error) {
	return "", nil
}

type TestInt int32

// MFuncTest CCCC  XX  X X X X
type MFuncTest struct {
	//dd TestInt
	*MTest
	//v2 int32
	*gen.Generator
	MInterface
	// gen.GeneratorIFace
	// chan int
	// []int32
	// int32
	// map[int32]int32
}

func (m *MFuncTest) MX(v1 MTest, v2 *MTest, v3 []*MTest, v4 map[int]*MTest, v5 map[int][]*MTest,
	v6 map[int]map[int]*MTest, v7 chan *MTest, v8 chan MTest) {
}
