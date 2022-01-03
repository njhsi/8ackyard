package event

import (
	"strings"

	"github.com/leandro-lugaresi/hub"
)

type Hub = hub.Hub
type Data = hub.Fields
type Message = hub.Message

var channelCap = 100
var sharedHub = NewHub()

func NewHub() *Hub {
	return hub.New()
}

func SharedHub() *Hub {
	return sharedHub
}

func Error(msg string) {
	Log.Error(strings.ToLower(msg))
	Publish("notify.error", Data{"message": msg})
}

func Success(msg string) {
	Log.Info(strings.ToLower(msg))
	Publish("notify.success", Data{"message": msg})
}

func Info(msg string) {
	Log.Info(strings.ToLower(msg))
	Publish("notify.info", Data{"message": msg})
}

func Warning(msg string) {
	Log.Warn(strings.ToLower(msg))
	Publish("notify.warning", Data{"message": msg})
}

func Publish(event string, data Data) {
	SharedHub().Publish(Message{
		Name:   event,
		Fields: data,
	})
}

func Subscribe(topics ...string) hub.Subscription {
	return SharedHub().NonBlockingSubscribe(channelCap, topics...)
}

func Unsubscribe(s hub.Subscription) {
	SharedHub().Unsubscribe(s)
}
