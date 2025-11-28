package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// BankDataStore is a thread-safe in-memory storage for bank accounts.
// It uses a mutex to ensure safe concurrent access to the account map.
type BankDataStore struct {
	BankAccounts map[string]*BankAccount
	mu           *sync.Mutex
}

// NewBankDataStore creates and initializes a new BankDataStore with sample data.
// It pre-populates the store with a default bank account for testing purposes.
func NewBankDataStore() *BankDataStore {
	bankAccounts := make(map[string]*BankAccount)
	bankAccounts["1234"] = &BankAccount{
		AccountIBAN: "1234",
		OwnerName:   "Gennaro Coppola",
		Amount:      12400_00,
	}

	return &BankDataStore{
		BankAccounts: bankAccounts,
		mu:           &sync.Mutex{},
	}
}

// Transaction represents a database-like transaction with snapshot isolation.
// It maintains a reference to the data store, a snapshot of the account state,
// and an active flag to prevent double-commit or double-rollback operations.
type Transaction struct {
	store    *BankDataStore
	snapshot *BankAccount
	active   bool
}

var Metrics struct {
	trx int32
}

// Begin starts a new transaction for the specified bank account.
// It locks the data store and creates a snapshot of the account state.
// Returns a Transaction object that must be either committed or rolled back.
// The caller is responsible for calling Commit or Rollback to release the lock.
func (ds *BankDataStore) Begin(iban string) (*Transaction, error) {
	ds.mu.Lock()

	if iban == "" {
		ds.mu.Unlock()
		return nil, fmt.Errorf("iban cannot be empty")
	}

	account, exists := ds.BankAccounts[iban]
	if !exists {
		ds.mu.Unlock()
		return nil, fmt.Errorf("account not found for iban: %s", iban)
	}

	// Deep copy of BankAccount
	snapshot := &BankAccount{
		AccountIBAN: account.AccountIBAN,
		OwnerName:   account.OwnerName,
		Amount:      account.Amount,
	}

	return &Transaction{
		store:    ds,
		snapshot: snapshot,
		active:   true,
	}, nil
}

// Commit finalizes the transaction and releases the lock on the data store.
// After commit, any changes made during the transaction become permanent.
// Returns an error if the transaction is no longer active.
func (tx *Transaction) Commit() error {
	if !tx.active {
		return errors.New("transaction is not active anymore")
	}

	tx.active = false
	tx.store.mu.Unlock()
	return nil
}

// Rollback reverts all changes made during the transaction by restoring the snapshot.
// It then releases the lock on the data store.
// Returns an error if the transaction is no longer active.
func (tx *Transaction) Rollback() error {
	if !tx.active {
		return errors.New("transaction is not active anymore")
	}

	tx.store.BankAccounts[tx.snapshot.AccountIBAN] = tx.snapshot
	tx.active = false
	tx.store.mu.Unlock()
	return nil
}

// BankAccount represents a bank account with its identification and balance information.
// Amount is stored in the smallest currency unit (e.g., cents) to avoid floating-point issues.
type BankAccount struct {
	AccountIBAN string `json:"account_iban"`
	OwnerName   string `json:"owner_name"`
	Amount      uint64 `json:"amount"`
}

// Transfer represents a single money transfer operation to a beneficiary.
// It contains all necessary information to identify and execute the transfer.
type Transfer struct {
	MerchantTrxID   string `json:"merchant_trx_id"`
	Amount          string `json:"amount"`
	AmountCents     uint64 `json:"-"` // Parsed amount in cents, not serialized
	BeneficiaryName string `json:"beneficiary_name"`
	BeneficiaryIBAN string `json:"beneficiary_iban"`
	BeneficiaryBIC  string `json:"beneficiary_bic"`
}

// BulkTransferRequest represents a request to execute multiple transfers in a single operation.
// It includes the source account details, a checksum for integrity verification,
// and a list of individual transfers to be processed.
type BulkTransferRequest struct {
	OrderID     string     `json:"order_id"`
	AccountID   string     `json:"account_id"`
	AccountIBAN string     `json:"account_iban"`
	AccountBIC  string     `json:"account_bic"`
	Checksum    string     `json:"checksum"`
	Transfers   []Transfer `json:"transfers"`
}

// ErrorResponse represents a standard error response structure.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// SuccessResponse represents a successful bulk transfer response
type SuccessResponse struct {
	Status        string `json:"status"`
	Message       string `json:"message"`
	OrderID       string `json:"order_id"`
	TransferCount int    `json:"transfer_count"`
	TotalAmount   uint64 `json:"total_amount"`
}

// writeJSONError writes a JSON error response to the client.
func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	})
}

// writeSuccessResponse writes a JSON success response to the client
func writeSuccessResponse(w http.ResponseWriter, response SuccessResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// parseAmountToCents converts a decimal string amount to cents (uint64).
// Examples: "100.50" -> 10050, "100" -> 10000, "0.50" -> 50
func parseAmountToCents(amount string) (uint64, error) {
	if amount == "" {
		return 0, errors.New("amount cannot be empty")
	}

	// Remove any whitespace
	amount = strings.TrimSpace(amount)

	// Split by decimal point
	parts := strings.Split(amount, ".")

	var cents uint64

	// Parse integer part
	if len(parts[0]) > 0 {
		intPart, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid amount format: %w", err)
		}
		cents = intPart * 100
	}

	// Parse decimal part (if exists)
	if len(parts) > 1 {
		if len(parts) > 2 {
			return 0, errors.New("invalid amount format: multiple decimal points")
		}

		decimalPart := parts[1]
		// Ensure we have exactly 2 decimal places
		if len(decimalPart) > 2 {
			return 0, errors.New("invalid amount format: more than 2 decimal places")
		}

		// Pad with zeros if needed (e.g., "100.5" -> "100.50")
		if len(decimalPart) == 1 {
			decimalPart += "0"
		}

		decimalValue, err := strconv.ParseUint(decimalPart, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid decimal format: %w", err)
		}
		cents += decimalValue
	}

	return cents, nil
}

