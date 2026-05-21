package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
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

type cache struct {
	store map[num]sequence
	mu    sync.Mutex
}

func main() {
	c := cache{
		store: make(map[num]sequence),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /syr", func(w http.ResponseWriter, r *http.Request) {
		var in req
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if !validateRequest(in.Num) {
			http.Error(w, "invalid request: cannot be empty and the starting numbers must be >= 2", http.StatusBadRequest)
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
		// Set a timeout on the query and prepare context for cancellation
		timeout := 5 * time.Second
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		var in reqBatch
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if len(in.Nums) < 1 || !validateBatchRequest(in.Nums) {
			http.Error(w, "invalid request: cannot be empty and the starting numbers must be >= 2", http.StatusBadRequest)
			return
		}

		results := make(chan sequence, len(in.Nums))

		for _, n := range in.Nums {
			go func(n num) {
				// check if sequence for starting num is already cached
				var seq sequence
				c.mu.Lock()
				seq, hit := c.store[n]
				c.mu.Unlock()

				// if cache not hit, make the calculation
				if !hit {
					slog.Info("cache miss", "num", n)
					var ok bool
					seq, ok = syrContextAware(ctx, n)
					// not ok means early return because cancellation
					if !ok {
						return
					}
				} else {
					slog.Info("cache hit", "num", n)
				}

				select {
				case results <- seq:
				case <-ctx.Done():
				}
				return
			}(n)
		}

		var res sequence
		sequences := make([]sequence, 0, len(in.Nums))

		// will block until first result comes or context is cancelled
		for range in.Nums {
			select {
			case res = <-results:
				// save the sequence in the result to the cache if not already there
				var n num
				if len(res) >= 1 {
					n = res[0]
				} else {
					http.Error(w, "computing the sequences: empty result sequence", http.StatusInternalServerError)
					return
				}

				c.mu.Lock()
				if _, ok := c.store[n]; !ok {
					c.store[n] = res
				}
				c.mu.Unlock()

				sequences = append(sequences, res)
			case <-ctx.Done():
				http.Error(w, "computing the sequences: timeout", http.StatusInternalServerError)
				return
			}
		}

		out := &respBatch{
			Sequences: sequences,
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(out); err != nil {
			log.Printf("write response: %v", err)
		}
	})

	mux.HandleFunc("POST /syr/batch/winner", func(w http.ResponseWriter, r *http.Request) {
		// Set a timeout on the query and prepare context for cancellation
		timeout := 1 * time.Second
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		var in reqBatch
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if len(in.Nums) < 1 || !validateBatchRequest(in.Nums) {
			http.Error(w, "invalid request: cannot be empty and the starting numbers must be >= 2", http.StatusBadRequest)
			return
		}

		results := make(chan sequence, len(in.Nums))

		for _, n := range in.Nums {
			go func(n num) {
				seq, ok := syrContextAware(ctx, n)
				if !ok {
					return
				}

				select {
				case results <- seq:
				case <-ctx.Done():
					return
				}
			}(n)
		}

		var res sequence

		// will block until first result comes or context is cancelled
		select {
		case res = <-results:
			cancel()
		case <-ctx.Done():
			http.Error(w, "error computing the sequences: no results", http.StatusInternalServerError)
			return
		}

		out := &resp{
			Sequence: res,
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

// syr is now context aware, returning even before finishing the iterations when context is cancelled
// it returns true when it finished the iterations
func syrContextAware(ctx context.Context, n num) (sequence, bool) {
	iter := sequence{}

	for n != 1 {
		select {
		case <-ctx.Done():
			return nil, false
		default:
		}

		iter = append(iter, n)

		if n%2 == 0 {
			n = n / 2
			continue
		}

		n = 3*n + 1
	}

	return iter, true
}

func validateRequest(n num) bool {
	if n < 2 {
		return false
	}
	return true
}

func validateBatchRequest(seq []num) bool {
	for _, n := range seq {
		if n < 2 {
			return false
		}
	}
	return true
}
