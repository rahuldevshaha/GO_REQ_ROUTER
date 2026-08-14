package main

import (
	"log"
	"net/http"
)

func main() {
	lbs, err := loadLBs("mainLBserver.json")
	if err != nil {
		log.Fatal(err)
	}

	router := NewRouter(lbs)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs.html")
	})
	http.HandleFunc("/route", router.RouteHandler)
	http.HandleFunc("/health", router.HealthHandler)

	log.Println("Router running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
