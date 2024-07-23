package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// Function to handle the incoming data
func capturePhotoAndLocation(w http.ResponseWriter, r *http.Request) {
	// Parse the multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		log.Println("Error parsing form:", err)
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Retrieve the file
	file, handler, err := r.FormFile("photo")
	if err != nil {
		log.Println("Error retrieving file:", err)
		http.Error(w, "Unable to retrieve file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Ensure the uploads directory exists
	uploadDir := "uploads"
	err = os.MkdirAll(uploadDir, os.ModePerm)
	if err != nil {
		log.Println("Error creating uploads directory:", err)
		http.Error(w, "Unable to create uploads directory", http.StatusInternalServerError)
		return
	}

	// Create a file path
	filePath := filepath.Join(uploadDir, handler.Filename)

	// Create a file to save the uploaded photo
	destFile, err := os.Create(filePath)
	if err != nil {
		log.Println("Error creating file:", err)
		http.Error(w, "Unable to create file", http.StatusInternalServerError)
		return
	}
	defer destFile.Close()

	// Copy the uploaded file to the destination file
	_, err = io.Copy(destFile, file)
	if err != nil {
		log.Println("Error saving file:", err)
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	// Retrieve location data
	location := r.FormValue("location")
	if location == "" {
		log.Println("No location provided")
		http.Error(w, "No location provided", http.StatusBadRequest)
		return
	}

	// Save location data to a file
	locationFilePath := filepath.Join(uploadDir, handler.Filename+".location.txt")
	locationFile, err := os.Create(locationFilePath)
	if err != nil {
		log.Println("Error creating location file:", err)
		http.Error(w, "Unable to create location file", http.StatusInternalServerError)
		return
	}
	defer locationFile.Close()

	_, err = locationFile.WriteString(location)
	if err != nil {
		log.Println("Error writing location data:", err)
		http.Error(w, "Unable to write location data", http.StatusInternalServerError)
		return
	}

	// Log success and send response
	log.Printf("Photo saved as: %s", filePath)
	log.Printf("Location saved as: %s", locationFilePath)
	fmt.Fprintf(w, "Photo and location received successfully\n")
	fmt.Fprintf(w, "Photo saved as: %s\n", filePath)
	fmt.Fprintf(w, "Location: %s\n", location)
}

func main() {
	// Serve static files from the current directory
	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)

	http.HandleFunc("/capture", capturePhotoAndLocation)

	// Load SSL certificates
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatalf("Error loading SSL certificates: %v", err)
	}

	// Create HTTPS server
	server := &http.Server{
		Addr: ":8443",
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
		ErrorLog: log.New(os.Stderr, "server: ", log.LstdFlags),
	}

	log.Println("Starting server on https://localhost:8443")
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("server.ListenAndServeTLS failed: %v", err)
	}
}
