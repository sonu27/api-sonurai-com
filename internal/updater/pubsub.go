package updater

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
)

// getOrCreateTopic gets a topic or creates it if it doesn't exist.
func getOrCreateTopic(ctx context.Context, client *pubsub.Client, topicID string) (*pubsub.Topic, error) {
	topic := client.Topic(topicID)
	ok, err := topic.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check if topic exists: %v", err)
	}
	if !ok {
		topic, err = client.CreateTopic(ctx, topicID)
		if err != nil {
			return nil, fmt.Errorf("failed to create topic (%q): %v", topicID, err)
		}
	}
	return topic, nil
}

// getOrCreateSub gets a subscription or creates it if it doesn't exist.
func getOrCreateSub(ctx context.Context, client *pubsub.Client, subID string, cfg *pubsub.SubscriptionConfig) (*pubsub.Subscription, error) {
	sub := client.Subscription(subID)
	ok, err := sub.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check if subscription exists: %v", err)
	}
	if !ok {
		sub, err = client.CreateSubscription(ctx, subID, *cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create subscription (%q): %v", topicID, err)
		}
	}
	return sub, nil
}
