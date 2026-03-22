package fcm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const endpoint = "https://fcm.googleapis.com/fcm/send"

type Client struct {
	serverKey  string
	httpClient *http.Client
}

func New(serverKey string) *Client {
	return &Client{
		serverKey:  serverKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type Message struct {
	To           string            `json:"to"`
	Notification *Notification     `json:"notification,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
}

type Notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (c *Client) Send(ctx context.Context, msg Message) error {
	if c.serverKey == "" {
		fmt.Printf("[FCM-MOCK] to=%s title=%s body=%s\n",
			msg.To, msg.Notification.Title, msg.Notification.Body)
		return nil
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal fcm message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create fcm request: %w", err)
	}
	req.Header.Set("Authorization", "key="+c.serverKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send fcm: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fcm returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) SendToMany(ctx context.Context, tokens []string, notif Notification, data map[string]string) {
	for _, token := range tokens {
		_ = c.Send(ctx, Message{
			To:           token,
			Notification: &notif,
			Data:         data,
		})
	}
}
