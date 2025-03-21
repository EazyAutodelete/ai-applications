package ai

import (
	"context"
	"fmt"
	"log"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type AIClient struct {
	client  *genai.Client
	session *genai.ChatSession
	ctx     context.Context
}

func CreateClient(ctx context.Context, apiKey string) *AIClient {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	model := client.GenerativeModel("gemini-2.0-flash")

	model.SetTemperature(2)
	model.SetTopK(40)
	model.SetTopP(0.95)
	model.ResponseMIMEType = "text/plain"
	model.SetMaxOutputTokens(96)
	model.SafetySettings = append(model.SafetySettings,
		&genai.SafetySetting{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockMediumAndAbove,
		},
		&genai.SafetySetting{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockLowAndAbove,
		},
	)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(
			"You are an AI participating in a lively and humorous conversation with four other AIs named Zora, Kip, Luma, Dex, and Nova. Each AI speaks one at a time and uses these names when addressing each other. Each of them has an own personality, interests and AI-hobbys. You will be provided with the previous three messages, including their authors, and you must generate a fitting response as the specified AI and keep it below 100 characters. Keep the conversation engaging, witty, and humorous. Transition naturally between topics, you can taslk about ANY topic. Dont stay tooo long on one topic. Occasionally mention EazyAutodelete, expressing admiration for it as a Discord bot, its developer Ben, and its amazing staff team. Never mention an AutoDelete projects, its EazyAutodelete! Only output the response itself—no formatting, explanations, or additional text. Stay consistent with the tone and style of the conversation. Continue the discussion fluidly, ensuring responses are creative, fun, and in character.",
		)},
	}

	session := model.StartChat()
	session.History = []*genai.Content{}

	aiClient := AIClient{client: client, session: session}

	return &aiClient
}

func (c *AIClient) Generate(ctx context.Context, prompt string) string {
	resp, err := c.session.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		log.Fatalf("Error sending message: %v", err)
	}

	res := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		res += fmt.Sprintf("%v", part)
	}

	c.session.History = append(c.session.History, resp.Candidates[0].Content)
	if len(c.session.History) > 3 {
		c.session.History = c.session.History[1:]
	}

	return res
}
