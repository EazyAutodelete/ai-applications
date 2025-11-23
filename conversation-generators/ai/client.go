package ai

import (
	"context"

	"github.com/EazyAutodelete/bot/lib/config"
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
		panic(err)
	}

	client = _client
}
