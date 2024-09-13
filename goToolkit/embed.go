package main

import (
	"fmt"
	"image/color"
	"sync"
)

type Point struct{ X, Y float64 }

type ColoredPoint struct {
	Point
	Color color.RGBA
}

func main() {
	//var index1, index2 *Index
	//index1 = &Index{1, "partition1"}
	//index2 = index1
	//
	//fmt.Printf("%T,%p,%p\n", index1, index1, &index1)
	//fmt.Printf("%T,%p,%p\n", index2, index2, &index2)
	c := Cache{index: Index{1, "partition1"}}
	fmt.Printf("%T,%p,%p\n", c.index, &c.index, &c.index)
	c.Pr()
}

type Index struct {
	indexId        int32
	indexPartition string
}

type Cache struct {
	sync.Mutex
	mapping map[string]string
	index   Index
}

func (c Cache) Pr() {
	fmt.Printf("%T,%p,%p\n", c.index, &c.index, &c.index)
}

func Lookup(c Cache, key string) string {
	c.Lock()
	v := c.mapping[key]
	c.Unlock()
	return v
}
