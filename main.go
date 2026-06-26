package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

const MAX_INT = int(^uint(0) >> 1)
const REBRICKABLE_API_BASE_URL = "https://rebrickable.com/api/v3/lego/"
var REBRICKABLE_API_KEY string

var resultTemplate *template.Template
var errorTemplate *template.Template

type set struct {
	SetNum       string `json:"set_num"`
	Name         string `json:"name"`
	SetImgUrl    string `json:"set_img_url"`
	BricklinkUrl string `json:"bricklink_url"`
}

type httpError struct {
	StatusCode string `json:"status_code"`
	Title      string `json:"title"`
	Message    string `json:"message"`
}

type resultPage struct {
	Results []set `json:"results"`
	Comment string `json:"comment,omitempty"`
}

var colorIdMap = make(map[string]int)

func initSecrets() {
	// Check if environment variable is set
	if apiKey := os.Getenv("REBRICKABLE_API_KEY"); apiKey != "" {
		fmt.Println("Using REBRICKABLE_API_KEY from environment variable")
		REBRICKABLE_API_KEY = apiKey
		return
	} else {
		// Else, read from .env file
		fmt.Println("Using REBRICKABLE_API_KEY from .env file")
		data, err := os.ReadFile(".env")
		if err != nil {
			log.Fatal("Error reading .env file:", err)
		}
		REBRICKABLE_API_KEY = strings.TrimSpace(strings.Split(string(data), "=")[1])
	}
}

func initTemplate() {
	resultTemplate = template.Must(template.ParseFiles("static/templates/query.html"))
	errorTemplate = template.Must(template.ParseFiles("static/templates/error.html"))
}

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

func reverseColorLookup(colorId int) string {
	for name, id := range colorIdMap {
		if id == colorId {
			return name
		}
	}
	return fmt.Sprintf("Unknown Color ID: %d", colorId)
}

func requestSetsContainingPiece(pieceId string, colorId int) ([]set, *httpError) {
	client := &http.Client{}

	req, _ := http.NewRequest("GET", fmt.Sprintf("%sparts/%s/colors/%d/sets/?page_size=10000&ordering=set_num", REBRICKABLE_API_BASE_URL, pieceId, colorId), nil)
	req.Header.Add("Authorization", fmt.Sprintf("key %s", REBRICKABLE_API_KEY))

	respJson, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching sets for piece %s and color %d: %v", pieceId, colorId, err)
		return nil, &httpError{StatusCode: "500", Title: "Internal Server Error", Message: "An internal server error occurred."}
	}
	defer respJson.Body.Close()

	body, err := io.ReadAll(respJson.Body)
	if err != nil {
		log.Printf("Error reading response body for piece %s and color %d: %v", pieceId, colorId, err)
		return nil, &httpError{StatusCode: "500", Title: "Internal Server Error", Message: "An internal server error occurred."}
	}

	if respJson.StatusCode == 401 {
		return nil, &httpError{StatusCode: "401", Title: "Unauthorized", Message: "It seems that your Rebrickable API key is invalid or missing. Please check your API key and try again."}
	} else if respJson.StatusCode == 404 {
		return nil, &httpError{StatusCode: "404", Title: "Not Found", Message: fmt.Sprint("Piece Num '", pieceId, "' in color '", reverseColorLookup(colorId), "' not found in the database.")}
	} else if respJson.StatusCode != 200 {
		return nil, &httpError{StatusCode: "500", Title: "Internal Server Error", Message: "An internal server error occurred."}
	}

	var resp struct {
		Results []set `json:"results"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		log.Printf("unmarshal error: %v", err)
	}
	sets := resp.Results

	if len(sets) == 0 {
		return nil, &httpError{StatusCode: "404", Title: "Not Found", Message: fmt.Sprint("Piece Num '", pieceId, "' in color '", reverseColorLookup(colorId), "' not found in any sets in the database.")}
	}

	for i := range sets {
		sets[i].BricklinkUrl = fmt.Sprintf("https://www.bricklink.com/v2/catalog/catalogitem.page?S=%s", sets[i].SetNum)
	}

	return sets, nil
}

// Binary search to find if a setNum existsin in a list of sets
func containsSetNum(sets []set, setNum string) bool {
	var l int = 0
	var r int = len(sets) - 1
	for l <= r {
		var m int = l + ((r - l) / 2)
		var currentSetNum string = sets[m].SetNum
		if currentSetNum < setNum {
			l = m + 1
		} else if currentSetNum > setNum {
			r = m - 1
		} else {
			return true
		}
	} 

	return false
}

// Finds which sets appear in all provided set lists
func setIntersection(superset [][]set) []set {
	// Find the shortest of the lists of sets
	min_len := MAX_INT
	min_len_index := 0
	for i := range superset {
		if len(superset[i]) < min_len {
			min_len = len(superset[i])
			min_len_index = i
		}
	}

	// Filter out sets with 500XXXX numbers
	var legoSetRe = regexp.MustCompile(`\b500\d{4}(?:-\d+)?\b`)

	var intersection []set
	for i := range superset[min_len_index] {
		if legoSetRe.MatchString(superset[min_len_index][i].SetNum) {
			continue
		}
		setNum := superset[min_len_index][i].SetNum
		foundInAll := true
		for j := range superset {
			if j == min_len_index {
				continue
			}
			if !containsSetNum(superset[j], setNum) {
				foundInAll = false
				break
			}
		}
		if foundInAll {
			fmt.Printf("Set %s contains all pieces\n", setNum)
			intersection = append(intersection, superset[min_len_index][i])
		}
	}

	return intersection
}

func queryHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v\n", err)
		return
	}

	fmt.Printf("\nRequest Received\n")
	piece_ids := r.Form["piece_ids"]
	colors := r.Form["colors"]

	var sets [][]set
	for i := range piece_ids {
		fmt.Printf("Piece ID: %s, Color: %s, Color ID: %d\n", piece_ids[i], colors[i], colorIdMap[colors[i]])
		setsWithPiece, err := requestSetsContainingPiece(piece_ids[i], colorIdMap[colors[i]])
		if err != nil {
			errorTemplate.Execute(w, err)
			return
		}
		sets = append(sets, setsWithPiece)
	}

	intersection := []set{}
	if len(sets) == 0 {
		errorTemplate.Execute(w, &httpError{StatusCode: "400", Title: "Bad Request", Message: "No valid piece IDs and colors were provided. Please check your input and try again."})
		return
	} else if len(sets) == 1 {
		intersection = sets[0]
	} else {
		intersection = setIntersection(sets)
		fmt.Printf("Intersection: %v\n", intersection)
	}

	if len(intersection) == 0 {
		resultTemplate.Execute(w, resultPage{Results: []set{}, Comment: "No sets found that contain all specified pieces."})
		return
	}

	resultTemplate.Execute(w, resultPage{Results: intersection, Comment: fmt.Sprintf("Found %d sets that contain all specified pieces.", len(intersection))})
}

func main() {
	initSecrets()
	initTemplate()

	if err := initializeColorMap(); err != nil {
		log.Fatal(err)
	}

	fileserver := http.FileServer(http.Dir("./static"))
	http.Handle("/", fileserver)
	http.HandleFunc("/query", queryHandler)
	fmt.Printf("Starting server at http://0.0.0.0:8090\n")

	if err := http.ListenAndServe(":8090", nil); err != nil {
		log.Fatal(err)
	}
}
