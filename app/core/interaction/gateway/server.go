package gateway

import (
	"context"
	"log"
	"strings"
	"sync"

	"alter0/app/pkg/types"
)

type DefaultGateway struct {
	agent    types.Agent
	channels map[string]types.Channel
	mu       sync.RWMutex
}

func NewGateway(agent types.Agent) *DefaultGateway {
	return &DefaultGateway{
		agent:    agent,
		channels: make(map[string]types.Channel),
	}
}

func (g *DefaultGateway) RegisterChannel(c types.Channel) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.channels[c.ID()] = c
	log.Printf("[Gateway] Registered channel: %s", c.ID())
}

func (g *DefaultGateway) Start(ctx context.Context) error {
	var wg sync.WaitGroup

	handler := func(msg types.Message) {
		log.Printf("[Gateway] Received message from channel=%s user=%s", msg.ChannelID, msg.UserID)
		response, err := g.agent.Process(ctx, msg)
		if err != nil {
			log.Printf("[Gateway] Agent error: %v", err)
			response = types.Message{
				ID:        "resp-" + msg.ID,
				Content:   "Error: " + err.Error(),
				Role:      "assistant",
				ChannelID: msg.ChannelID,
				UserID:    msg.UserID,
				TaskID:    msg.TaskID,
				RequestID: msg.RequestID,
				Meta:      msg.Meta,
			}
		}
		if strings.TrimSpace(msg.Content) == "" && strings.TrimSpace(response.Content) == "" {
			return
		}

		g.mu.RLock()
		channel, exists := g.channels[msg.ChannelID]
		g.mu.RUnlock()
		if !exists {
			log.Printf("[Gateway] Channel not found for reply: %s", msg.ChannelID)
			return
		}

		if response.ID == "" {
			response.ID = "resp-" + msg.ID
		}
		if response.ChannelID == "" {
			response.ChannelID = msg.ChannelID
		}
		if response.Role == "" {
			response.Role = "assistant"
		}
		if response.UserID == "" {
			response.UserID = msg.UserID
		}
		if response.RequestID == "" {
			response.RequestID = msg.RequestID
		}
		if response.TaskID == "" {
			response.TaskID = msg.TaskID
		}
		if response.Meta == nil {
			response.Meta = map[string]interface{}{}
		}
		for k, v := range msg.Meta {
			if _, exists := response.Meta[k]; !exists {
				response.Meta[k] = v
			}
		}

		if err := channel.Send(ctx, response); err != nil {
			log.Printf("[Gateway] Failed to send reply: %v", err)
		}
	}

	g.mu.RLock()
	for _, c := range g.channels {
		wg.Add(1)
		go func(ch types.Channel) {
			defer wg.Done()
			if err := ch.Start(ctx, handler); err != nil {
				log.Printf("[Gateway] Channel %s error: %v", ch.ID(), err)
			}
		}(c)
	}
	g.mu.RUnlock()

	log.Println("[Gateway] Started all channels")
	wg.Wait()
	return nil
}
