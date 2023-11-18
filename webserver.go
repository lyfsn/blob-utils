package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/urfave/cli"
)

/*
Some examples:

Empty: http://localhost:3333/stream/html?slot=125751
Hello world: http://localhost:3333/stream/html?slot=125753
Html: http://localhost:3333/stream/html?slot=125754
Video: // http://localhost:3333/stream/html?slot=125754

Image: http://localhost:3333/stream/html?slot=129252
	   https://blobscan.com/tx/0xd148b4fad7e559687855b87c78d13a81c9777dcb06313b641f8ffc72177c7c7b
*/

func streamHtmlHandler(w http.ResponseWriter, r *http.Request) {
	//serveFile(w, r, []string{"126296-0.bin", "126296-1.bin", "126296-2.bin"}, "text/html")
	serveBlob(w, r, "text/html")
}

func streamVideoHandler(w http.ResponseWriter, r *http.Request) {
	serveFile(w, r, []string{"test_files/devconnect.mp4"}, "video/mp4")
}

func streamImageHandler(w http.ResponseWriter, r *http.Request) {
	serveFile(w, r, []string{"test_files/DoD.jpg"}, "image/jpeg")
}

func serveFile(w http.ResponseWriter, r *http.Request, filePaths []string, contentType string) {
	// Set the content type
	w.Header().Set("Content-Type", contentType)
	w.(http.Flusher).Flush()

	for _, filePath := range filePaths {
		// Open the file
		file, err := os.Open(filePath)
		if err != nil {
			http.Error(w, "Error opening the file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// Stream the file directly to the response writer
		_, err = io.Copy(w, file)
		if err != nil {
			fmt.Println("Error streaming file:", err)
			return
		}

		w.(http.Flusher).Flush()
	}
}

func serveBlob(w http.ResponseWriter, r *http.Request, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.(http.Flusher).Flush()

	params, _ := url.ParseQuery(r.URL.RawQuery)
	fmt.Println("params", params)

	addr := "http://10.128.0.8:5052"
	slot := params.Get("slot")

	blobChannel := make(chan []byte)
	doneChannel := make(chan struct{})

	go GetMultiPartBlob(blobChannel, addr, slot)

	fmt.Println("Waiting for blobChannel...")
	for {
		select {
		case result, ok := <-blobChannel:
			if !ok {
				fmt.Println("blobChannel is closed, break out of the loop")
				return
			}
			fmt.Println("Blob received through channel")

			w.Write(result)
			w.(http.Flusher).Flush()

		case <-doneChannel:
			// doneChannel is closed, exit the loop
			fmt.Println("All results received.")
			return
		}
	}
}

func WebserverApp(cliCtx *cli.Context) error {

	http.HandleFunc("/stream/video", streamVideoHandler)
	http.HandleFunc("/stream/image", streamImageHandler)
	http.HandleFunc("/stream/html", streamHtmlHandler)

	// addr := cliCtx.String(WebserverRPCURLFlag.Name)

	port := 3333
	fmt.Printf("Server listening on :%d\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
	return nil
}
