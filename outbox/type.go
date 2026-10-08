package outbox

import (
	"fmt"
	"strings"
)

type EventType string

const (
	RabbitMQ EventType = "RabbitMQ"
	Kafka    EventType = "Kafka"
	RestAPI  EventType = "RestAPI"
)

func (e EventType) String() string {
	return string(e)
}

const RabbitMQDestinationFormat = "%s;%s"

func FormatRabbitMQDestination(exchange string, routingKey string) string {
	return fmt.Sprintf(RabbitMQDestinationFormat, exchange, routingKey)
}

// ParseRabbitMQDestination return pair exchange and routing key if contain ";" separator
// otherwise return queue name
func ParseRabbitMQDestination(dest string) (string, string) {
	if dest == "" {
		return "", ""
	}
	s := strings.Split(dest, ";")

	// exchange, routing key
	if len(s) >= 2 {
		return s[0], s[1]
	}

	// only queue name
	return "", s[0]
}
