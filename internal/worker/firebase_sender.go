package worker

import (
	"context"
	"errors"

	"firebase.google.com/go/v4/messaging"
)

type FirebaseSender struct {
	client *messaging.Client
}

func NewFirebaseSender(client *messaging.Client) *FirebaseSender {
	return &FirebaseSender{client: client}
}

func (s *FirebaseSender) Send(ctx context.Context, title, body string, tokens []string) ([]SendResult, error) {
	if len(tokens) == 0 || len(tokens) > tokenBatchSize {
		return nil, errors.New("FCM call must contain between 1 and 500 tokens")
	}
	response, err := s.client.SendEachForMulticast(ctx, &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	})
	if err != nil {
		return nil, err
	}
	if response == nil || len(response.Responses) != len(tokens) {
		return nil, errors.New("FCM returned an incomplete response")
	}
	results := make([]SendResult, len(tokens))
	for i, result := range response.Responses {
		if result == nil {
			results[i] = SendResult{Error: "FCM returned an empty token response"}
			continue
		}
		if result.Success {
			results[i] = SendResult{Success: true}
			continue
		}
		failure := SendResult{Error: "FCM send failed"}
		if result.Error != nil {
			failure.Error = result.Error.Error()
			failure.Invalid = messaging.IsUnregistered(result.Error) || messaging.IsInvalidArgument(result.Error)
			failure.Transient = messaging.IsUnavailable(result.Error) || messaging.IsInternal(result.Error) || messaging.IsQuotaExceeded(result.Error)
		}
		results[i] = failure
	}
	return results, nil
}
