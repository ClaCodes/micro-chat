package main;

import (
    "fmt"
    "log"
    "net/http"
)

var i int

func index(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, Goth %d", i)
    log.Printf("was called %d", i)
    i++
}

func main() {
    http.HandleFunc("/hello", index)

    fmt.Printf("Server running (port=8080), route: http://localhost:8080/hello\n")
    if err := http.ListenAndServe(":8080", nil); err != nil{
        log.Fatal(err)
    }
}
