package events

import "github.com/google/uuid"

type OrderItem struct {
    ProductID uuid.UUID `json:"product_id"`
    Quantity  int32     `json:"quantity"`
}
type OrderEvent struct {
    OrderID uuid.UUID   `json:"order_id"`
    Items   []OrderItem `json:"items"`
}
