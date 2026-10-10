package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type ChatMessage struct {
	Text string `json:"text"`
}

type chatItem struct {
	msg       string
	threadKey string
}

type Notifier struct {
	WebhookURL string
	queue      chan chatItem
}

func ThreadKey(container, metric string) string {
	return container + "-" + metric + "-" + time.Now().Format("2006-01-02")
}

func (n *Notifier) send(ctx context.Context, msg string, threadKey string) error {
	log := logf.FromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	//build msg format and URL
	payload := ChatMessage{Text: msg}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Error(err, "Error marshalling payload")
		return err
	}

	u, err := url.Parse(n.WebhookURL)
	if err != nil {
		log.Error(err, "Error parsing url", "WebhookURL", n.WebhookURL)
		return err
	}

	q := u.Query()

	q.Set("threadKey", threadKey)
	q.Set("messageReplyOption", "REPLY_MESSAGE_FALLBACK_TO_NEW_THREAD")
	u.RawQuery = q.Encode()
	finalURL := u.String()

	//Send HTTP call
	req, err := http.NewRequestWithContext(ctx, "POST", finalURL, bytes.NewReader(body))
	if err != nil {
		log.Error(err, "Error creating HTTP Request")
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err, "Error sending HTTP Request")
		return err
	}

	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Error(nil, "Chat webhook returned non-OK status", "status", resp.StatusCode, "response", string(respBody))
		return fmt.Errorf("chat webhook returned status %d with response %s", resp.StatusCode, respBody)
	}

	return nil
}

func (n *Notifier) Start(ctx context.Context) error {
	log := logf.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case item := <-n.queue:
			if err := n.send(ctx, item.msg, item.threadKey); err != nil {
				log.Error(err, "failed to send chat message.", "threadKey", item.threadKey)
			}
			time.Sleep(time.Second)
		}
	}
}

func (n *Notifier) SendMessage(ctx context.Context, msg string, threadKey string) error {
	item := chatItem{msg: msg, threadKey: threadKey}

	select {
	case n.queue <- item:
		return nil

	default:
		return fmt.Errorf("chat queue full, message dropped")
	}

}

func NewNotifier(webhookURL string) *Notifier {
	return &Notifier{WebhookURL: webhookURL, queue: make(chan chatItem, 100)}
}