// parseAmountsMiddleware parses all transfer amounts from string to uint64 (cents).
// It only modifies the request object by populating AmountCents fields.
func parseAmountsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request BulkTransferRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
			return
		}

		// Parse amounts for all transfers
		for i := range request.Transfers {
			amountCents, err := parseAmountToCents(request.Transfers[i].Amount)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest,
					fmt.Sprintf("Invalid amount for transfer %d (%s): %v",
						i, request.Transfers[i].MerchantTrxID, err))
				return
			}
			request.Transfers[i].AmountCents = amountCents
		}

		// Pass the modified request to the next handler via context
		ctx := context.WithValue(r.Context(), "parsedRequest", &request)
		next(w, r.WithContext(ctx))
	}
}

// transfer handles the /transfer/bulk endpoint
func transfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSONError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	// Get the parsed request from context (populated by middleware)
	request, ok := r.Context().Value("parsedRequest").(*BulkTransferRequest)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "Request parsing failed")
		return
	}

	if _, err := ValidateOrder(request); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Order validation failed: %v", err))
		return
	}

	// Process the bulk transfer
	if err := processBulkTransfer(request); err != nil {
		writeJSONError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// Calculate total amount for response
	var totalAmount uint64
	for _, transfer := range request.Transfers {
		totalAmount += transfer.AmountCents
	}

	w.Header().Set("Content-Type", "application/json")
	writeSuccessResponse(w, SuccessResponse{
		Status:        "success",
		Message:       "All transfers processed successfully",
		OrderID:       request.OrderID,
		TransferCount: len(request.Transfers),
		TotalAmount:   totalAmount,
	})
}

// processBulkTransfer handles the business logic of processing multiple transfers.
// It ensures atomicity by using transactions and validates each transfer.
func processBulkTransfer(request *BulkTransferRequest) error {
	// Start a transaction
	tx, err := ds.Begin(request.AccountIBAN)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	// Ensure transaction is finalized
	defer func() {
		if tx.active {
			tx.Rollback()
		}
	}()

	// Get the account
	bankAccount := ds.BankAccounts[request.AccountIBAN]
	if bankAccount == nil {
		return fmt.Errorf("bank account not found")
	}

	// Calculate total amount needed (using pre-parsed AmountCents)
	var totalAmount uint64
	for i, transfer := range request.Transfers {
		Metrics.trx += 1
		if _, err := ValidateTransfer(&transfer); err != nil {
			return fmt.Errorf("validation failed for transfer %d (%s): %w", i, transfer.MerchantTrxID, err)
		}

		totalAmount += transfer.AmountCents
	}

	// Check if account has sufficient funds for all transfers
	if bankAccount.Amount < totalAmount {
		return fmt.Errorf("insufficient funds: need %d, have %d", totalAmount, bankAccount.Amount)
	}

	// Process all transfers
	for _, transfer := range request.Transfers {
		bankAccount.Amount -= transfer.AmountCents
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

var ds *BankDataStore

func init() {
	ds = NewBankDataStore()
}

// main is the entry point of the application.
// It registers the HTTP handlers and starts the server on port 8080.
func main() {
	// Add middleware for better request handling
	http.HandleFunc("/transfer/bulk", loggingMiddleware(parseAmountsMiddleware(transfer)))

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// loggingMiddleware logs incoming requests and their duration.
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s request to %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next(w, r)
	}
}

// ValidateOrder validates the bulk transfer request order-level data.
// It checks the order ID, account details, and checksum for correctness.
// Returns a debug string and an error if validation fails.
func ValidateOrder(request *BulkTransferRequest) (dbg string, err error) {
	if request.OrderID == "" {
		return "order_id", errors.New("order_id is required")
	}
	if request.AccountIBAN == "" {
		return "account_iban", errors.New("account_iban is required")
	}
	if len(request.Transfers) == 0 {
		return "transfers", errors.New("at least one transfer is required")
	}
	if len(request.Transfers) > 100 {
		return "transfers", errors.New("maximum 100 transfers per bulk request")
	}
	return "", nil
}

// ValidateTransfer validates an individual transfer within a bulk request.
// It checks the transfer details including beneficiary information and amount format.
// Returns a debug string and an error if validation fails.
func ValidateTransfer(request *Transfer) (dbg string, err error) {
	if request.MerchantTrxID == "" {
		return "merchant_trx_id", errors.New("merchant_trx_id is required")
	}
	if request.Amount == "" {
		return "amount", errors.New("amount is required")
	}
	if request.BeneficiaryIBAN == "" {
		return "beneficiary_iban", errors.New("beneficiary_iban is required")
	}
	if request.BeneficiaryName == "" {
		return "beneficiary_name", errors.New("beneficiary_name is required")
	}

	// AmountCents should be set by middleware
	if request.AmountCents == 0 {
		return "amount", errors.New("amount must be greater than zero")
	}

	return "", nil
}
