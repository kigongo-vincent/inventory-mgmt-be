package Sale

import (
	"encoding/json"
	"fmt"
	"sync"

	User "github.com/kigongo-vincent/inventory-mgmt-be.git/modules/User"
)

// SSEClient represents a connected SSE client
type SSEClient struct {
	UserID    uint
	CompanyID uint
	Role      User.UserRole
	Channel   chan []byte
}

// SSEService manages SSE connections
type SSEService struct {
	clients map[uint]map[uint]*SSEClient // companyID -> userID -> client
	mu      sync.RWMutex
}

var sseService *SSEService
var once sync.Once

// GetSSEService returns the singleton SSE service
func GetSSEService() *SSEService {
	once.Do(func() {
		sseService = &SSEService{
			clients: make(map[uint]map[uint]*SSEClient),
		}
	})
	return sseService
}

// RegisterClient registers a new SSE client
func (s *SSEService) RegisterClient(userID uint, companyID uint, role User.UserRole) *SSEClient {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.clients[companyID] == nil {
		s.clients[companyID] = make(map[uint]*SSEClient)
	}

	client := &SSEClient{
		UserID:    userID,
		CompanyID: companyID,
		Role:      role,
		Channel:   make(chan []byte, 10), // Buffered channel
	}

	s.clients[companyID][userID] = client
	return client
}

// UnregisterClient removes a client from the service
func (s *SSEService) UnregisterClient(userID uint, companyID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if companyClients, exists := s.clients[companyID]; exists {
		if client, exists := companyClients[userID]; exists {
			close(client.Channel)
			delete(companyClients, userID)
		}
		if len(companyClients) == 0 {
			delete(s.clients, companyID)
		}
	}
}

// BroadcastSaleEvent broadcasts a sale event to all super admin clients of a company
func (s *SSEService) BroadcastSaleEvent(companyID uint, saleEvent SaleEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	companyClients, exists := s.clients[companyID]
	if !exists {
		return
	}

	// Serialize event to JSON
	eventData, err := json.Marshal(saleEvent)
	if err != nil {
		fmt.Printf("Error marshaling sale event: %v\n", err)
		return
	}

	// Format as SSE message
	sseMessage := fmt.Sprintf("data: %s\n\n", string(eventData))

	// Send to all super admin clients in the company
	for userID, client := range companyClients {
		if client.Role == User.SuperAdmin {
			select {
			case client.Channel <- []byte(sseMessage):
				// Successfully sent
			default:
				// Channel is full, skip this client
				fmt.Printf("Warning: Channel full for user %d, skipping\n", userID)
			}
		}
	}
}

// SaleEvent represents a sale event to be broadcast
type SaleEvent struct {
	Type      string  `json:"type"` // "new_sale"
	SaleID    uint    `json:"saleId"`
	ProductName string `json:"productName"`
	Quantity   int    `json:"quantity"`
	TotalPrice float64 `json:"totalPrice"`
	Currency   string  `json:"currency"`
	SellerName string  `json:"sellerName"`
	BranchName string  `json:"branchName,omitempty"`
	CreatedAt  string  `json:"createdAt"`
}
