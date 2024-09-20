package jpath

type A struct {
	b
	c
}

type b struct {
	valueb  int32
	Point2x X
}

type c struct {
	valuec int32
}

type X struct {
	Value int32
}
