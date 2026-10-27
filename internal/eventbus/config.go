package eventbus

import (
	"fmt"
	"strings"

	"rent/internal/platform"
)

type Config struct {
	Transport string
	Brokers   []string
	Topic     string
	Group     string
}

func Load() (Config, error) {
	config := Config{
		Transport: platform.Env("NOTIFICATION_TRANSPORT", "grpc"),
		Topic:     platform.Env("KAFKA_TOPIC", "rental.orders"),
		Group:     platform.Env("KAFKA_GROUP", "rent-notifier"),
	}
	if config.Transport != "grpc" && config.Transport != "kafka" {
		return config, fmt.Errorf("NOTIFICATION_TRANSPORT must be grpc or kafka")
	}
	for _, broker := range strings.Split(platform.Env("KAFKA_BROKERS", "127.0.0.1:19092"), ",") {
		if broker = strings.TrimSpace(broker); broker != "" {
			config.Brokers = append(config.Brokers, broker)
		}
	}
	if config.Transport == "kafka" && len(config.Brokers) == 0 {
		return config, fmt.Errorf("KAFKA_BROKERS is empty")
	}
	return config, nil
}
