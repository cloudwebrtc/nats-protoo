package nprotoo

import (
	"encoding/json"

	"github.com/chuckpreslar/emission"
	"github.com/cloudwebrtc/nats-protoo/logger"
	nats "github.com/nats-io/nats.go"
)

// Broadcaster .
type Broadcaster struct {
	emission.Emitter
	subj string
	np   *NatsProtoo
}

func newBroadcaster(subj string, np *NatsProtoo, nc *nats.Conn) *Broadcaster {
	var bc Broadcaster
	bc.Emitter = *emission.NewEmitter()
	bc.subj = subj
	bc.np = np
	bc.np.On("close", func(code int, err string) {
		logger.Infof("Transport closed [%d] %s", code, err)
		bc.Emit("close", code, err)
	})
	bc.np.On("error", func(code int, err string) {
		logger.Warnf("Transport got error (%d, %s)", code, err)
		bc.Emit("error", code, err)
	})
	return &bc
}

// Say .
func (bc *Broadcaster) Say(method string, data map[string]interface{}) {
	notification := &Notification{
		Notification: true,
		Method:       method,
		Data:         data,
	}
	str, err := json.Marshal(notification)
	if err != nil {
		logger.Errorf("Marshal %v", err)
		return
	}
	logger.Debugf("Send notification [%s]", method)
	bc.np.Send(str, bc.subj, _EMPTY_)
}
