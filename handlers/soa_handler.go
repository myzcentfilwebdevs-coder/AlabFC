package handlers

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"alabapi/database"
	"alabapi/models"

	"github.com/gofiber/fiber/v2"
)

func recalculateSOA(soaID int) {
	db := database.Get()
	var soa models.StatementOfAccount
	if err := db.First(&soa, soaID).Error; err != nil {
		return
	}

	var totalItems float64
	db.Model(&models.SOAItem{}).Where("soa_id = ?", soaID).Select("COALESCE(SUM(amount), 0)").Scan(&totalItems)

	var totalPaid float64
	db.Model(&models.SOAPayment{}).Where("soa_id = ?", soaID).Select("COALESCE(SUM(amount_paid), 0)").Scan(&totalPaid)

	balance := math.Max(0, totalItems-totalPaid)

	statusID := models.RefSOAUnpaid
	if totalItems > 0 && balance <= 0 {
		statusID = models.RefSOAPaid
	} else if totalPaid > 0 && balance > 0 {
		statusID = models.RefSOAPartial
	}

	db.Model(&soa).Updates(map[string]interface{}{
		"total_amount": totalItems,
		"amount_paid":  totalPaid,
		"balance":      balance,
		"status_id":    statusID,
		"updated_at":   time.Now(),
	})
}

func formatSOAResponse(soa models.StatementOfAccount, refMap map[int]string) fiber.Map {
	playerName := ""
	ageCategory := ""
	var jersey *int
	if soa.Player != nil {
		playerName = strings.TrimSpace(soa.Player.FirstName + " " + soa.Player.LastName)
		jersey = soa.Player.JerseyNumber
		if soa.Player.AgeCategoryID != nil {
			ageCategory = refMap[*soa.Player.AgeCategoryID]
		}
	}

	parentName := ""
	if soa.Parent != nil && soa.Parent.Profile != nil {
		parentName = strings.TrimSpace(soa.Parent.Profile.FirstName + " " + soa.Parent.Profile.LastName)
	}

	statusName := ""
	if soa.Status != nil {
		statusName = soa.Status.Details
	} else if soa.StatusID != nil {
		statusName = refMap[*soa.StatusID]
	}

	dueDate := soa.DueDate
	if len(dueDate) >= 10 {
		dueDate = dueDate[:10]
	}

	return fiber.Map{
		"soa_id":         soa.SOAID,
		"soa_number":     soa.SOANumber,
		"player_id":      soa.PlayerID,
		"player_name":    playerName,
		"jersey_number":  jersey,
		"age_category":   ageCategory,
		"parent_user_id": soa.ParentUserID,
		"parent_name":    parentName,
		"billing_period": soa.BillingPeriod,
		"due_date":       dueDate,
		"total_amount":   soa.TotalAmount,
		"amount_paid":    soa.AmountPaid,
		"balance":        soa.Balance,
		"status_id":      soa.StatusID,
		"status":         statusName,
		"notes":          soa.Notes,
		"items":          soa.Items,
		"payments":       soa.Payments,
		"created_at":     soa.CreatedAt,
	}
}

func GetSOAs(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(int)
	roleID, _ := c.Locals("role_id").(int)
	db := database.Get()

	query := db.Model(&models.StatementOfAccount{}).
		Preload("Player").
		Preload("Parent.Profile").
		Preload("Status").
		Preload("Items.FeeType").
		Preload("Payments.PaymentMethod")

	if roleID == models.RefRoleParent {
		childIDs := getParentChildIDs(userID)
		if len(childIDs) > 0 {
			query = query.Where("parent_user_id = ? OR player_id IN ?", userID, childIDs)
		} else {
			query = query.Where("parent_user_id = ?", userID)
		}
	} else if roleID != models.RefRoleAdmin {
		return c.Status(403).JSON(fiber.Map{"error": "Forbidden"})
	}

	if status := strings.TrimSpace(c.Query("status")); status != "" {
		if seq := getRefSeqnumByName(status); seq != nil {
			query = query.Where("status_id = ?", *seq)
		}
	}

	playerParam := strings.TrimSpace(c.Query("player_id"))
	if playerParam == "" {
		playerParam = strings.TrimSpace(c.Query("player_user_id"))
	}
	if playerParam != "" {
		pid, err := strconv.Atoi(playerParam)
		if err == nil {
			if !canAccessPlayer(c, pid) {
				return c.Status(403).JSON(fiber.Map{"error": "Forbidden: you cannot view another child's SOA"})
			}
			query = query.Where("player_id = ?", pid)
		}
	}

	var soas []models.StatementOfAccount
	query.Order("due_date DESC, soa_id DESC").Find(&soas)

	refMap := getRefMap()
	result := make([]fiber.Map, 0, len(soas))
	totalBilled, totalPaid, totalBalance := 0.0, 0.0, 0.0

	for _, soa := range soas {
		result = append(result, formatSOAResponse(soa, refMap))
		totalBilled += soa.TotalAmount
		totalPaid += soa.AmountPaid
		totalBalance += soa.Balance
	}

	return c.JSON(fiber.Map{
		"count": len(result),
		"summary": fiber.Map{
			"total_billed":  totalBilled,
			"total_paid":    totalPaid,
			"total_balance": totalBalance,
		},
		"soas": result,
	})
}

