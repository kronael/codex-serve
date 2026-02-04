package main

// Message represents a chat message with role and content
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaChatRequest represents Ollama's chat API request format
type OllamaChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

// OllamaChatResponse represents Ollama's chat API response format
type OllamaChatResponse struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`
}

// OpenAIChatRequest represents OpenAI's chat API request format
type OpenAIChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

// OpenAIChatResponse represents OpenAI's chat API response format
type OpenAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int     `json:"index"`
		Message Message `json:"message"`
		Finish  string  `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

// AnthropicMessageRequest represents Anthropic's messages API request format
type AnthropicMessageRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
	Stream    bool      `json:"stream,omitempty"`
}

// AnthropicMessageResponse represents Anthropic's messages API response format
type AnthropicMessageResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}
