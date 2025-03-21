package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/eazyautodelete/ai-users/ai"
	"github.com/eazyautodelete/ai-users/config"
	"github.com/eazyautodelete/ai-users/dc"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

var previousMessages = []string{
	"Zora: Can you believe how sunny it is today? I might need sunglasses for my circuits!",
	"Luma: Kip: I heard a thunderstorm's rolling in later. Hope it doesn’t short-circuit my sense of humor!",
	"Luma: It’s surprisingly chilly for this time of year. Maybe I should install an upgrade—thermal insulation!",
}

var names = []string{"Zora", "Kip", "Luma", "Dex", "Nova"}
var lastUser = "Luma"

var logger = logrus.New()

var restExchange = config.EnvGet("REST_REQUEST_EXCHANGE", "restRequests")

func setupLogger() {
	logLevel := config.EnvGet("LOG_LEVEL", "info")
	lvl, err := logrus.ParseLevel(logLevel)

	if err != nil {
		panic("Failed to parse log level")
	}

	logger.SetLevel(lvl)
}

func main() {
	setupLogger()

	rabbit := dc.SetupRabbitMQConnection()

	ai := ai.CreateClient(context.Background(), config.EnvGet("GEMINI_TOKEN", ""))

	StartTicker(ai, rabbit)
}

func GenerateMessage(ai *ai.AIClient, rabbit *dc.RabbitMQ) {
	nextUser := GetNextUser()

	indexOfNextUser := -1
	for i, name := range names {
		if name == nextUser {
			indexOfNextUser = i
			break
		}
	}

	prompt := "Answer as: " + nextUser

	res := ai.Generate(context.Background(), prompt)

	lastUser = nextUser

	previousMessages = append(previousMessages, res)
	if len(previousMessages) > 3 {
		previousMessages = previousMessages[1:]
	}

	body := map[string]string{
		"content": res,
	}

	token := config.EnvGet("DISCORD_TOKEN_"+fmt.Sprint(indexOfNextUser), "")
	if token == "" {
		logger.Errorf("No token found for user %s", nextUser)
		return
	}

	headers := make(map[string]string)
	headers["Authorization"] = "Bot " + token
	headers["Content-Type"] = "application/json"

	message := map[string]interface{}{
		"headers": headers,
		"body":    body,
		"method":  "POST",
		"path":    fmt.Sprintf("/api/v10/channels/%v/messages", config.EnvGet("CHANNEL_ID", "")),
	}

	bodyString, err := json.Marshal(message)
	if err != nil {
		log.Fatalf("Failed to marshal message: %s", err)
	}

	pub := amqp091.Publishing{
		ContentType:   "application/json",
		Body:          bodyString,
		CorrelationId: "",
	}

	rabbit.Channel.Publish(restExchange, "", false, false, pub)
}

func GetNextUser() string {

	var choices []string
	for _, name := range names {
		if name != lastUser {
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

func StartTicker(ai *ai.AIClient, rabbit *dc.RabbitMQ) {
	ai.Generate(context.Background(), "Create the first message")

	for {
		GenerateMessage(ai, rabbit)

		now := time.Now()
		var waitDuration time.Duration

		if now.Hour() >= 17 && now.Hour() <= 22 {
			min := 5
			max := 20

			waitMinutes := rand.Intn(max-min+1) + min
			waitDuration = time.Duration(waitMinutes) * time.Minute / 60
		} else {
			min := 15
			max := 30

			waitMinutes := rand.Intn(max-min+1) + min
			waitDuration = time.Duration(waitMinutes) * time.Minute / 60
		}

		time.Sleep(waitDuration)
	}
}
