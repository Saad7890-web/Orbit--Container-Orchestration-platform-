package triggers

import (
	"context"
	"errors"
	"fmt"
)

var ErrMQTTNotImplemented = errors.New("mqtt triggers are not wired yet")

type MQTTBroker interface {
	Subscribe(ctx context.Context, topic string, handler func(topic string, payload []byte) error) error
	Close() error
}

type MQTTDriver struct {
	broker MQTTBroker
}

func NewMQTTDriver(broker MQTTBroker) *MQTTDriver {
	return &MQTTDriver{broker: broker}
}

func (m *MQTTDriver) Start(ctx context.Context) error {
	if m == nil || m.broker == nil {
		return ErrMQTTNotImplemented
	}
	return fmt.Errorf("mqtt trigger start: %w", ErrMQTTNotImplemented)
}