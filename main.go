package main

import (
	"fmt" //used to format and print something
	"log" //similar to console log
	"net/http" //Go built in web server
)

func main() {
	//creating a router using the NewServeMux()
	mux := http.NewServeMux()

	//calling the HandleFunc and passing the req type, path and also a function that accepts and res, and req.
	// here r is a pointer to the request.
	// fmt.Fprintln print the response 
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello from Jobseek!")
	})

	//logs that the server is listening on port 808
	log.Println("listening on http://localhost:8080")

	// Start the HTTP server on port 8080.
	// If the server fails, log the error and terminate the program.
	log.Fatal(http.ListenAndServe(":8080", mux))
}
