package channel

import (
	"bytes"
	"dust/intelligence"
	"dust/model"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"time"
	//"dust/bucket"
	"dust/utils"
)

func WhatsAppHandler(w http.ResponseWriter, r *http.Request) {
	var from string
	var body string
	var messageType string
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Warning: Unable to load the .env file. Relying on host environment arrays.")
	}
	KAPSO_API_KEY := os.Getenv("KAPSO_API_KEY")
	GROQ_API_KEY := os.Getenv("GROQ_API_KEY")
	fmt.Println("GROQ_API_KEY", GROQ_API_KEY)

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Read and parse the body
	var webHookPayload model.WebhookPayload
	d, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		fmt.Println("Error reading payload:", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	fmt.Println("The raw webhook:", string(d))
	err = json.Unmarshal(d, &webHookPayload)
	if err != nil {
		fmt.Println("Error unmarshaling:", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Extract the data we need BEFORE the goroutine starts
	fmt.Println("The payload of whatsapp:", webHookPayload)
	if webHookPayload.Message.Type == "audio" {
		from = webHookPayload.Message.From
		body = webHookPayload.Message.Kapso.Transcript.Text
		messageType = webHookPayload.Message.Type

	} else {
		from = webHookPayload.Message.From
		body = webHookPayload.Message.Text.Body
		messageType = webHookPayload.Message.Type
	}

	fmt.Printf("Received message from %s: %s\n", from, body, messageType)

	// (This prepares the response, but doesn't close it yet
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

	// Offload the heavy lifting to a background Goroutine
	// We pass 'from' and 'body' as arguments to avoid closure variable capture bugs
	go func(phone string, msg string, messageType string) {
		fmt.Println("Background worker started for:", phone)

		// Note: Make sure your function returns (string, error) in that order!
		response, err := intelligence.SendMessageToLLM(GROQ_API_KEY, msg)
		if err != nil {
			fmt.Println("LLM Error:", err)
			return
		}

		fmt.Println("LLM Response received:", response)

		if messageType == "audio" {
			// TODO: Get the link to the uploaded Opus extension file.
			mp3Path, err := intelligence.TextToSpeech(response)
			if err != nil {
				fmt.Println("[WhatsApp]: Was unable to retrieve the link to speech", err)
				return
			}
			oggFileURL, err := utils.ConvertMp3ToOGG(mp3Path)
			if err != nil {
				fmt.Println("[WhatsApp]: Unable to receive ogg bytes after passing in mp3 path:", err)
				return
			}
			//uploadURL, err := bucket.UploadVoiceToSupabase(oggBytes)
			phoneNumberID := "597907523413541"
			mediaID, err := UploadMediaToKapso(phoneNumberID, KAPSO_API_KEY, oggFileURL)
			if err != nil {
				fmt.Println("[WhatsApp]: Unable to upload file to Meta!!", err)
				return
			}
			err = SendWhatsAppVoiceNoteByID(KAPSO_API_KEY, phone, mediaID)
			if err != nil {
				fmt.Println("Failed to send WhatsApp reply:", err)
			} else {
				fmt.Println("Successfully replied to user!")
			}
		} else {
			// Send the reply back to WhatsApp
			err = SendWhatsAppMessage(KAPSO_API_KEY, phone, response)
			if err != nil {
				fmt.Println("Failed to send WhatsApp reply:", err)
			} else {
				fmt.Println("Successfully replied to user!")
			}
		}
	}(from, body, messageType)
}

// This sends a Message to WhatsApp using text
func SendWhatsAppMessage(apiKey, phoneNumber, messageBody string) error {
	phoneNumberID := "597907523413541"

	payload := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                phoneNumber,
		"type":              "text",
		"text": map[string]string{
			"body": messageBody,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := "https://api.kapso.ai/meta/whatsapp/v24.0/" +
		phoneNumberID + "/messages"

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("kapso api error %s: %s",
			resp.Status, string(responseBody))
	}

	fmt.Println("kapso response:", string(responseBody))
	return nil
}

// SendWhatsAppVoiceNoteByID sends a native inline voice note using a Meta Media ID
func SendWhatsAppVoiceNoteByID(kapsoAPIKey, phoneNumber, mediaID string) error {

	phoneNumberID := "597907523413541"

	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                phoneNumber,
		"type":              "audio",
		"audio": map[string]any{
			"id":    mediaID, // 🚨 USING ID INSTEAD OF LINK
			"voice": true,    // 🚨 THIS MAKES IT A NATIVE VOICE NOTE
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	url := "https://api.kapso.ai/meta/whatsapp/v24.0/" + phoneNumberID + "/messages"

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("X-API-Key", kapsoAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("kapso api error %s: %s", resp.Status, string(responseBody))
	}

	fmt.Println("✅ Kapso response (Native Voice Note Sent):", string(responseBody))
	return nil
}

// UploadMediaToKapso uploads the .ogg file to Meta via Kapso and returns the Media ID
func UploadMediaToKapso(phoneNumberID, kapsoAPIKey, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Create a multipart form for the file upload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 🚨 FIX: Meta requires the exact Content-Type 'audio/ogg' for the file part.
	// CreateFormFile defaults to 'application/octet-stream', which Meta rejects.
	// We must use CreatePart to set the custom MIME header.
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filepath.Base(filePath)))
	h.Set("Content-Type", "audio/ogg") // 🚨 THIS IS THE MAGIC LINE

	part, err := writer.CreatePart(h)
	if err != nil {
		return "", fmt.Errorf("failed to create multipart part: %v", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file data: %v", err)
	}

	// WhatsApp requires BOTH 'messaging_product' and 'type' fields for media uploads
	writer.WriteField("messaging_product", "whatsapp")
	writer.WriteField("type", "audio/ogg")
	writer.Close()

	// Kapso's media upload endpoint
	url := fmt.Sprintf("https://api.kapso.ai/meta/whatsapp/v24.0/%s/media", phoneNumberID)

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("X-API-Key", kapsoAPIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Kapso media upload failed %s: %s", resp.Status, string(respBody))
	}

	// Parse the ID from the response: {"id": "123456789"}
	var result map[string]any
	json.Unmarshal(respBody, &result)

	mediaID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("could not find media id in response: %s", string(respBody))
	}

	fmt.Println("✅ Media uploaded successfully. Media ID:", mediaID)
	return mediaID, nil
}