func GetSOAByID(c *fiber.Ctx) error {
	db := database.Get()
	var soa models.StatementOfAccount
	err := db.Preload("Player").
		Preload("Parent.Profile").
		Preload("Status").
		Preload("Items.FeeType").
		Preload("Payments.PaymentMethod").
		First(&soa, c.Params("id")).Error

	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "SOA not found"})
	}

	if !canAccessSOA(c, soa) {
		return c.Status(403).JSON(fiber.Map{"error": "Forbidden: you are not allowed to view this SOA"})
	}

	return c.JSON(formatSOAResponse(soa, getRefMap()))
}

func GetPlayerSOAs(c *fiber.Ctx) error {
	playerID, err := strconv.Atoi(c.Params("player_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid player_id"})
	}

	if !canAccessPlayer(c, playerID) {
		return c.Status(403).JSON(fiber.Map{"error": "Forbidden: you are not allowed to view this child's SOA"})
	}

	var soas []models.StatementOfAccount
	database.Get().
		Preload("Player").
		Preload("Parent.Profile").
		Preload("Status").
		Preload("Items.FeeType").
		Preload("Payments.PaymentMethod").
		Where("player_id = ?", playerID).
		Order("due_date DESC, soa_id DESC").
		Find(&soas)

	refMap := getRefMap()
	result := make([]fiber.Map, 0, len(soas))
	totalBilled, totalPaid, totalBalance := 0.0, 0.0, 0.0

	for _, soa := range soas {
		result = append(result, formatSOAResponse(soa, refMap))
		totalBilled += soa.TotalAmount
		totalPaid += soa.AmountPaid
		totalBalance += soa.Balance
	}

	return c.JSON(fiber.Map{
		"player_id": playerID,
		"count":     len(result),
		"summary": fiber.Map{
			"total_billed":  totalBilled,
			"total_paid":    totalPaid,
			"total_balance": totalBalance,
		},
		"soas": result,
	})
}

func CreateSOA(c *fiber.Ctx) error {
	var req struct {
		SOANumber     string `json:"soa_number"`
		PlayerID      int    `json:"player_id"`
		PlayerUserID  int    `json:"player_user_id"`
		ParentUserID  *int   `json:"parent_user_id"`
		BillingPeriod string `json:"billing_period"`
		DueDate       string `json:"due_date"`
		Notes         string `json:"notes"`
		Items         []struct {
			FeeTypeID   *int    `json:"fee_type_id"`
			FeeType     string  `json:"fee_type"`
			Description string  `json:"description"`
			Amount      float64 `json:"amount"`
		} `json:"items"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	pid := req.PlayerID
	if pid == 0 {
		pid = req.PlayerUserID
	}

	if pid == 0 || strings.TrimSpace(req.BillingPeriod) == "" || strings.TrimSpace(req.DueDate) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "player_id, billing_period, and due_date are required"})
	}

	db := database.Get()

	parentID := req.ParentUserID
	if parentID == nil {
		var link models.ParentPlayer
		if err := db.Where("player_id = ?", pid).First(&link).Error; err == nil {
			parentCopy := link.ParentUserID
			parentID = &parentCopy
		}
	}

	soaNumber := strings.TrimSpace(req.SOANumber)
	if soaNumber == "" {
		soaNumber = fmt.Sprintf("SOA-%s-%d", time.Now().Format("20060102"), time.Now().Unix()%10000)
	}

	totalAmount := 0.0
	for _, it := range req.Items {
		totalAmount += it.Amount
	}

	unpaidStatus := models.RefSOAUnpaid
	soa := models.StatementOfAccount{
		SOANumber:     soaNumber,
		PlayerID:      pid,
		ParentUserID:  parentID,
		BillingPeriod: strings.TrimSpace(req.BillingPeriod),
		DueDate:       strings.TrimSpace(req.DueDate),
		TotalAmount:   totalAmount,
		AmountPaid:    0,
		Balance:       totalAmount,
		StatusID:      &unpaidStatus,
		Notes:         strings.TrimSpace(req.Notes),
	}

	if err := db.Create(&soa).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	for _, it := range req.Items {
		feeTypeID := it.FeeTypeID
		if feeTypeID == nil && strings.TrimSpace(it.FeeType) != "" {
			feeTypeID = getRefSeqnumByName(it.FeeType)
		}
		item := models.SOAItem{
			SOAID:       soa.SOAID,
			FeeTypeID:   feeTypeID,
			Description: strings.TrimSpace(it.Description),
			Amount:      it.Amount,
		}
		db.Create(&item)
	}

	recalculateSOA(soa.SOAID)

	db.Preload("Player").
		Preload("Parent.Profile").
		Preload("Status").
		Preload("Items.FeeType").
		Preload("Payments.PaymentMethod").
		First(&soa, soa.SOAID)

	return c.Status(201).JSON(fiber.Map{
		"message": "SOA created",
		"data":    formatSOAResponse(soa, getRefMap()),
	})
}

func UpdateSOA(c *fiber.Ctx) error {
	db := database.Get()
	var soa models.StatementOfAccount
	if err := db.First(&soa, c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "SOA not found"})
	}

	var req struct {
		BillingPeriod string `json:"billing_period"`
		DueDate       string `json:"due_date"`
		Status        string `json:"status"`
		StatusID      *int   `json:"status_id"`
		Notes         string `json:"notes"`
		Items         []struct {
			FeeTypeID   *int    `json:"fee_type_id"`
			FeeType     string  `json:"fee_type"`
			Description string  `json:"description"`
			Amount      float64 `json:"amount"`
		} `json:"items"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if strings.TrimSpace(req.BillingPeriod) != "" {
		soa.BillingPeriod = strings.TrimSpace(req.BillingPeriod)
	}
	if strings.TrimSpace(req.DueDate) != "" {
		soa.DueDate = strings.TrimSpace(req.DueDate)
	}
	if req.Notes != "" {
		soa.Notes = strings.TrimSpace(req.Notes)
	}
	if req.StatusID != nil {
		soa.StatusID = req.StatusID
	} else if strings.TrimSpace(req.Status) != "" {
		soa.StatusID = getRefSeqnumByName(req.Status)
	}
	soa.UpdatedAt = time.Now()
	db.Save(&soa)

	if len(req.Items) > 0 {
		db.Where("soa_id = ?", soa.SOAID).Delete(&models.SOAItem{})
		for _, it := range req.Items {
			feeTypeID := it.FeeTypeID
			if feeTypeID == nil && strings.TrimSpace(it.FeeType) != "" {
				feeTypeID = getRefSeqnumByName(it.FeeType)
			}
			db.Create(&models.SOAItem{
				SOAID:       soa.SOAID,
				FeeTypeID:   feeTypeID,
				Description: strings.TrimSpace(it.Description),
				Amount:      it.Amount,
			})
		}
		recalculateSOA(soa.SOAID)
	}

	db.Preload("Player").
		Preload("Parent.Profile").
		Preload("Status").
		Preload("Items.FeeType").
		Preload("Payments.PaymentMethod").
		First(&soa, soa.SOAID)

	return c.JSON(fiber.Map{
		"message": "SOA updated",
		"data":    formatSOAResponse(soa, getRefMap()),
	})
}

func DeleteSOA(c *fiber.Ctx) error {
	soaID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid soa_id"})
	}
	res := database.Get().Where("soa_id = ?", soaID).Delete(&models.StatementOfAccount{})
	if res.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "SOA not found"})
	}
	return c.JSON(fiber.Map{"message": "SOA deleted"})
}

