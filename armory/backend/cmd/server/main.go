// package main

// import (
// 	"log"
// 	"net/http"
// )

// func main() {
// 	mux := http.NewServeMux()

// 	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
// 		w.Write([]byte("ok"))
// 	})

//		addr := ":8080"
//		log.Printf("Listening on %s", addr)
//		log.Fatal(http.ListenAndServe(addr, mux))
//	}
package main

import (
	"log"

	"armory/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
