package main

import (
	"reflect"
)

type Book struct {
	title  string
	author Author
}

type Author struct {
	Person
	penName string
}

type Person struct {
	name string
	age  int
}

func createObject(t reflect.Type) interface{} {
	v := reflect.New(t)
	return v.Interface()
}

func main() {
	book := Book{
		title: "",
		author: Author{
			Person: Person{
				name: "John Doe",
				age:  30,
			},
			penName: "",
		},
	}
	book.author.name = "Book Title"
	// book.name
}
