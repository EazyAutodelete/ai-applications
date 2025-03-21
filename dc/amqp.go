package dc

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/eazyautodelete/ai-users/config"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()
var restExchange = config.EnvGet("REST_REQUEST_EXCHANGE", "restRequests")
var retryExchange = config.EnvGet("REST_RETRY_EXCHANGE", "restRequestsRetry")
var responseExchange = config.EnvGet("REST_RESPONSE_EXCHANGE", "restResponses")
var requestQueue = config.EnvGet("REST_REQUEST_QUEUE", "restRequestsQueue")
var retryQueue = config.EnvGet("REST_RETRY_QUEUE", "restRetryQueue")
var prefetch = 1

func removeUrlCredentials(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		fmt.Println("Invalid URL:", err)
		return rawURL
	}
	userInfo := parsedURL.User.Username()
	if userInfo != "" {
		userInfo += ":****@"
	}

	loggedURL := fmt.Sprintf("%s://%s%s%s", parsedURL.Scheme, userInfo, parsedURL.Host, parsedURL.RequestURI())
	loggedURL = strings.TrimRight(loggedURL, "/")

	return loggedURL
}

type RabbitMQ struct {
	Connection *amqp091.Connection
	Channel    *amqp091.Channel
}

func ConnectRabbitMQ() (*amqp091.Connection, error) {
	var conn *amqp091.Connection
	var err error

	queueUser := config.EnvGet("QUEUE_USER", "guest")
	queuePass := config.EnvGet("QUEUE_PASSWORD", "guest")
	rawQueueHosts := config.EnvGet("QUEUE_HOSTS", "localhost:5672")
	queueHostStrings := strings.Split(rawQueueHosts, ",")

	queueHosts := make([]string, len(queueHostStrings))
	for i, host := range queueHostStrings {
		queueHosts[i] = fmt.Sprintf("amqp://%s:%s@%s", queueUser, queuePass, host)
	}

	for {
		for _, url := range queueHosts {
			logger.Infof("Trying to connect to RabbitMQ at %s...", removeUrlCredentials(url))
			conn, err = amqp091.Dial(url)
			if err == nil {
				logger.Infof("Connected to RabbitMQ at %s", removeUrlCredentials(url))
				return conn, nil
			}
			logger.Warnf("Failed to connect to RabbitMQ at %s: %s", removeUrlCredentials(url), err)
		}

		logger.Warnf("Failed to connect to RabbitMQ, retrying in 1 second")
		time.Sleep(1 * time.Second)
	}
}

func SetupRabbitMQConnection() *RabbitMQ {
	for {
		conn, err := ConnectRabbitMQ()
		if err != nil {
			logger.Warnf("Failed to connect to RabbitMQ cluster: %s", err)
			time.Sleep(1 * time.Second)
			continue
		}

		ch, err := PrepareRabbitMQChannel(conn)

		rmq := &RabbitMQ{Connection: conn, Channel: ch}

		// Handle reconnection on connection failure
		go func() {
			<-conn.NotifyClose(make(chan *amqp091.Error))
			logger.Warnf("RabbitMQ connection closed. Attempting to reconnect...")
			rmq.Reconnect()
		}()

		return rmq
	}
}

func (rmq *RabbitMQ) Reconnect() {
	for {
		conn, err := ConnectRabbitMQ()
		if err != nil {
			logger.Warnf("Failed to reconnect to RabbitMQ: %s", err)
			time.Sleep(1 * time.Second)
			continue
		}

		ch, err := PrepareRabbitMQChannel(conn)
		if err != nil {
			logger.Warnf("Failed to create RabbitMQ channel during reconnect: %s", err)
			_ = conn.Close()
			time.Sleep(1 * time.Second)
			continue
		}

		rmq.Connection = conn
		rmq.Channel = ch

		// Restart reconnection logic
		go func() {
			<-conn.NotifyClose(make(chan *amqp091.Error))
			logger.Warnf("RabbitMQ connection closed. Attempting to reconnect...")
			rmq.Reconnect()
		}()

		logger.Infof("Successfully reconnected to RabbitMQ")
		return
	}
}

func PrepareRabbitMQChannel(conn *amqp091.Connection) (*amqp091.Channel, error) {
	ch, err := conn.Channel()
	if err != nil {
		logger.Errorf("Failed to open a channel: %s", err)
		return nil, err
	}

	err = ch.ExchangeDeclare(restExchange, "direct", true, false, false, false, nil)
	if err != nil {
		logger.Errorf("Failed to declare exchange %s: %s", restExchange, err)
		return nil, err
	}

	err = ch.ExchangeDeclare(retryExchange, "direct", true, false, false, false, nil)
	if err != nil {
		logger.Errorf("Failed to declare exchange %s: %s", retryExchange, err)
		return nil, err
	}

	err = ch.ExchangeDeclare(responseExchange, "direct", true, false, false, false, nil)
	if err != nil {
		logger.Errorf("Failed to declare exchange %s: %s", responseExchange, err)
		return nil, err
	}

	_, err = ch.QueueDeclare(requestQueue, true, false, false, false, amqp091.Table{"x-dead-letter-exchange": retryExchange})
	if err != nil {
		logger.Errorf("Failed to declare queue %s: %s", requestQueue, err)
		return nil, err
	}

	_, err = ch.QueueDeclare(retryQueue, true, false, false, false, amqp091.Table{
		"x-dead-letter-exchange": restExchange,
		"x-message-ttl":          1000,
	})
	if err != nil {
		logger.Errorf("Failed to declare queue %s: %s", retryQueue, err)
		return nil, err
	}

	err = ch.QueueBind(requestQueue, "", restExchange, false, nil)
	if err != nil {
		logger.Errorf("Failed to bind queue %s to exchange %s: %s", requestQueue, restExchange, err)
		return nil, err
	}

	err = ch.QueueBind(retryQueue, "", retryExchange, false, nil)
	if err != nil {
		logger.Errorf("Failed to bind queue %s to exchange %s: %s", retryQueue, retryExchange, err)
		return nil, err
	}
	// ^ normal rest setup

	return ch, nil
}

func SendMessageToDiscord(ch *amqp091.Channel, msg string) error {
	err := ch.Publish(
		"",
		"discord",
		false,
		false,
		amqp091.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		},
	)
	if err != nil {
		logger.Errorf("Failed to publish message to Discord: %s", err)
		return err
	}

	return nil
}
