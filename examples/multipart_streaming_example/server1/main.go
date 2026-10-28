package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", "attachment; filename=large.txt")

		flusher, ok := w.(http.Flusher)
		ctx := r.Context()

		const totalLines = 10000000
		for i := 1; i <= totalLines; i++ {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if _, err := fmt.Fprintf(w, "Line %d: This is a line in the large file.\n", i); err != nil {
				return
			}

			if ok && i%5000 == 0 {
				flusher.Flush()
			}
		}

		if ok {
			flusher.Flush()
		}
	})

	fmt.Println("Server 1 running on :8080")
	http.ListenAndServe(":8080", nil)
}
