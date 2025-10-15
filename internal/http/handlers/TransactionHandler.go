package handlers

import (
	"net/http"
	"strconv"
	"time"

	"test-go2/ent"
	"test-go2/ent/transaction"
	"test-go2/internal/utils"

	"github.com/gin-gonic/gin"
)

type TransactionHandler struct {
	client *ent.Client
}

func NewTransactionHandler(client *ent.Client) *TransactionHandler {
	return &TransactionHandler{client: client}
}

func (h *TransactionHandler) List(c *gin.Context) {
	params := utils.ParsePaginationParams(c)

	query := h.client.Transaction.Query()

	if userID := c.Query("user_id"); userID != "" {
		if id, err := strconv.Atoi(userID); err == nil {
			query = query.Where(transaction.UserIDEQ(id))
		}
	}

	if categoryID := c.Query("category_id"); categoryID != "" {
		if id, err := strconv.Atoi(categoryID); err == nil {
			query = query.Where(transaction.CategoryIDEQ(id))
		}
	}

	if transactionType := c.Query("type"); transactionType != "" {
		if transactionType == "income" || transactionType == "expense" {
			query = query.Where(transaction.TypeEQ(transaction.Type(transactionType)))
		}
	}

	total, err := query.Count(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count transactions: " + err.Error()})
		return
	}

	transactions, err := query.
		WithUser().
		WithCategory().
		Offset(params.Offset).
		Limit(params.Limit).
		Order(ent.Desc(transaction.FieldTxDate)).
		All(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions: " + err.Error()})
		return
	}

	response := utils.CreatePaginationResponse(transactions, params, total)
	c.JSON(http.StatusOK, response)
}

// GetByID retrieves a single transaction by ID
func (h *TransactionHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	transaction, err := h.client.Transaction.Query().
		Where(transaction.IDEQ(id)).
		WithUser().
		WithCategory().
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// Create creates a new transaction
func (h *TransactionHandler) Create(c *gin.Context) {
	var body struct {
		UserID         *int     `json:"user_id" binding:"omitempty,min=1"`
		CategoryID     *int     `json:"category_id" binding:"omitempty,min=1"`
		Type           string   `json:"type" binding:"required,oneof=income expense"`
		Amount         float64  `json:"amount" binding:"required,gt=0"`
		Currency       string   `json:"currency" binding:"required,len=3"`
		Description    *string  `json:"description"`
		ConversionRate *float64 `json:"conversion_rate" binding:"omitempty,gt=0"`
		TxDate         string   `json:"tx_date" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Parse transaction date
	txDate, err := time.Parse("2006-01-02T15:04:05Z07:00", body.TxDate)
	if err != nil {
		// Try parsing as date only
		txDate, err = time.Parse("2006-01-02", body.TxDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use ISO 8601 format or YYYY-MM-DD"})
			return
		}
	}

	// Create transaction
	create := h.client.Transaction.Create().
		SetType(transaction.Type(body.Type)).
		SetAmount(body.Amount).
		SetCurrency(body.Currency).
		SetTxDate(txDate).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now())

	if body.UserID != nil {
		create = create.SetUserID(*body.UserID)
	}

	if body.CategoryID != nil {
		create = create.SetCategoryID(*body.CategoryID)
	}

	if body.Description != nil {
		create = create.SetDescription(*body.Description)
	}


	if body.ConversionRate != nil {
		create = create.SetConversionRate(*body.ConversionRate)
	} else {
		// Default conversion rate to 1.0 if not provided
		create = create.SetConversionRate(1.0)
	}

	transaction, err := create.Save(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, transaction)
}

// Update updates an existing transaction
func (h *TransactionHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	var body struct {
		UserID         *int     `json:"user_id" binding:"omitempty,min=1"`
		CategoryID     *int     `json:"category_id" binding:"omitempty,min=1"`
		Type           *string  `json:"type" binding:"omitempty,oneof=income expense"`
		Amount         *float64 `json:"amount" binding:"omitempty,gt=0"`
		Currency       *string  `json:"currency" binding:"omitempty,len=3"`
		Description    *string  `json:"description"`
		ConversionRate *float64 `json:"conversion_rate" binding:"omitempty,gt=0"`
		TxDate         *string  `json:"tx_date"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Check if transaction exists
	exists, err := h.client.Transaction.Query().Where(transaction.IDEQ(id)).Exist(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check transaction existence"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	// Build update query
	update := h.client.Transaction.UpdateOneID(id).SetUpdatedAt(time.Now())

	if body.UserID != nil {
		update = update.SetUserID(*body.UserID)
	}

	if body.CategoryID != nil {
		update = update.SetCategoryID(*body.CategoryID)
	}

	if body.Type != nil {
		update = update.SetType(transaction.Type(*body.Type))
	}

	if body.Amount != nil {
		update = update.SetAmount(*body.Amount)
	}

	if body.Currency != nil {
		update = update.SetCurrency(*body.Currency)
	}

	if body.Description != nil {
		update = update.SetDescription(*body.Description)
	}


	if body.ConversionRate != nil {
		update = update.SetConversionRate(*body.ConversionRate)
	}

	if body.TxDate != nil {
		txDate, err := time.Parse("2006-01-02T15:04:05Z07:00", *body.TxDate)
		if err != nil {
			txDate, err = time.Parse("2006-01-02", *body.TxDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use ISO 8601 format or YYYY-MM-DD"})
				return
			}
		}
		update = update.SetTxDate(txDate)
	}

	updatedTransaction, err := update.Save(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedTransaction)
}

// Delete deletes a transaction
func (h *TransactionHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	err = h.client.Transaction.DeleteOneID(id).Exec(c)
	if err != nil {
		if ent.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction deleted successfully"})
}

// GetByUserID retrieves transactions for a specific user
func (h *TransactionHandler) GetByUserID(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	params := utils.ParsePaginationParams(c)

	query := h.client.Transaction.Query().Where(transaction.UserIDEQ(userID))

	// Optional type filter
	if transactionType := c.Query("type"); transactionType != "" {
		if transactionType == "income" || transactionType == "expense" {
			query = query.Where(transaction.TypeEQ(transaction.Type(transactionType)))
		}
	}

	total, err := query.Count(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count transactions: " + err.Error()})
		return
	}

	transactions, err := query.
		WithCategory().
		Offset(params.Offset).
		Limit(params.Limit).
		Order(ent.Desc(transaction.FieldTxDate)).
		All(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions: " + err.Error()})
		return
	}

	response := utils.CreatePaginationResponse(transactions, params, total)
	c.JSON(http.StatusOK, response)
}
