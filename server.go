package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Response struct {
	Latency string `json:"latency"`
	Status  string `json:"status"`
}

func CheckHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading body: %v", err), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		type CheckRequest struct {
			URL string `json:"url"`
		}
		var req CheckRequest
		err = json.Unmarshal(body, &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}

		if req.URL == "" {
			http.Error(w, "No URL provided", http.StatusBadRequest)
			return
		}

		data := make(chan Response)
		go check(req.URL, data)

		type responsestruct struct {
			URL  string   `json:"url"`
			Data Response `json:"data"`
		}

		response := &responsestruct{
			URL:  req.URL,
			Data: <-data,
		}

		responsejson, _ := json.Marshal(response)
		fmt.Printf("Check Response JSON: %s\n", string(responsejson))
		w.Header().Set("Content-Type", "application/json")
		w.Write(responsejson)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading body: %v", err), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		type RegisterRequest struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		var req RegisterRequest
		err = json.Unmarshal(body, &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.Password == "" {
			http.Error(w, "No name or password provided", http.StatusBadRequest)
			return
		}
		done := make(chan bool)
		token := make(chan string)
		go CreateUser(req.Username, req.Password, done, token)
		select {
		case <-done:
			tokenString := <-token
			w.Header().Set("Content-Type", "application/json")
			response := fmt.Sprintf(`{"username":"%v","token":"%v"}`, req.Username, tokenString)
			w.Write([]byte(response))
			return
		case <-time.After(5 * time.Second):
			w.Write([]byte(fmt.Sprintf("%v", http.StatusInternalServerError)))
			return
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading body: %v", err), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		type LoginRequest struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		var req LoginRequest
		err = json.Unmarshal(body, &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.Password == "" {
			http.Error(w, "No name or password provided", http.StatusBadRequest)
			return
		}
		done := make(chan bool)
		token := make(chan string)
		go ValidateUser(req.Username, req.Password, done, token)
		select {
		case <-done:
			tokenString := <-token
			w.Header().Set("Content-Type", "application/json")
			response := fmt.Sprintf(`{"username":"%v","token":"%v"}`, req.Username, tokenString)
			w.Write([]byte(response))
			return
		case <-time.After(5 * time.Second):
			w.Write([]byte(fmt.Sprintf("%v", http.StatusInternalServerError)))
			return
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
func AddSiteHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:

		userID, err := ValidateTokenFromRequest(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading body: %v", err), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		type AddSiteRequest struct {
			URL string `json:"url"`
		}

		var req AddSiteRequest
		err = json.Unmarshal(body, &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}

		if req.URL == "" {
			http.Error(w, "No URL provided", http.StatusBadRequest)
			return
		}
		done := make(chan bool)
		defer w.Header().Set("Content-Type", "application/json")

		go AddSiteToDatabase(userID, req.URL, done)

		select {
		case <-done:
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{ "success": true }`))
			return
		case <-time.After(5 * time.Second):
			w.Write([]byte(fmt.Sprintf("%v", http.StatusInternalServerError)))
			return
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
func GetLog(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		userId, err := ValidateTokenFromRequest(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusUnauthorized)
		}
		done := make(chan bool)
		data := make(chan []URLCheckLogs)
		go GetCheckLogs(userId, done, data)
		select {
		case <-done:
			response := <-data
			responsejson, _ := json.Marshal(response)
			w.Header().Set("Content-Type", "application/json")
			w.Write(responsejson)
		case <-time.After(5 * time.Second):
			w.Write([]byte(fmt.Sprintf("%v", http.StatusInternalServerError)))
			return
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
func main() {

	_, err := ConnectDatabase()
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		return
	}
	defer DisconnectDatabase()

	fs := http.FileServer(http.Dir("./public"))

	http.Handle("/", fs)

	http.HandleFunc("/check", CheckHandler)
	http.HandleFunc("/register", RegisterHandler)
	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/addsite", AddSiteHandler)
	http.HandleFunc("/getlog", GetLog)

	go Scheduler()

	fmt.Println("Server started on http://localhost:3000")
	http.ListenAndServe(":3000", nil)
}
