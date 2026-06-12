package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type Choice struct {
	Message Message `json:"message"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  "llama-3.1-8b-instant", // standard and fast Groq model
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("groq api key is empty")
	}

	reqBody := ChatRequest{
		Model: c.model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.5,
		MaxTokens:   250, // Keep responses short and tokens low
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshalling groq request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error performing request to groq: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("groq returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("error decoding groq response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from groq")
	}

	// Log usage details for optimization monitoring
	fmt.Printf("[Groq Token Usage] Prompt: %d, Completion: %d, Total: %d\n",
		chatResp.Usage.PromptTokens,
		chatResp.Usage.CompletionTokens,
		chatResp.Usage.TotalTokens,
	)

	return chatResp.Choices[0].Message.Content, nil
}

// CondenseHistory requests Groq to condense a chat history into a brief executive summary to fit inside a sliding window.
func (c *Client) CondenseHistory(ctx context.Context, currentHistory string, newMessages string) (string, error) {
	systemPrompt := `Eres un sintetizador de historial de chat. Tu objetivo es resumir y condensar el historial de chat actual combinándolo con los nuevos mensajes, guardando únicamente los hechos, decisiones y estados importantes (como deudas de alquiler, asignación de tareas, hábitos y metas).
Genera un resumen ejecutivo en español de máximo 150 palabras. Mantén la estructura muy limpia y concisa.`

	userPrompt := fmt.Sprintf("Historial actual:\n%s\n\nNuevos mensajes:\n%s", currentHistory, newMessages)

	return c.Chat(ctx, systemPrompt, userPrompt)
}
