package main

import (
	"container/list"
	"sync"
	"time"
)

type Message struct {
	Data      string
	Timestamp time.Time
}

type MessageQueue struct {
	limit    int
	messages *list.List
	lock     sync.Mutex
}

func NewMessageQueue(limit int) *MessageQueue {
	return &MessageQueue{
		limit:    limit,
		messages: list.New(),
	}
}

func (mq *MessageQueue) AddMessage(data string) {
	mq.lock.Lock()
	defer mq.lock.Unlock()

	mq.messages.PushBack(Message{Data: data, Timestamp: time.Now()})

	if mq.messages.Len() > mq.limit {
		mq.messages.Remove(mq.messages.Front())
	}
}

func (mq *MessageQueue) GetMessage() *Message {
	mq.lock.Lock()
	defer mq.lock.Unlock()

	if mq.messages.Len() > 0 {
		front := mq.messages.Front()
		if msg, ok := front.Value.(Message); ok {
			return &msg
		}
	}
	return nil
}

func (mq *MessageQueue) GetLast100Messages() []*Message {
	mq.lock.Lock()
	defer mq.lock.Unlock()

	var last100Messages []*Message
	count := 0
	for e := mq.messages.Back(); e != nil && count < 100; e = e.Prev() {
		if msg, ok := e.Value.(Message); ok {
			last100Messages = append([]*Message{&msg}, last100Messages...)
			count++
		}
	}
	return last100Messages
}
