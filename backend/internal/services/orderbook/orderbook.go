package orderbook

import (
	"fmt"
	"sync"

	"github.com/prediction-market/backend/internal/models"
	"github.com/shopspring/decimal"
)

// PriceLevel represents a single price level in the order book
type PriceLevel struct {
	Price    decimal.Decimal
	Quantity decimal.Decimal
	Orders   []*models.Order
}

// OrderBook represents an order book for a specific market outcome
type OrderBook struct {
	MarketID uint64
	Outcome  uint8
	Buys     []PriceLevel // sorted by price descending (best buy first)
	Sells    []PriceLevel // sorted by price ascending (best sell first)
	mu       sync.RWMutex
}

// OrderBookManager manages multiple order books
type OrderBookManager struct {
	books map[string]*OrderBook // key: "marketId-outcome"
	mu    sync.RWMutex
}

// MatchResult represents the result of order matching
type MatchResult struct {
	Trades      []models.Trade
	MakerOrders []*models.Order
	TakerOrder  *models.Order
}

// NewOrderBookManager creates a new OrderBookManager
func NewOrderBookManager() *OrderBookManager {
	return &OrderBookManager{
		books: make(map[string]*OrderBook),
	}
}

// makeKey generates a key for the order book map
func makeKey(marketID uint64, outcome uint8) string {
	return fmt.Sprintf("%d-%d", marketID, outcome)
}

