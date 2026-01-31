package main

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type LoginResponse struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

type Product struct {
	ID         string `json:"ID"`
	Name       string `json:"Name"`
	PriceCents int    `json:"PriceCents"`
	Stock      int    `json:"Stock"`
}

type CartItem struct {
	ID         string `json:"id"`
	ProductID  string `json:"product_id"`
	Quantity   int    `json:"quantity"`
	PriceCents int    `json:"price_cents"`
}

type Cart struct {
	ID         string     `json:"id"`
	Items      []CartItem `json:"items"`
	TotalCents int        `json:"total_cents"`
}

type Order struct {
	ID         string `json:"ID"`
	Status     string `json:"Status"`
	TotalCents int    `json:"TotalCents"`
	CreatedAt  string `json:"CreatedAt"`
}
