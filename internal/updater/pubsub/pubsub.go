package pubsub

import (
	"context"
	"fmt"

	"google.golang.org/api/option"

	"cloud.google.com/go/pubsub"
)

func Start(
	ctx context.Context,
	projectID string,
	topicID string,
	subID string,
	fn func(ctx context.Context) error,
	opts ...option.ClientOption,
) error {
	pubsubClient, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		return err
	}
	defer pubsubClient.Close()

	topic, err := getOrCreateTopic(ctx, pubsubClient, topicID)
	if err != nil {
		return err
	}

	sub, err := getOrCreateSub(ctx, pubsubClient, subID, &pubsub.SubscriptionConfig{
		Topic:                     topic,
		EnableExactlyOnceDelivery: true,
	})
	if err != nil {
		return err
	}

	fmt.Println("image updater listening")
	return sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		if err := fn(ctx); err != nil {
			fmt.Printf("message processing failed: %v\n", err)
			msg.Nack()
			return
		}
		msg.Ack()
	})
}

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
			return nil, fmt.Errorf("failed to create subscription (%q): %v", subID, err)
		}
	}
	return sub, nil
}
