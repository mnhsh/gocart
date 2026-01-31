package rabbitmq

import (
	"context"
	"database/sql"
	"fmt"
	"encoding/json"

	"github.com/herodragmon/scalable-ecommerce/services/product-service/internal/events"
	"github.com/herodragmon/scalable-ecommerce/services/product-service/internal/database"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	db       *sql.DB
	queries  *database.Queries
}

func NewConsumer(url string, db *sql.DB, queries *database.Queries) (*Consumer, error) {
	fmt.Println("product-service connecting to RabbitMQ")

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("could not connect to RabbitMQ: %w", err)
	}
	fmt.Println("product-service connected to RabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("could not create channel: %w", err)
	}

	err = ch.ExchangeDeclare(
    "orders",
    "topic",
    true,   // durable
    false,  // auto-delete
    false,  // internal
    false,  // no-wait
    nil,
	)
	if err != nil {
    return nil, fmt.Errorf("could not declare exchange: %w", err)
	}

	queue, err := ch.QueueDeclare(
		"product-stock-updates",
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("could not declare queue: %w", err)
	}

	err = ch.QueueBind(
		queue.Name,
		"order.*",
		"orders",
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("could not bind queue: %w", err)
	}
	return &Consumer{
		conn:			conn,
		channel:	ch,
		db:				db,
		queries:	queries,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		"product-stock-updates",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not start consuming: %w", err)
	}
	
	for msg := range msgs {
		var event events.OrderEvent
		err := json.Unmarshal(msg.Body, &event)
		if err != nil {
			fmt.Printf("could not unmarshal message: %v\n", err)
			msg.Nack(false, false)
      continue
		}
		var multiplier int32
		if msg.RoutingKey == "order.created" {
			multiplier = -1
		} else if msg.RoutingKey == "order.cancelled" {
			multiplier = 1
    }

		tx, err := c.db.BeginTx(ctx, nil)
		if err != nil {
			fmt.Printf("could not begin tx: %v\n", err)
			msg.Nack(false, true)
      continue
    }

		qtx := c.queries.WithTx(tx)

		success := true
		for _, item := range event.Items {
			_, err := qtx.UpdateStock(ctx, database.UpdateStockParams{
				ID:    item.ProductID,
				Stock: item.Quantity * multiplier,
			})
			if err != nil {
				fmt.Printf("could not update stock for %s: %v\n", item.ProductID, err)
				success = false
				break
			}
    }
		if success {
			tx.Commit()
			msg.Ack(false)
			fmt.Printf("processed %s for order %s\n", msg.RoutingKey, event.OrderID)
		} else {
			tx.Rollback()
			msg.Nack(false, true)
		}
	}
	return nil
}

func (c *Consumer) Close() {
	c.channel.Close()
	c.conn.Close()
}
