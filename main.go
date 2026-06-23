package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

var colorIdMap = make(map[string]int)

func initializeColorMap() error {
	data, err := os.ReadFile("static/assets/colors.json")
	if err != nil {
		return fmt.Errorf("failed to read colors file: %w", err)
	}

	var payload struct {
		Colors []struct {
			Name string `json:"name"`
			Id   int    `json:"id"`
		} `json:"colors"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("failed to parse colors file: %w", err)
	}

	for _, result := range payload.Colors {
		if result.Name != "" {
			colorIdMap[result.Name] = result.Id
		}
	}

	return nil
}

func queryHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v\n", err)
		return
	}

	fmt.Printf("\nRequest Received\n")
	piece_ids := r.Form["piece_ids"]
	colors := r.Form["colors"]

	for i := range piece_ids {
		fmt.Printf("Piece ID: %s, Color: %s, Color ID: %d\n", piece_ids[i], colors[i], colorIdMap[colors[i]])
	}

}

func main() {
	if err := initializeColorMap(); err != nil {
		log.Fatal(err)
	}

	fileserver := http.FileServer(http.Dir("./static"))
	http.Handle("/", fileserver)
	http.HandleFunc("/query", queryHandler)
	fmt.Printf("Starting server at http://localhost:8090\n")

	if err := http.ListenAndServe(":8090", nil); err != nil {
		log.Fatal(err)
	}
}
