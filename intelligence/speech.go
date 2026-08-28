package intelligence

import (
	"bytes"
	"context"
	"dust/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func TextToSpeech(aiResponse string) (string, error) {

	// create a dir if not exist
	filename := fmt.Sprintf("reply_%d.mp3", time.Now().Unix())
	filePath := filepath.Join("public", "replies", filename)

	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		fmt.Errorf("failed to create public directory: %v", err)
		return "", err
	}

	apiKey := os.Getenv("DEEPGRAM_API_KEY")
	fmt.Println("Deep", apiKey)
	// if apiKey == "" {
	// 	fmt.Println("DEEPGRAM_API_KEY environment variable is required")
	// 	os.Exit(1)
	// }

	ctx := context.Background()

	// Query parameters for Flux TTS REST endpoint (/v2/speak)
	baseURL := "https://api.deepgram.com/v2/speak"
	params := url.Values{}
	params.Add("model", "flux-gemma-en")
	params.Add("encoding", "mp3")
	params.Add("speed", "1")
	params.Add("expressivity", "2")

	endpoint := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Prepare payload
	reqBody, err := json.Marshal(model.SpeakRequest{Text: aiResponse})
	if err != nil {
		fmt.Printf("json.Marshal failed: %v\n", err)
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		return "", err
	}

	req.Header.Set("Authorization", "Token "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Execute HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Deepgram API returned error (status %d): %s\n", resp.StatusCode, string(body))
		return "", err
	}

	// Save the output audio file
	outFile, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return "", err
	}

	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		fmt.Printf("Failed to save audio to file: %v\n", err)
		return "", err
	}

	fmt.Printf("Audio successfully saved to %s\n", filePath)
	return filePath, nil
}
