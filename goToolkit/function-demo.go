package main

import "fmt"

func double(x int) (result int) {
	defer func() {
		fmt.Printf("double(%d) = %d\n", x, result)
	}()
	return x + x
}

func main() {
	_ = double(4)
}
