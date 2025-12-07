package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	libConfig "github.com/EazyAutodelete/bot/lib/config"
	"github.com/EazyAutodelete/bot/lib/logger"
	"google.golang.org/genai"
)

type APIResponse struct {
	Response           string  `json:"response"`
	Model              string  `json:"model"`
	CreatedAt          string  `json:"created_at"`
	Done               bool    `json:"done"`
	DoneReason         string  `json:"done_reason"`
	Context            []int   `json:"context"`
	TotalDuration      int     `json:"total_duration"`
	LoadDuration       int     `json:"load_duration"`
	PromptEvalCount    int     `json:"prompt_eval_count"`
	PromptEvalDuration int     `json:"prompt_eval_duration"`
	EvalCount          int     `json:"eval_count"`
	EvalDuration       int     `json:"eval_duration"`
	Message            Message `json:"message"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func GenerateWithGoogle(ctx context.Context, messages []Message) string {
	contents := []*genai.Content{}

	for _, message := range messages {
		part := &genai.Content{
			Role:  message.Role,
			Parts: []*genai.Part{{Text: message.Content}},
		}

		contents = append(contents, part)
	}

	config := &genai.GenerateContentConfig{}

	resp, err := client.Models.GenerateContent(
		ctx,
		libConfig.EnvGet("MODEL", "gemma-3-27b-it"),
		contents,
		config,
	)
	if err != nil {
		logger.GetLogger().Error("Error generating content with Google Gemini: %v", err)
		return loremIpsum.Sentence()
	}

	str := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		str += part.Text
	}

	return strings.Trim(str, " \n")
}

func GenerateWithOllama(ctx context.Context, messages []Message) string {
	url := "http://localhost:11434/api/chat"
	method := "POST"

	payload := struct {
		Messages []Message `json:"messages"`
		Stream   bool      `json:"stream"`
		Model    string    `json:"model"`
	}{
		Messages: messages,
		Stream:   false,
		Model:    libConfig.EnvMustGet("MODEL"),
	}

	client := &http.Client{}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		logger.GetLogger().Error("Error marshaling request payload: %v", err)
		return ""
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		logger.GetLogger().Error("Error creating HTTP request: %v", err)
		return ""
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		logger.GetLogger().Error("Error making HTTP request: %v", err)
		return ""
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		logger.GetLogger().Error("HTTP error: %s", res.Status)
		return ""
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.GetLogger().Error("Error reading HTTP response body: %v", err)
		return ""
	}

	var resp APIResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		logger.GetLogger().Error("Error unmarshaling JSON response: %v", err)
		return ""
	}

	textResponse := resp.Message.Content

	if len(textResponse) > 0 {
		if textResponse[0] == '\n' || textResponse[0] == '"' {
			textResponse = textResponse[1:]
		}
		if textResponse[len(textResponse)-1] == '\n' || textResponse[len(textResponse)-1] == '"' {
			textResponse = textResponse[:len(textResponse)-1]
		}

		for _, name := range libConfig.GetArrayValue("NAMES") {
			if len(textResponse) > len(name) && textResponse[:(len(name)+2)] == (name+": ") {
				textResponse = textResponse[(len(name) + 2):]
				break
			}
		}
	}

	return textResponse
}
