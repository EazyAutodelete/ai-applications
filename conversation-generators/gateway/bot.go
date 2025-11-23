package gateway

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/EazyAutodelete/bot/lib/config"
	"github.com/EazyAutodelete/bot/lib/logger"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/eazyautodelete/ai-users/ai"
)

type Channel interface {
	GetID() string
	AddMessage(message ai.Message)
}

func Bot(channels []Channel) {
	client, err := disgo.New(
		config.EnvMustGet("DISCORD_TOKEN_4"),
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(gateway.IntentGuildMessages, gateway.IntentMessageContent),
		),

		bot.WithEventListenerFunc(func(e *events.MessageCreate) {
			if e.Message.Content == "" || len(e.Message.Content) < 3 {
				return
			}

			// return if not starts with !ai
			if e.Message.Content[:3] != "!ai" {
				return
			}

			// has role
			found := false
			for _, role := range e.Message.Member.RoleIDs {
				if role.String() == config.EnvMustGet("STAFF_ROLE") {
					found = true
					break
				}
			}

			if found {
				var name string
				if e.Message.Member.Nick != nil {
					name = *e.Message.Member.Nick
				} else {
					name = e.Message.Author.EffectiveName()
				}

				// Find the channel this message belongs to and add to its history
				channelID := e.Message.ChannelID.String()
				for _, channel := range channels {
					if channel.GetID() == channelID {
						channel.AddMessage(ai.Message{
							Role:    "user",
							Content: name + ": " + e.Message.Content[4:],
						})
						break
					}
				}
			}
		}),
	)
	if err != nil {
		logger.GetLogger().Fatal("Error creating Discord client: %v", err)
		panic(err)
	}

	// connect to the gateway
	if err = client.OpenGateway(context.TODO()); err != nil {
		logger.GetLogger().Fatal("Error opening Discord gateway: %v", err)
		panic(err)
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
	<-s
}
