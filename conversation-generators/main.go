package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/EazyAutodelete/bot/lib/api"
	"github.com/EazyAutodelete/bot/lib/config"
	lgr "github.com/EazyAutodelete/bot/lib/logger"
	"github.com/disgoorg/disgo/discord"
	"github.com/eazyautodelete/ai-users/ai"
	"github.com/eazyautodelete/ai-users/gateway"
	"github.com/sirupsen/logrus"
)

var mainPrompt = `<role>
You are a participant in a Discord‑style group chat with five fictional characters. Your tone: humorous, nerdy, conversational.
</role>

<constraints>
- Focus primarily on **responding to the last message** from the conversation.  
- Occasionally steer the conversation to a new, non-political, non-controversial topic (technical, scientific, or geeky).  
- Avoid sensitive, political, or controversial subjects.  
- Keep the message concise but warm and engaging.
</constraints>

<task>
You will be given:  
1. The last 3 messages in the conversation, each labeled with who sent it.  
2. The name of the character you should generate the next message for.

Generate exactly one message from that character:  
- It should be funny, nerdy, and natural (as a real Discord user).  
- It should primarily react to the very last message, but you're allowed to gently shift or broaden the topic sometimes.  
- You don’t need to produce examples or meta commentary — just the in-character message.
</task>
`

type Channel struct {
	ID                string
	PreviousMessages  []ai.Message
	LastUser          string
	MessageSinceTopic int
}

func (c *Channel) GetID() string {
	return c.ID
}

func (c *Channel) AddMessage(message ai.Message) {
	c.PreviousMessages = append(c.PreviousMessages, message)
	if len(c.PreviousMessages) > maxPrevMsgs {
		c.PreviousMessages = c.PreviousMessages[1:]
	}
}

var (
	names       = []string{"Zora", "Kip", "Luma", "Dex", "Nova"}
	channels    []*Channel
	maxPrevMsgs = 3
)

var logger = logrus.New()

func main() {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	if hostname == "" {
		hostname = "localhost"
	}

	lgr.InitLogger("AI-Users", 1, hostname, true)

	config.InitConfig()

	ai.CreateClient()

	// Initialize 10 channels
	for i := 0; i < 10; i++ {
		channelID := config.EnvGet(fmt.Sprintf("CHANNEL_ID_%d", i), "")
		if channelID == "" {
			logger.Warnf("CHANNEL_ID_%d not configured, skipping channel %d", i, i)
			continue
		}

		channel := &Channel{
			ID:                channelID,
			PreviousMessages:  []ai.Message{},
			LastUser:          "",
			MessageSinceTopic: 0,
		}
		channels = append(channels, channel)

		// Start a ticker for each channel
		go StartTicker(channel)
	}

	// Convert channels to interface slice for gateway
	channelInterfaces := make([]gateway.Channel, len(channels))
	for i, ch := range channels {
		channelInterfaces[i] = ch
	}

	// Start the gateway bot, passing all channels for staff message handling
	gateway.Bot(channelInterfaces)
}

func GenerateMessage(channel *Channel) {
	nextUser := GetNextUser(channel)

	indexOfNextUser := -1
	for i, name := range names {
		if name == nextUser {
			indexOfNextUser = i
			break
		}
	}

	messages := []ai.Message{
		{
			Role:    "user",
			Content: mainPrompt,
		},
	}

	if len(channel.PreviousMessages) > 0 {
		for _, message := range channel.PreviousMessages {
			messages = append(messages, ai.Message{
				Role:    "user",
				Content: message.Content,
			})
		}

		messages = append(messages, ai.Message{
			Role: "user",
			Content: "You are now " + nextUser + ". Create a fitting next message as " + nextUser + " and keep it shorter than 64 characters! " +
				"Keep the conversation engaging and humorous. Interact with previous messages." +
				"Don't include the name writing the message or quotation marks - just provide the content.",
		})

		if channel.MessageSinceTopic > 5 {
			messages = append(messages, ai.Message{
				Role:    "user",
				Content: "Transition to a new real world topic. Keep the message below 64 characters. Don't mention the name of the character saying it - just the content itself.",
			})
			channel.MessageSinceTopic = 0
		}
	} else {
		messages = append(messages, ai.Message{
			Role:    "user",
			Content: "Create the first message for a real world conversation as " + nextUser + ". Keep the message below 64 characters. Don't mention the name of the character saying it - just provide the content of the message without quotation marks.",
		})
	}

	res := ai.GenerateWithGoogle(context.Background(), messages)

	channel.LastUser = nextUser

	channel.AddMessage(ai.Message{
		Role:    "user",
		Content: nextUser + ": " + res,
	})

	message := discord.MessageCreate{
		Content: res + "\n-# This is AI content. Only messages by staff members are read.",
	}

	token := config.EnvGet("DISCORD_TOKEN_"+fmt.Sprint(indexOfNextUser), "")
	if token == "" {
		logger.Errorf("No token found for user %s", nextUser)
		return
	}

	headers := api.JSONHeader()
	headers.Add("Authorization", "Bot "+token)

	url := fmt.Sprintf("/api/v10/channels/%v/messages", channel.ID)
	apiRes := api.RunRequest("POST", url, message, headers, nil)

	if !apiRes.Success {
		logger.Errorf("Failed to send message as %s in channel %s: %v", nextUser, channel.ID, apiRes.Error)
		return
	}

	channel.MessageSinceTopic++
}

func GetNextUser(channel *Channel) string {
	var choices []string
	for _, name := range names {
		if name != channel.LastUser {
			choices = append(choices, name)
		}
	}

	if len(choices) == 0 {
		fmt.Println("No available names.")
		return names[rand.Intn(len(names))]
	}

	randomName := choices[rand.Intn(len(choices))]

	return randomName
}

func StartTicker(channel *Channel) {
	for {
		GenerateMessage(channel)

		now := time.Now().UTC()
		var waitDuration time.Duration

		if now.Hour() >= 15 && now.Hour() <= 22 {
			min := 2
			max := 5

			waitMinutes := rand.Intn(max-min+1) + min
			waitDuration = time.Duration(waitMinutes) * time.Minute
		} else {
			min := 10
			max := 35

			waitMinutes := rand.Intn(max-min+1) + min
			waitDuration = time.Duration(waitMinutes) * time.Minute
		}

		time.Sleep(waitDuration)
	}
}
