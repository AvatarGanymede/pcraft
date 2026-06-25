package websocket

import "github.com/AvatarGanymede/pcraft/internal/common/logger"

// Provide creates the unified WebSocket gateway.
func Provide(log *logger.Logger) (*Gateway, error) {
	gateway := NewGateway(log)
	return gateway, nil
}
