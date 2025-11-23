package ai

import (
	"context"

	"github.com/EazyAutodelete/bot/lib/config"
	"github.com/EazyAutodelete/bot/lib/logger"
	"google.golang.org/genai"
	"gopkg.in/loremipsum.v1"
)

var (
	client     *genai.Client
	loremIpsum *loremipsum.LoremIpsum
)

func CreateClient() {
	loremIpsum = loremipsum.New()

	ctx := context.Background()

	_client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: config.EnvGet("AI_TOKEN", ""),
	})
	if err != nil {
		logger.GetLogger().Fatal("Error creating AI client: %v", err)
		panic(err)
	}

	client = _client
}
