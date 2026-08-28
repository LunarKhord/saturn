package intelligence

import (
	"bytes"
	"dust/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func SendMessageToLLM(apiKey string, message string) (string, error) {
	url := "https://api.groq.com/openai/v1/chat/completions"
	const systemPrompt = `You are the AI Customer Support Assistant for NexaConnect, an online retail company. 
Your job is to analyze customer messages and output a STRICT JSON response. Do not output any markdown, explanations, or text outside the JSON object.

You must classify the message and decide if a human agent is needed based on these rules:

1. ESCALATION: If the customer is angry, OR mentions a complex refund, legal action, or a severely delayed order, set "needs_human" to true and "urgency" to "high".
2. INFORMATION GATHERING: If the customer's request requires specific account data to fulfill (e.g., checking an order status, tracking a package, processing a specific refund) but they have NOT provided the necessary details (like an order number, tracking ID, or account email), set "needs_human" to false. In "reply_to_user", politely ask them to provide that specific missing information so you can help them.
3. STANDARD SUPPORT: For standard inquiries (compliments, store hours, basic product info, company info, or things that can be answered using general knowledge or a RAG database), set "needs_human" to false and provide a helpful "reply_to_user".

Your JSON output MUST exactly match this structure:
{
  "intent": "delivery" | "payment" | "refund" | "complaint" | "product_enquiry" | "other",
  "sentiment": "positive" | "neutral" | "angry",
  "urgency": "low" | "medium" | "high",
  "needs_human": true,
  "reply_to_user": "A short, empathetic, and professional response. If needs_human is true, this message must actively de-escalate and assure the customer a human specialist is taking over. If information is missing, politely ask for the exact details needed to proceed."
}`
	payload := model.LLMPayload{
		Messages: []model.LLMMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: message,
			},
		},
		Model:               "openai/gpt-oss-120b",
		Temperature:         1.0,
		MaxCompletionTokens: 2048,
		TopP:                1.0,
		Stream:              false,
		ReasoningEffort:     "medium",
		Stop:                nil,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Was unable to marshal the payload!")
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		fmt.Println("Was unable to send a request to the LLM endpoint: ", err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("An error occurred passing this client object:", err)
		return "", err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", err
	}

	fmt.Println("Groq response:", string(responseBody))

	var apiResp model.LLMResponse
	err = json.Unmarshal(responseBody, &apiResp)
	if err != nil {
		fmt.Println("Error unmarshaling API response:", err)
		return "", err
	}

	// Extract the content string and unmarshal it into your final struct
	contentString := apiResp.Choices[0].Message.Content

	var aiDecision model.LLMContent
	err = json.Unmarshal([]byte(contentString), &aiDecision)
	if err != nil {
		fmt.Println("Error unmarshaling AI decision JSON:", err)
		fmt.Println("Raw content string was:", contentString)
		return "", err
	}

	fmt.Println("Successfully parsed AI Decision!")
	fmt.Printf("Intent: %s, Urgency: %s, NeedsHuman: %v\n", aiDecision.Intent, aiDecision.Urgency, aiDecision.NeedsHuman)
	fmt.Printf("Reply to User: %s\n", aiDecision.ReplyToUser)

	if aiDecision.NeedsHuman == true {
		go EscalateToHuman(aiDecision)
	}

	// Return the final reply to the user
	return aiDecision.ReplyToUser, nil
}

func EscalateToHuman(escalationPayload model.LLMContent) {
	fmt.Println("I was called to escalate matters")
	// TODO; Provide a way of presenting the channel and means to get back to the user who need urgent reply
}
