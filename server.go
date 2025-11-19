package main

import (
	"html/template"
	"net/http"
)

func home(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("static/index.html")
	t.Execute(w, nil)
}
func journal(w http.ResponseWriter, r *http.Request) {
	//fmt.Fprintf(w, "journal")
	t, _ := template.ParseFiles("hello.html")
	t.Execute(w, nil)
}
func getRequest() {
	http.HandleFunc("/", home)
	http.HandleFunc("/journal/", journal)
	http.ListenAndServe(":8080", nil)
}

func main() {
	getRequest()
}
