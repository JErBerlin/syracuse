package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type num int

type req struct {
	Num num
}

type reqBatch struct {
	Nums []num
}

type sequence []num

type resp struct {
	Sequence sequence
}

type respBatch struct {
	Sequences []sequence
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

	mux.HandleFunc("POST /syr/batch", func(w http.ResponseWriter, r *http.Request) {
		var in reqBatch
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if len(in.Nums) < 1 {
			http.Error(w, "invalid request, cannot be empty", http.StatusBadRequest)
			return
		}

		results := make([]sequence, 0, len(in.Nums))

		for _, n := range in.Nums {
			seq := syr(n)
			results = append(results, seq)
		}

		// small redundance, will make sense when concurrency implemented
		var sequences []sequence
		for _, res := range results {
			sequences = append(sequences, res)
		}

		out := &respBatch{
			Sequences: sequences,
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

func syr(n num) []num {
	iter := []num{}
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
