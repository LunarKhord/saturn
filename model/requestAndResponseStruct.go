package model

// The outermost layer (The root of JSON from Webhook)
type WebhookPayload struct {
	Message Message `json:"message"`
}

// The "message" layer
type Message struct {
	ID               string   `json:"id"`
	From             string   `json:"from"`
	Text             TextBody `json:"text"`
	Type             string   `json:"type"`
	Kapso            Kapso    `json:"kapso"`
	Context          string   `json:"context"`
	Username         string   `json:"username"`
	Timestamp        string   `json:"timestamp"`
	FromUserID       string   `json:"from_user_id"`
	FromParentUserID string   `json:"from_parent_user_id"`
}

// The nested "text" layer
type TextBody struct {
	Body string `json:"body"`
}

type KapsoTranscript struct {
	Text string `json:"text"`
}

// The nested "kapso" layer
type Kapso struct {
	Origin           string          `json:"origin"`
	Status           string          `json:"status"`
	Content          string          `json:"content"`
	Transcript       KapsoTranscript `json:"transcript"`
	Direction        string          `json:"direction"`
	HasMedia         bool            `json:"has_media"`
	ProcessingStatus string          `json:"processing_status"`
}

type MessageResponseText struct {
	Body string `json:"body"`
}

type MessageResponse struct {
	MessagingProduct string              `json:"messaging_product"`
	To               string              `json:"to"`
	Type             string              `json:"type"`
	Text             MessageResponseText `json:"body"`
}

// The nested "message" object (Keep this as is)
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// The outermost payload object (Keep this as is)
type LLMPayload struct {
	Messages            []LLMMessage `json:"messages"`
	Model               string       `json:"model"`
	Temperature         float64      `json:"temperature"`
	MaxCompletionTokens int          `json:"max_completion_tokens"`
	TopP                float64      `json:"top_p"`
	Stream              bool         `json:"stream"`
	ReasoningEffort     string       `json:"reasoning_effort"`
	Stop                any          `json:"stop"`
}

// The final structured data we want to extract
type LLMContent struct {
	Intent      string `json:"intent"`
	Sentiment   string `json:"sentiment"`
	Urgency     string `json:"urgency"`
	NeedsHuman  bool   `json:"needs_human"`
	ReplyToUser string `json:"reply_to_user"`
}

// he API Response Structs
type LLMInternalPayload struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChoicesPayload struct {
	Index   int                `json:"index"`
	Message LLMInternalPayload `json:"message"`
}

type LLMResponse struct {
	ID      string           `json:"id"`
	Choices []ChoicesPayload `json:"choices"`
}

// The outermost webhook envelope
type NylasWebhookPayload struct {
	Type string      `json:"type"`
	Data WebhookData `json:"data"`
}

// The "data" wrapper
type WebhookData struct {
	Object MessageObject `json:"object"`
}

type MessageObject struct {
	Body    string            `json:"body"`
	Folders []string          `json:"folders"`
	From    []GmailFromObject `json:"from"`
	GrantID string            `json:"grant_id"`
	Subject string            `json:"subject"`
	ID      string            `json:"id"`
}

// The sender details
type GmailFromObject struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type NylasTokenPayload struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
}

// ExchangeRequest matches the incoming payload structure from callback.html
type ExchangeRequest struct {
	Code     string `json:"code"`
	Platform string `json:"platform"` // "gmail", "instagram", or "facebook"
}

// NylasTokenResponse captures the response from Nylas v3
type NylasTokenResponse struct {
	GrantID string `json:"grant_id"`
}

// ZernioLinkRequest models the request structure sent to Zernio
type ZernioLinkRequest struct {
	Platform    string `json:"platform"`
	RedirectURL string `json:"redirect_url"`
}

// ZernioLinkResponse models the response payload returned by Zernio
type ZernioLinkResponse struct {
	AuthURL string `json:"authUrl"`
}

// ZernioExchangePayload models the request structure to exchange an auth code via Zernio
type ZernioExchangePayload struct {
	Code     string `json:"code"`
	Platform string `json:"platform"`
}

// ZernioExchangeResponse models the successful profile creation return structure
type ZernioExchangeResponse struct {
	AccountID string `json:"account_id"`
}

// Nylas Email Response model
type NylasToEmail struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}
type NylasEmailPayload struct {
	Subject string         `json:"subject"`
	Body    string         `json:"body"`
	To      []NylasToEmail `json:"to"`
}

// Utilized by DeepGram

type SpeakRequest struct {
	Text string `json:"text"`
}
