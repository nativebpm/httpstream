package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

type progressWriter struct {
	writer   io.Writer
	logger   *slog.Logger
	total    int64
	reported int64
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		// Parse multipart
		err := r.ParseMultipartForm(32 << 20) // 32MB max
		if err != nil {
			logger.Error("Failed to parse multipart", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			logger.Error("Failed to get form file", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		dst, err := os.Create(header.Filename)
		if err != nil {
			logger.Error("Failed to create file", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		_, err = io.Copy(dst, file)
		if err != nil {
			logger.Error("Failed to copy file", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		logger.Info("File saved successfully", "filename", header.Filename)
		fmt.Fprintf(w, "File %s uploaded and save", header.Filename)
	})

	fmt.Println("Server 2 running on :8081")
	http.ListenAndServe(":8081", nil)
}
