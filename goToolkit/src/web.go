package main

import (
	"fmt"
	"net/http"
)

//创建处理器函数
func handler(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintln(w, "Hello World!", r.URL.Path)
	if err != nil {
		return
	}
}
func main() {
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}
}
