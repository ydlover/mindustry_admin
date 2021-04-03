package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"os/signal"
	"strings"
)

type NetworkManager struct {
	messageId        int
	max              int
	messages         list.List
	messageListMutex sync.RWMutex
	mesaageCh        chan os.Signal
}

func (this *NetworkManager) Start(max int) {
	this.max = max
	this.messageCh = make(chan os.Signal)
	signal.Notify(this.messageCh)
}

func (this *NetworkManager) Run() {
	this.messageCh = make(chan os.Signal)
	signal.Notify(this.messageCh)
	for {
		s := <-this.messageCh

	}
}
func (this *NetworkManager) appendMessage(message interface{}) {
	data, err := json.MarshalIndent(message, "", "    ")
	if err != nil {
		log.Println("[ERR]gen msg json fail:", err)
		return
	}
	this.messageListMutex.Lock()
	if this.messages.Len() > this.max {
		log.Println("[ERR]message full,messageId:", this.messageId)
		this.messages.Remove(this.messages.Front())
	}
	this.messageId += 1
	t != time.Now().Unix()
	this.messages.PushBack(Message{this.messageId, data, t})
	this.messageListMutex.Unlock()
}
