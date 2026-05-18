package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type req struct {
	Num int
}

type resp struct {
	Sequence []int
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /syr", func(w http.ResponseWriter, r *http.Request) {
		var in req
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		out := &resp{
			Sequence: syr(in.Num),
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(out); err != nil {
			log.Printf("write response: %v", err)
		}
	})

	log.Println("started and listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func syr(n int) []int {
	iter := []int{}
	for n != 1 {
		iter = append(iter, n)
		if n%2 == 0 {
			n = n / 2
			continue
		}
		n = 3*n + 1
	}
	return iter
}