func SubmitSOAPayment(c *fiber.Ctx) error {
	db := database.Get()
	var soa models.StatementOfAccount
	if err := db.First(&soa, c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "SOA not found"})
	}

	if !canAccessSOA(c, soa) {
		return c.Status(403).JSON(fiber.Map{"error": "Forbidden: you can only pay your own child's SOA"})
	}

	var req struct {
		AmountPaid      float64 `json:"amount_paid"`
		PaymentMethodID *int    `json:"payment_method_id"`
		PaymentMethod   string  `json:"payment_method"`
		ReferenceNumber string  `json:"reference_number"`
		ReceiptURL      string  `json:"receipt_url"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.AmountPaid <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "amount_paid must be greater than 0"})
	}

	methodID := req.PaymentMethodID
	if methodID == nil && strings.TrimSpace(req.PaymentMethod) != "" {
		methodID = getRefSeqnumByName(req.PaymentMethod)
	}

	var verifiedBy *int
	if rID, ok := c.Locals("role_id").(int); ok && rID == models.RefRoleAdmin {
		if uID, ok := c.Locals("user_id").(int); ok {
			verifiedBy = &uID
		}
	}

	payment := models.SOAPayment{
		SOAID:           soa.SOAID,
		AmountPaid:      req.AmountPaid,
		PaymentMethodID: methodID,
		ReferenceNumber: strings.TrimSpace(req.ReferenceNumber),
		ReceiptURL:      strings.TrimSpace(req.ReceiptURL),
		VerifiedBy:      verifiedBy,
		PaidAt:          time.Now(),
	}

	if err := db.Create(&payment).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	recalculateSOA(soa.SOAID)

	db.Preload("Player").
		Preload("Parent.Profile").
		Preload("Status").
		Preload("Items.FeeType").
		Preload("Payments.PaymentMethod").
		First(&soa, soa.SOAID)

	return c.Status(201).JSON(fiber.Map{
		"message": "Payment submitted",
		"payment": payment,
		"soa":     formatSOAResponse(soa, getRefMap()),
	})
}

func VerifySOAPayment(c *fiber.Ctx) error {
	db := database.Get()
	var payment models.SOAPayment
	if err := db.First(&payment, c.Params("payment_id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Payment not found"})
	}

	userID, _ := c.Locals("user_id").(int)
	payment.VerifiedBy = &userID
	db.Save(&payment)

	return c.JSON(fiber.Map{
		"message": "Payment verified",
		"payment": payment,
	})
}
