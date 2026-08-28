package channel

import (
	"bytes"
	"dust/intelligence"
	"dust/model"
	"dust/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func GmailHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// READ AND PARSE IN THE MAIN THREAD
	d, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		fmt.Println("Error reading payload:", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var emailPayload model.NylasWebhookPayload
	err = json.Unmarshal(d, &emailPayload)
	if err != nil {
		fmt.Println("Was unable to unmarshal JSON:", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// ACKNOWLEDGE IMMEDIATELY
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

	// Extract the fields we need
	msg := emailPayload.Data.Object

	// Safety check: Only process if it's actually in the INBOX
	isInbox := false
	for _, folder := range msg.Folders {
		if folder == "INBOX" {
			isInbox = true
			break
		}
	}

	if !isInbox {
		fmt.Println("Ignoring non-INBOX message (e.g., SENT or DRAFT)")
		return
	}

	// Prevent panic if From array is somehow empty
	fromName := "Unknown"
	fromEmail := "unknown@example.com"
	if len(msg.From) > 0 {
		fromName = msg.From[0].Name
		fromEmail = msg.From[0].Email
	}

	go func(grantID, fName, fEmail, subject, body string) {
		GROQ_API_KEY := os.Getenv("GROQ_API_KEY")

		// Clean the HTML
		cleanedBody := utils.CleanHTMLToText(body)
		fmt.Println(cleanedBody)

		// Call the LLM
		aiReply, err := intelligence.SendMessageToLLM(GROQ_API_KEY, cleanedBody)
		if err != nil {
			fmt.Println("AI Router failed:", err)
			return
		}

		fmt.Println(" AI Decision Received Successfully!")
		fmt.Println("Reply to send:", aiReply)

		// TODO: Next step is to check if it needs human escalation, or just send the reply via Nylas!
		go SendNylasEmail(grantID, fName, fEmail, "Re: "+subject, aiReply)

	}(msg.GrantID, fromName, fromEmail, msg.Subject, msg.Body)
}

func SendNylasEmail(grantID, toName, toMail, subject, body string) error {
	nylasAPIKey := os.Getenv("NYLAS_API_KEY")

	fmt.Println("[DEBUG]: SendNylasEmail was called!")
	fmt.Println("[DEBUG]: Grant ID is:", grantID)
	fmt.Println("[DEBUG]: API Key length is:", len(nylasAPIKey))

	if len(nylasAPIKey) == 0 {
		return fmt.Errorf("NYLAS_API_KEY is empty! Check your .env file.")
	}

	// The URL with the specific Grant ID
	url := fmt.Sprintf("https://api.us.nylas.com/v3/grants/%s/messages/send", grantID)

	// Build the payload
	payload := model.NylasEmailPayload{
		Subject: subject,
		Body:    body,
		To: []model.NylasToEmail{
			{
				Name:  toName,
				Email: toMail,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+nylasAPIKey)

	// ADD A 10-SECOND TIMEOUT so it never hangs forever!
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	fmt.Println("DEBUG: Sending HTTP request to Nylas...")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("HTTP Request Error:", err)
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// PRINT THE EXACT RESPONSE FROM NYLAS
	fmt.Println("DEBUG: Nylas Status Code:", resp.StatusCode)
	fmt.Println("DEBUG: Nylas Response Body:", string(respBody))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Nylas API error %s: %s", resp.Status, string(respBody))
	}

	fmt.Println("Email sent successfully via Nylas!")
	return nil
}
