package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

var client *Client
var currentUser *User

func main() {
	client = NewClient("http://localhost:8080")

	clearScreen()
	fmt.Println("========================================")
	fmt.Println("       ECOM CLI - E-Commerce Store     ")
	fmt.Println("========================================")
	fmt.Println()

	for {
		if client.Token == "" {
			showAuthMenu()
		} else {
			showMainMenu()
		}
	}
}

// Helper functions

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func prompt(label string) string {
	fmt.Print(label)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func promptPassword(label string) string {
	fmt.Print(label)
	password, _ := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	return string(password)
}

func promptInt(label string) int {
	for {
		input := prompt(label)
		num, err := strconv.Atoi(input)
		if err == nil && num >= 0 {
			return num
		}
		fmt.Println("Please enter a valid number.")
	}
}

func pressEnterToContinue() {
	prompt("\nPress Enter to continue...")
}

func formatPrice(cents int) string {
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}

func promptPrice(label string) int {
	for {
		input := prompt(label)
		price, err := strconv.ParseFloat(input, 64)
		if err == nil && price > 0 {
			return int(price * 100)
		}
		fmt.Println("Please enter a valid price (e.g., 19.99).")
	}
}

// Auth Menu

func showAuthMenu() {
	fmt.Println("\n--- Welcome ---")
	fmt.Println("1. Login")
	fmt.Println("2. Register")
	fmt.Println("3. Exit")
	fmt.Println()

	choice := prompt("Enter choice: ")

	switch choice {
	case "1":
		handleLogin()
	case "2":
		handleRegister()
	case "3":
		fmt.Println("Goodbye!")
		os.Exit(0)
	default:
		fmt.Println("Invalid choice.")
	}
}

func handleLogin() {
	fmt.Println("\n--- Login ---")
	email := prompt("Email: ")
	password := promptPassword("Password: ")

	resp, err := client.Login(email, password)
	if err != nil {
		fmt.Printf("Login failed: %s\n", err)
		pressEnterToContinue()
		return
	}

	currentUser = &resp.User
	clearScreen()
	fmt.Printf("Welcome back, %s!\n", currentUser.Email)
}

func handleRegister() {
	fmt.Println("\n--- Register ---")
	email := prompt("Email: ")
	password := promptPassword("Password: ")

	err := client.Register(email, password)
	if err != nil {
		fmt.Printf("Registration failed: %s\n", err)
		pressEnterToContinue()
		return
	}

	fmt.Println("Registration successful! Please login.")
	pressEnterToContinue()
}

// Main Menu

func showMainMenu() {
	fmt.Println("\n--- Main Menu ---")
	fmt.Printf("Logged in as: %s", currentUser.Email)
	if currentUser.Role == "admin" {
		fmt.Print(" [ADMIN]")
	}
	fmt.Println("\n")
	fmt.Println("1. Browse Products")
	fmt.Println("2. View Cart")
	fmt.Println("3. My Orders")
	if currentUser.Role == "admin" {
		fmt.Println("5. Admin: Manage Products")
	}
	fmt.Println("4. Logout")
	fmt.Println()

	choice := prompt("Enter choice: ")

	switch choice {
	case "1":
		showProducts()
	case "2":
		showCart()
	case "3":
		showOrders()
	case "4":
		client.Token = ""
		currentUser = nil
		clearScreen()
		fmt.Println("Logged out successfully.")
	case "5":
		if currentUser.Role == "admin" {
			showAdminMenu()
		} else {
			fmt.Println("Invalid choice.")
		}
	default:
		fmt.Println("Invalid choice.")
	}
}

// Products

func showProducts() {
	clearScreen()
	fmt.Println("\n--- Products ---\n")

	products, err := client.GetProducts()
	if err != nil {
		fmt.Printf("Failed to fetch products: %s\n", err)
		pressEnterToContinue()
		return
	}

	if len(products) == 0 {
		fmt.Println("No products available.")
		pressEnterToContinue()
		return
	}

	// Display products
	fmt.Printf("%-4s %-30s %-10s %-10s\n", "#", "Name", "Price", "Stock")
	fmt.Println(strings.Repeat("-", 58))
	for i, p := range products {
		fmt.Printf("%-4d %-30s %-10s %-10d\n", i+1, p.Name, formatPrice(p.PriceCents), p.Stock)
	}

	fmt.Println()
	fmt.Println("Enter product number to add to cart, or 0 to go back.")
	choice := promptInt("Choice: ")

	if choice == 0 {
		return
	}

	if choice < 1 || choice > len(products) {
		fmt.Println("Invalid product number.")
		pressEnterToContinue()
		return
	}

	product := products[choice-1]
	quantity := promptInt(fmt.Sprintf("Quantity for '%s': ", product.Name))

	if quantity == 0 {
		fmt.Println("Cancelled.")
		pressEnterToContinue()
		return
	}

	err = client.AddToCart(product.ID, quantity)
	if err != nil {
		fmt.Printf("Failed to add to cart: %s\n", err)
		pressEnterToContinue()
		return
	}

	fmt.Printf("Added %d x %s to cart!\n", quantity, product.Name)
	pressEnterToContinue()
}

// Cart

func showCart() {
	clearScreen()
	fmt.Println("\n--- Your Cart ---\n")

	cart, err := client.GetCart()
	if err != nil {
		fmt.Printf("Failed to fetch cart: %s\n", err)
		pressEnterToContinue()
		return
	}

	if len(cart.Items) == 0 {
		fmt.Println("Your cart is empty.")
		pressEnterToContinue()
		return
	}

	// Display cart items
	fmt.Printf("%-4s %-20s %-10s %-10s\n", "#", "Product ID", "Qty", "Price")
	fmt.Println(strings.Repeat("-", 48))
	for i, item := range cart.Items {
		fmt.Printf("%-4d %-20s %-10d %-10s\n", i+1, item.ProductID[:8]+"...", item.Quantity, formatPrice(item.PriceCents*item.Quantity))
	}
	fmt.Println(strings.Repeat("-", 48))
	fmt.Printf("%-34s %-10s\n", "Total:", formatPrice(cart.TotalCents))

	fmt.Println()
	fmt.Println("1. Checkout (Create Order)")
	fmt.Println("2. Clear Cart")
	fmt.Println("0. Back")
	fmt.Println()

	choice := prompt("Choice: ")

	switch choice {
	case "1":
		handleCheckout()
	case "2":
		handleClearCart()
	case "0":
		return
	default:
		fmt.Println("Invalid choice.")
	}
}

func handleCheckout() {
	fmt.Println("\nCreating order...")

	order, err := client.CreateOrder()
	if err != nil {
		fmt.Printf("Failed to create order: %s\n", err)
		pressEnterToContinue()
		return
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("         ORDER CREATED SUCCESSFULLY!   ")
	fmt.Println("========================================")
	fmt.Printf("Order ID: %s\n", order.ID)
	fmt.Printf("Status: %s\n", order.Status)
	fmt.Printf("Total: %s\n", formatPrice(order.TotalCents))
	fmt.Println("========================================")
	pressEnterToContinue()
}

func handleClearCart() {
	err := client.ClearCart()
	if err != nil {
		fmt.Printf("Failed to clear cart: %s\n", err)
		pressEnterToContinue()
		return
	}

	fmt.Println("Cart cleared!")
	pressEnterToContinue()
}

// Orders

func showOrders() {
	clearScreen()
	fmt.Println("\n--- Your Orders ---\n")

	orders, err := client.GetOrders()
	if err != nil {
		fmt.Printf("Failed to fetch orders: %s\n", err)
		pressEnterToContinue()
		return
	}

	if len(orders) == 0 {
		fmt.Println("You have no orders.")
		pressEnterToContinue()
		return
	}

	// Display orders
	fmt.Printf("%-4s %-12s %-12s %-10s\n", "#", "Order ID", "Status", "Total")
	fmt.Println(strings.Repeat("-", 42))
	for i, o := range orders {
		fmt.Printf("%-4d %-12s %-12s %-10s\n", i+1, o.ID[:8]+"...", o.Status, formatPrice(o.TotalCents))
	}

	fmt.Println()
	fmt.Println("Enter order number to cancel (pending only), or 0 to go back.")
	choice := promptInt("Choice: ")

	if choice == 0 {
		return
	}

	if choice < 1 || choice > len(orders) {
		fmt.Println("Invalid order number.")
		pressEnterToContinue()
		return
	}

	order := orders[choice-1]
	if order.Status != "pending" {
		fmt.Printf("Cannot cancel order with status '%s'. Only pending orders can be cancelled.\n", order.Status)
		pressEnterToContinue()
		return
	}

	confirm := prompt(fmt.Sprintf("Cancel order %s? (y/n): ", order.ID[:8]+"..."))
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Cancelled.")
		pressEnterToContinue()
		return
	}

	err = client.CancelOrder(order.ID)
	if err != nil {
		fmt.Printf("Failed to cancel order: %s\n", err)
		pressEnterToContinue()
		return
	}

	fmt.Println("Order cancelled! Stock has been restored.")
	pressEnterToContinue()
}

// Admin Menu

func showAdminMenu() {
	clearScreen()
	fmt.Println("\n--- Admin: Manage Products ---\n")
	fmt.Println("1. Add Product")
	fmt.Println("2. Delete Product")
	fmt.Println("0. Back")
	fmt.Println()

	choice := prompt("Choice: ")

	switch choice {
	case "1":
		handleAddProduct()
	case "2":
		handleDeleteProduct()
	case "0":
		return
	default:
		fmt.Println("Invalid choice.")
	}
}

func handleAddProduct() {
	fmt.Println("\n--- Add New Product ---\n")

	name := prompt("Product Name: ")
	if name == "" {
		fmt.Println("Name is required.")
		pressEnterToContinue()
		return
	}

	description := prompt("Description (optional): ")
	priceCents := promptPrice("Price (e.g., 19.99): ")
	stock := promptInt("Initial Stock: ")

	fmt.Println("\nCreating product...")

	product, err := client.CreateProduct(name, description, priceCents, stock)
	if err != nil {
		fmt.Printf("Failed to create product: %s\n", err)
		pressEnterToContinue()
		return
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("       PRODUCT CREATED SUCCESSFULLY!   ")
	fmt.Println("========================================")
	fmt.Printf("ID: %s\n", product.ID)
	fmt.Printf("Name: %s\n", product.Name)
	fmt.Printf("Price: %s\n", formatPrice(product.PriceCents))
	fmt.Printf("Stock: %d\n", product.Stock)
	fmt.Println("========================================")
	pressEnterToContinue()
}

func handleDeleteProduct() {
	clearScreen()
	fmt.Println("\n--- Delete Product ---\n")

	products, err := client.GetProducts()
	if err != nil {
		fmt.Printf("Failed to fetch products: %s\n", err)
		pressEnterToContinue()
		return
	}

	if len(products) == 0 {
		fmt.Println("No products available.")
		pressEnterToContinue()
		return
	}

	// Display products
	fmt.Printf("%-4s %-30s %-10s %-10s\n", "#", "Name", "Price", "Stock")
	fmt.Println(strings.Repeat("-", 58))
	for i, p := range products {
		fmt.Printf("%-4d %-30s %-10s %-10d\n", i+1, p.Name, formatPrice(p.PriceCents), p.Stock)
	}

	fmt.Println()
	fmt.Println("Enter product number to delete, or 0 to go back.")
	choice := promptInt("Choice: ")

	if choice == 0 {
		return
	}

	if choice < 1 || choice > len(products) {
		fmt.Println("Invalid product number.")
		pressEnterToContinue()
		return
	}

	product := products[choice-1]
	confirm := prompt(fmt.Sprintf("Delete '%s'? This cannot be undone. (y/n): ", product.Name))
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Cancelled.")
		pressEnterToContinue()
		return
	}

	err = client.DeleteProduct(product.ID)
	if err != nil {
		fmt.Printf("Failed to delete product: %s\n", err)
		pressEnterToContinue()
		return
	}

	fmt.Printf("Product '%s' deleted successfully!\n", product.Name)
	pressEnterToContinue()
}
