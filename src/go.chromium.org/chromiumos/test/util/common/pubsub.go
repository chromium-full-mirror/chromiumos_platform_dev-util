// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"context"
	"log"

	"cloud.google.com/go/pubsub"
)

type Publisher struct {
	Topic *pubsub.Topic
}

// Publish publishes a message.
func (p *Publisher) Publish(ctx context.Context, data []byte) error {
	m := &pubsub.Message{Data: data}
	serviceID, err := p.Topic.Publish(ctx, m).Get(ctx)
	if err != nil {
		return err
	}
	log.Printf("\nPublished message. ServiceId %s, topicId: %s", serviceID, p.Topic.ID())
	return nil
}