// GetOrCreate returns an existing order book or creates a new one
func (m *OrderBookManager) GetOrCreate(marketID uint64, outcome uint8) *OrderBook {
	key := makeKey(marketID, outcome)

	m.mu.RLock()
	book, exists := m.books[key]
	m.mu.RUnlock()

	if exists {
		return book
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if book, exists = m.books[key]; exists {
		return book
	}

	book = &OrderBook{
		MarketID: marketID,
		Outcome:  outcome,
		Buys:     make([]PriceLevel, 0),
		Sells:    make([]PriceLevel, 0),
	}
	m.books[key] = book
	return book
}

// GetDepth returns a copy of the order book for a specific market outcome
func (m *OrderBookManager) GetDepth(marketID uint64, outcome uint8) *OrderBook {
	key := makeKey(marketID, outcome)

	m.mu.RLock()
	book, exists := m.books[key]
	m.mu.RUnlock()

	if !exists {
		return nil
	}

	book.mu.RLock()
	defer book.mu.RUnlock()

	// Create a copy of the order book for safe external use
	copyBook := &OrderBook{
		MarketID: book.MarketID,
		Outcome:  book.Outcome,
		Buys:     make([]PriceLevel, len(book.Buys)),
		Sells:    make([]PriceLevel, len(book.Sells)),
	}

	for i, level := range book.Buys {
		copyBook.Buys[i] = PriceLevel{
			Price:    level.Price,
			Quantity: level.Quantity,
			Orders:   make([]*models.Order, len(level.Orders)),
		}
		copy(copyBook.Buys[i].Orders, level.Orders)
	}

	for i, level := range book.Sells {
		copyBook.Sells[i] = PriceLevel{
			Price:    level.Price,
			Quantity: level.Quantity,
			Orders:   make([]*models.Order, len(level.Orders)),
		}
		copy(copyBook.Sells[i].Orders, level.Orders)
	}

	return copyBook
}

// AddOrder adds an order to the order book and performs matching
func (ob *OrderBook) AddOrder(order *models.Order) *MatchResult {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	result := &MatchResult{
		Trades:      make([]models.Trade, 0),
		MakerOrders: make([]*models.Order, 0),
		TakerOrder:  order,
	}

	// Match the order against the opposite side
	if order.Side == models.OrderSideBuy {
		ob.matchBuyOrder(order, result)
	} else {
		ob.matchSellOrder(order, result)
	}

	// Add remaining quantity to the book if not fully filled
	if order.RemainingQuantity().GreaterThan(decimal.Zero) {
		ob.addToBook(order)
	}

	return result
}

// matchBuyOrder matches a buy order against sell levels
func (ob *OrderBook) matchBuyOrder(order *models.Order, result *MatchResult) {
	remaining := order.RemainingQuantity()

	for len(ob.Sells) > 0 && remaining.GreaterThan(decimal.Zero) {
		level := &ob.Sells[0]

		// Buy orders match when buy price >= sell price
		if order.Price.LessThan(level.Price) {
			break
		}

		for len(level.Orders) > 0 && remaining.GreaterThan(decimal.Zero) {
			makerOrder := level.Orders[0]
			makerRemaining := makerOrder.RemainingQuantity()

			// Determine the trade quantity
			tradeQty := decimal.Min(remaining, makerRemaining)

			// Create the trade (at maker's price for price improvement)
			trade := models.Trade{
				MarketID:     ob.MarketID,
				MakerOrderID: makerOrder.ID,
				TakerOrderID: order.ID,
				MakerAddress: makerOrder.UserAddress,
				TakerAddress: order.UserAddress,
				Outcome:      ob.Outcome,
				Price:        makerOrder.Price,
				Quantity:     tradeQty,
			}
			result.Trades = append(result.Trades, trade)

			// Update filled quantities
			order.FilledQuantity = order.FilledQuantity.Add(tradeQty)
			makerOrder.FilledQuantity = makerOrder.FilledQuantity.Add(tradeQty)

			// Update order statuses
			ob.updateOrderStatus(order)
			ob.updateOrderStatus(makerOrder)

			// Track affected maker orders
			result.MakerOrders = append(result.MakerOrders, makerOrder)

			// Update remaining
			remaining = order.RemainingQuantity()

			// Update level quantity
			level.Quantity = level.Quantity.Sub(tradeQty)

			// Remove fully filled maker order from level
			if makerOrder.RemainingQuantity().IsZero() {
				level.Orders = level.Orders[1:]
			}
		}

		// Remove empty price levels
		if len(level.Orders) == 0 {
			ob.Sells = ob.Sells[1:]
		}
	}
}

// matchSellOrder matches a sell order against buy levels
func (ob *OrderBook) matchSellOrder(order *models.Order, result *MatchResult) {
	remaining := order.RemainingQuantity()

	for len(ob.Buys) > 0 && remaining.GreaterThan(decimal.Zero) {
		level := &ob.Buys[0]

		// Sell orders match when sell price <= buy price
		if order.Price.GreaterThan(level.Price) {
			break
		}

		for len(level.Orders) > 0 && remaining.GreaterThan(decimal.Zero) {
			makerOrder := level.Orders[0]
			makerRemaining := makerOrder.RemainingQuantity()

			// Determine the trade quantity
			tradeQty := decimal.Min(remaining, makerRemaining)

			// Create the trade (at maker's price for price improvement)
			trade := models.Trade{
				MarketID:     ob.MarketID,
				MakerOrderID: makerOrder.ID,
				TakerOrderID: order.ID,
				MakerAddress: makerOrder.UserAddress,
				TakerAddress: order.UserAddress,
				Outcome:      ob.Outcome,
				Price:        makerOrder.Price,
				Quantity:     tradeQty,
			}
			result.Trades = append(result.Trades, trade)

			// Update filled quantities
			order.FilledQuantity = order.FilledQuantity.Add(tradeQty)
			makerOrder.FilledQuantity = makerOrder.FilledQuantity.Add(tradeQty)

			// Update order statuses
			ob.updateOrderStatus(order)
			ob.updateOrderStatus(makerOrder)

			// Track affected maker orders
			result.MakerOrders = append(result.MakerOrders, makerOrder)

			// Update remaining
			remaining = order.RemainingQuantity()

			// Update level quantity
			level.Quantity = level.Quantity.Sub(tradeQty)

			// Remove fully filled maker order from level
			if makerOrder.RemainingQuantity().IsZero() {
				level.Orders = level.Orders[1:]
			}
		}

		// Remove empty price levels
		if len(level.Orders) == 0 {
			ob.Buys = ob.Buys[1:]
		}
	}
}

// updateOrderStatus updates the status of an order based on fill state
func (ob *OrderBook) updateOrderStatus(order *models.Order) {
	if order.RemainingQuantity().IsZero() {
		order.Status = models.OrderStatusFilled
	} else if order.FilledQuantity.GreaterThan(decimal.Zero) {
		order.Status = models.OrderStatusPartial
	}
}

// addToBook adds an order to the appropriate side of the book
func (ob *OrderBook) addToBook(order *models.Order) {
	if order.Side == models.OrderSideBuy {
		ob.addToBuys(order)
	} else {
		ob.addToSells(order)
	}
}

// addToBuys inserts an order into the Buys side maintaining price-descending order
func (ob *OrderBook) addToBuys(order *models.Order) {
	remaining := order.RemainingQuantity()

	// Find the correct position (price descending)
	pos := 0
	for pos < len(ob.Buys) && ob.Buys[pos].Price.GreaterThan(order.Price) {
		pos++
	}

	// Check if we should add to an existing level
	if pos < len(ob.Buys) && ob.Buys[pos].Price.Equal(order.Price) {
		ob.Buys[pos].Orders = append(ob.Buys[pos].Orders, order)
		ob.Buys[pos].Quantity = ob.Buys[pos].Quantity.Add(remaining)
		return
	}

	// Insert a new price level
	newLevel := PriceLevel{
		Price:    order.Price,
		Quantity: remaining,
		Orders:   []*models.Order{order},
	}

	// Insert at position
	ob.Buys = append(ob.Buys, PriceLevel{})
	copy(ob.Buys[pos+1:], ob.Buys[pos:])
	ob.Buys[pos] = newLevel
}

// addToSells inserts an order into the Sells side maintaining price-ascending order
func (ob *OrderBook) addToSells(order *models.Order) {
	remaining := order.RemainingQuantity()

	// Find the correct position (price ascending)
	pos := 0
	for pos < len(ob.Sells) && ob.Sells[pos].Price.LessThan(order.Price) {
		pos++
	}

	// Check if we should add to an existing level
	if pos < len(ob.Sells) && ob.Sells[pos].Price.Equal(order.Price) {
		ob.Sells[pos].Orders = append(ob.Sells[pos].Orders, order)
		ob.Sells[pos].Quantity = ob.Sells[pos].Quantity.Add(remaining)
		return
	}

	// Insert a new price level
	newLevel := PriceLevel{
		Price:    order.Price,
		Quantity: remaining,
		Orders:   []*models.Order{order},
	}

	// Insert at position
	ob.Sells = append(ob.Sells, PriceLevel{})
	copy(ob.Sells[pos+1:], ob.Sells[pos:])
	ob.Sells[pos] = newLevel
}

// RemoveOrder removes an order from the book
func (ob *OrderBook) RemoveOrder(order *models.Order) bool {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	if order.Side == models.OrderSideBuy {
		return ob.removeFromBuys(order)
	}
	return ob.removeFromSells(order)
}

// removeFromBuys removes an order from the Buys side
func (ob *OrderBook) removeFromBuys(order *models.Order) bool {
	for i := range ob.Buys {
		if ob.Buys[i].Price.Equal(order.Price) {
			for j, o := range ob.Buys[i].Orders {
				if o.ID == order.ID {
					// Remove order from level
					ob.Buys[i].Quantity = ob.Buys[i].Quantity.Sub(o.RemainingQuantity())
					ob.Buys[i].Orders = append(ob.Buys[i].Orders[:j], ob.Buys[i].Orders[j+1:]...)

					// Remove empty level
					if len(ob.Buys[i].Orders) == 0 {
						ob.Buys = append(ob.Buys[:i], ob.Buys[i+1:]...)
					}
					return true
				}
			}
			break
		}
	}
	return false
}

// removeFromSells removes an order from the Sells side
func (ob *OrderBook) removeFromSells(order *models.Order) bool {
	for i := range ob.Sells {
		if ob.Sells[i].Price.Equal(order.Price) {
			for j, o := range ob.Sells[i].Orders {
				if o.ID == order.ID {
					// Remove order from level
					ob.Sells[i].Quantity = ob.Sells[i].Quantity.Sub(o.RemainingQuantity())
					ob.Sells[i].Orders = append(ob.Sells[i].Orders[:j], ob.Sells[i].Orders[j+1:]...)

					// Remove empty level
					if len(ob.Sells[i].Orders) == 0 {
						ob.Sells = append(ob.Sells[:i], ob.Sells[i+1:]...)
					}
					return true
				}
			}
			break
		}
	}
	return false
}
