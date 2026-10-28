package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

type progressWriter struct {
	writer   io.Writer
	logger   *slog.Logger
	total    int64
	reported int64
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.writer.Write(p)
	pw.total += int64(n)
	if pw.logger != nil && pw.total-pw.reported >= 50*1024*1024 {
		pw.reported = pw.total
		pw.logger.Info("Upload progress", "bytes", pw.total, "MB", fmt.Sprintf("%.2f MB", float64(pw.total)/(1024*1024)))
	}
	return n, err
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		mr, err := r.MultipartReader()
		if err != nil {
			logger.Error("Failed to get multipart reader", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var uploadedFilename string

		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				logger.Error("Failed to read next part", "error", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if part.FormName() == "file" {
				rawFilename := part.FileName()
				if rawFilename == "" {
					rawFilename = "uploaded.bin"
				}
				cleanFilename := filepath.Base(rawFilename)
				if cleanFilename == "." || cleanFilename == "/" || cleanFilename == "\\" || cleanFilename == "" {
					cleanFilename = "uploaded.bin"
				}

				dst, err := os.Create(cleanFilename)
				if err != nil {
					part.Close()
					logger.Error("Failed to create file", "error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				pw := &progressWriter{
					writer: dst,
					logger: logger,
				}

				_, copyErr := io.Copy(pw, part)
				closeDstErr := dst.Close()
				closePartErr := part.Close()

				if copyErr != nil {
					logger.Error("Failed to copy file", "error", copyErr)
					http.Error(w, copyErr.Error(), http.StatusInternalServerError)
					return
				}
				if closeDstErr != nil {
					logger.Error("Failed to close file", "error", closeDstErr)
					http.Error(w, closeDstErr.Error(), http.StatusInternalServerError)
					return
				}
				if closePartErr != nil {
					logger.Error("Failed to close part", "error", closePartErr)
				}

				uploadedFilename = cleanFilename
			} else {
				part.Close()
			}
		}

		if uploadedFilename == "" {
			http.Error(w, "file part not found", http.StatusBadRequest)
			return
		}

		logger.Info("File saved successfully", "filename", uploadedFilename)
		fmt.Fprintf(w, "File %s uploaded and saved", uploadedFilename)
	})

	fmt.Println("Server 2 running on :8081")
	http.ListenAndServe(":8081", nil)
}
