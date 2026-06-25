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

type set struct {
	SetNum       string `json:"set_num"`
	Name         string `json:"name"`
	SetImgUrl    string `json:"set_img_url"`
	BricklinkUrl string `json:"bricklink_url"`
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
	resultTemplate = template.Must(template.ParseFiles("static/query.html"))
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

func requestSetsContainingPiece(pieceId string, colorId int) []set {
	client := &http.Client{}

	req, _ := http.NewRequest("GET", fmt.Sprintf("%sparts/%s/colors/%d/sets/?page_size=10000&ordering=set_num", REBRICKABLE_API_BASE_URL, pieceId, colorId), nil)
	req.Header.Add("Authorization", fmt.Sprintf("key %s", REBRICKABLE_API_KEY))

	respJson, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching sets for piece %s and color %d: %v", pieceId, colorId, err)
		return nil
	}
	defer respJson.Body.Close()

	body, err := io.ReadAll(respJson.Body)
	if err != nil {
		log.Printf("Error reading response body for piece %s and color %d: %v", pieceId, colorId, err)
		return nil
	}

	var resp struct {
		Results []set `json:"results"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		log.Printf("unmarshal error: %v", err)
	}
	sets := resp.Results

	for i := range sets {
		sets[i].BricklinkUrl = fmt.Sprintf("https://www.bricklink.com/v2/catalog/catalogitem.page?S=%s", sets[i].SetNum)
	}

	return sets
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
	if len(superset) == 0 {
		return nil
	} else if len(superset) == 1 {
		return superset[0]
	}

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
		setsWithPiece := requestSetsContainingPiece(piece_ids[i], colorIdMap[colors[i]])
		sets = append(sets, setsWithPiece)
	}

	intersection := setIntersection(sets)
	fmt.Printf("Intersection: %v\n", intersection)

	resultTemplate.Execute(w, intersection)

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
