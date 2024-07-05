package main

import (
	"bee"
	"fmt"
	"net/http"
)

func main() {
	engine := bee.New()
	engine.GET("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("URL.Path = " + request.URL.Path + "\n"))
	})
	engine.GET("/hello/*", func(writer http.ResponseWriter, request *http.Request) {
		for k, v := range request.Header {
			fmt.Fprintf(writer, "Header[%q] = %q\n", k, v)
		}
	})
	fmt.Println("Server is running at http://localhost:8000")
	engine.Run(":8000")
}
