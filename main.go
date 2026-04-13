package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("hello")

	mux := http.NewServeMux()

	log.Fatal(http.ListenAndServe(":8080", mux))

}
