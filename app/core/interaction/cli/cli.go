package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"alter0/app/pkg/types"
)

type CLIChannel struct {
	id     string
	userID string
}

func NewCLIChannel(userID string) *CLIChannel {
	if strings.TrimSpace(userID) == "" {
		userID = "local_user"
	}
	return &CLIChannel{id: "cli", userID: userID}
}

func (c *CLIChannel) ID() string {
	return c.id
}

func (c *CLIChannel) Start(ctx context.Context, handler func(types.Message)) error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(">> Alter0 CLI started. Type 'exit' to quit.")

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			fmt.Print("> ")
			if !scanner.Scan() {
				return nil
			}
			text := strings.TrimSpace(scanner.Text())
			if text == "exit" || text == "quit" {
				fmt.Println("Exiting CLI loop...")
				return nil
			}

			msgID := fmt.Sprintf("cli-%d", time.Now().UnixNano())
			msg := types.Message{
				ID:        msgID,
				Content:   text,
				Role:      types.MessageRoleUser,
				ChannelID: c.id,
				UserID:    c.userID,
				Envelope: &types.MessageEnvelope{
					Direction: types.EnvelopeDirectionInbound,
					Channel:   c.id,
					PeerID:    c.userID,
					PeerType:  "user",
					MessageID: msgID,
					Parts: []types.EnvelopePart{
						{Type: types.EnvelopePartText, Text: text},
					},
				},
				Meta: map[string]interface{}{
					"user_id": c.userID,
				},
			}
			handler(msg)
		}
	}
}

func (c *CLIChannel) Send(ctx context.Context, msg types.Message) error {
	if strings.TrimSpace(msg.TaskID) != "" {
		fmt.Printf("[Alter0][task:%s]: %s\n", msg.TaskID, msg.Content)
		return nil
	}
	fmt.Printf("[Alter0]: %s\n", msg.Content)
	return nil
}
