package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"net/http"
	"os"
	"time"

	"dust/channel"
	"dust/model"
)

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("I am here")
}

// Endpoint to generate a Zernio OAuth link based on frontend selection
func handleGetConnectLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Enable CORS if debugging across ports
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	zernioAPIKey := os.Getenv("ZERNIO_API_KEY")
	if zernioAPIKey == "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Server missing ZERNIO_API_KEY setup"})
		return
	}

	var input map[string]string
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input["platform"] == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing platform target string"})
		return
	}

	platform := input["platform"]
	// This instructs Zernio to append platform info to your callback url parameters automatically
	callbackRedirect := fmt.Sprintf("http://localhost:8000/callback.html?platform=%s", platform)

	payload := model.ZernioLinkRequest{
		Platform:    platform,
		RedirectURL: callbackRedirect,
	}
	jsonPayload, _ := json.Marshal(payload)

	reqZernio, err := http.NewRequest("POST", "https://zernio.com", bytes.NewBuffer(jsonPayload))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed creating external client request context"})
		return
	}

	reqZernio.Header.Set("Content-Type", "application/json")
	reqZernio.Header.Set("Authorization", "Bearer "+zernioAPIKey)

	client := &http.Client{}
	resp, err := client.Do(reqZernio)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Could not contact Zernio engine core"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(resp.StatusCode)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Zernio rejection: %s", string(bodyBytes))})
		return
	}

	var zernioResp model.ZernioLinkResponse
	if err := json.Unmarshal(bodyBytes, &zernioResp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed tracking payload serialization rules"})
		return
	}

	// Ship the direct authentication link back to index.html popup engine
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"url": zernioResp.AuthURL})
}

// Unified Code Exchange Endpoint supporting Nylas (Gmail) and Zernio (Instagram/Messenger)
func handleExchange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	var req model.ExchangeRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Code == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing Authorization code"})
		return
	}

	// ROUTE A: Handle Instagram or Facebook via Zernio Flow
	if req.Platform == "instagram" || req.Platform == "facebook" {
		zernioAPIKey := os.Getenv("ZERNIO_API_KEY")

		zernioPayload := model.ZernioExchangePayload{
			Code:     req.Code,
			Platform: req.Platform,
		}
		jsonPayload, _ := json.Marshal(zernioPayload)

		reqZernio, err := http.NewRequest("POST", "https://zernio.com", bytes.NewBuffer(jsonPayload))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed creating inner client loop request"})
			return
		}

		reqZernio.Header.Set("Content-Type", "application/json")
		reqZernio.Header.Set("Authorization", "Bearer "+zernioAPIKey)

		client := &http.Client{}
		resp, err := client.Do(reqZernio)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed contacting Zernio authentication core"})
			return
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("❌ Zernio Token Error: %s\n", string(bodyBytes))
			w.WriteHeader(resp.StatusCode)
			json.NewEncoder(w).Encode(map[string]string{"error": "Zernio token validation rejected"})
			return
		}

		var zernioResp model.ZernioExchangeResponse
		if err := json.Unmarshal(bodyBytes, &zernioResp); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed parsing account structure data strings"})
			return
		}

		fmt.Printf("\n🎉 Success! Connected Social Platform Profile: %s\n", req.Platform)
		fmt.Printf("Save this Account ID to your Database: %s\n\n", zernioResp.AccountID)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"account_id": zernioResp.AccountID,
		})
		return
	}

	// ROUTE B: Original Nylas v3 Gmail Flow (Triggered when platform is empty or equals "gmail")
	nylasAPIKey := os.Getenv("NYLAS_API_KEY")
	nylasClientID := os.Getenv("NYLAS_CLIENT_ID")
	redirectURI := "http://localhost:8000/callback.html"

	payload := map[string]string{
		"client_id":     nylasClientID,
		"client_secret": nylasAPIKey,
		"grant_type":    "authorization_code",
		"code":          req.Code,
		"redirect_uri":  redirectURI,
	}
	jsonPayload, _ := json.Marshal(payload)

	nylasURL := "https://api.us.nylas.com/v3/connect/token"
	reqNylas, err := http.NewRequest("POST", nylasURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create request"})
		return
	}

	reqNylas.Header.Set("Content-Type", "application/json")
	reqNylas.Header.Set("Authorization", "Bearer "+nylasAPIKey)

	client := &http.Client{}
	resp, err := client.Do(reqNylas)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed contacting Nylas API"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ Nylas API Error: %s\n", string(bodyBytes))
		w.WriteHeader(resp.StatusCode)
		json.NewEncoder(w).Encode(map[string]string{"error": "Nylas API rejected the request"})
		return
	}

	var nylasResp model.NylasTokenResponse
	if err := json.Unmarshal(bodyBytes, &nylasResp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed parsing token payload"})
		return
	}

	fmt.Printf("\n🎉 Success! Connected Gmail User Account!\n")
	fmt.Printf("Save this Grant ID to your Database: %s\n\n", nylasResp.GrantID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"grant_id": nylasResp.GrantID,
	})
}

func main() {
	err := godotenv.Load("/home/krazygenus/Desktop/dust/.env")
	if err != nil {
		fmt.Println("Warning: Unable to load the .env file. Relying on host environment arrays.")
	}

	mux := http.NewServeMux()

	s := &http.Server{
		Addr:         ":8000",
		Handler:      mux,
		IdleTimeout:  10 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	fs := http.FileServer(http.Dir("./frontend"))
	mux.Handle("/", fs)

	mux.HandleFunc("/whatsapp/webhook", channel.WhatsAppHandler)
	mux.HandleFunc("/mail/webhook", channel.GmailHandler)

	// Hook up our two functional endpoint routing hooks
	mux.HandleFunc("/api/get-connect-link", handleGetConnectLink)
	mux.HandleFunc("/api/exchange", handleExchange)

	fmt.Println("Ready to serve at: http://localhost:8000")
	err = s.ListenAndServe()
	if err != nil {
		fmt.Println(err)
		return
	}
}
