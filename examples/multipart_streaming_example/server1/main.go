package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		// Generate a large file
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", "attachment; filename=large.txt")

		// Create a reader that generates data on the fly with line numbering
		var builder strings.Builder
		for i := 1; i <= 10000000; i++ {
			builder.WriteString(fmt.Sprintf("Line %d: This is a line in the large file.\n", i))
		}
		reader := strings.NewReader(builder.String())

		_, err := io.Copy(w, reader)
		if err != nil {
			http.Error(w, "Failed to generate file", http.StatusInternalServerError)
		}
	})

	fmt.Println("Server 1 running on :8080")
	http.ListenAndServe(":8080", nil)
}
