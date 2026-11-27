package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"alabapi/database"
	"alabapi/models"

	"github.com/gofiber/fiber/v2"
)

func getRefMap() map[int]string {
	var refs []models.GenericReference
	database.Get().Find(&refs)
	m := make(map[int]string, len(refs))
	for _, r := range refs {
		m[r.Seqnum] = r.Details
	}
	return m
}

func resolveAttendanceStatus(statusID *int, statusName string) (*int, string, error) {
	if strings.TrimSpace(statusName) != "" {
		lower := strings.ToLower(strings.TrimSpace(statusName))
		if lower != "present" && lower != "absent" {
			return nil, "", fmt.Errorf("status must be Present or Absent")
		}
		seq := getRefSeqnumByName(statusName)
		if seq == nil {
			if lower == "present" {
				id := 32
				seq = &id
			} else {
				id := 33
				seq = &id
			}
		}
		return seq, lower, nil
	}

	if statusID != nil {
		refMap := getRefMap()
		name := strings.ToLower(strings.TrimSpace(refMap[*statusID]))
		if name != "present" && name != "absent" {
			return nil, "", fmt.Errorf("status_id must be Present or Absent")
		}
		return statusID, name, nil
	}

	return nil, "", fmt.Errorf("status is required (Present or Absent)")
}

func GetTrainingSessions(c *fiber.Ctx) error {
	db := database.Get()
	query := db.Preload("AgeCategory").Preload("Coach.Profile").Order("session_date DESC, session_id DESC")

	if ageCat := strings.TrimSpace(c.Query("age_category")); ageCat != "" {
		if seq := getRefSeqnumByName(ageCat); seq != nil {
			query = query.Where("age_category_id = ?", *seq)
		}
	} else if ageCatID := strings.TrimSpace(c.Query("age_category_id")); ageCatID != "" {
		query = query.Where("age_category_id = ?", ageCatID)
	}

	if date := strings.TrimSpace(c.Query("date")); date != "" {
		query = query.Where("session_date = ?", date)
	}

	var sessions []models.TrainingSession
	query.Find(&sessions)
	return c.JSON(sessions)
}

func CreateTrainingSession(c *fiber.Ctx) error {
	var req struct {
		Title         string  `json:"title"`
		SessionDate   string  `json:"session_date"`
		StartTime     *string `json:"start_time"`
		EndTime       *string `json:"end_time"`
		Venue         string  `json:"venue"`
		AgeCategory   string  `json:"age_category"`
		AgeCategoryID *int    `json:"age_category_id"`
		CoachUserID   *int    `json:"coach_user_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.SessionDate) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "title and session_date are required"})
	}

	ageCategoryID := req.AgeCategoryID
	if ageCategoryID == nil && strings.TrimSpace(req.AgeCategory) != "" {
		ageCategoryID = getRefSeqnumByName(req.AgeCategory)
	}

	coachID := req.CoachUserID
	if coachID == nil {
		if uid, ok := c.Locals("user_id").(int); ok && uid > 0 {
			coachID = &uid
		}
	}

	session := models.TrainingSession{
		Title:         strings.TrimSpace(req.Title),
		SessionDate:   strings.TrimSpace(req.SessionDate),
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		Venue:         strings.TrimSpace(req.Venue),
		AgeCategoryID: ageCategoryID,
		CoachUserID:   coachID,
	}

	db := database.Get()
	if err := db.Create(&session).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	db.Preload("AgeCategory").Preload("Coach.Profile").First(&session, session.SessionID)
	return c.Status(201).JSON(fiber.Map{
		"message": "Training session created",
		"data":    session,
	})
}

func UpdateTrainingSession(c *fiber.Ctx) error {
	db := database.Get()
	var session models.TrainingSession
	if err := db.First(&session, c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Training session not found"})
	}

	var req struct {
		Title         string  `json:"title"`
		SessionDate   string  `json:"session_date"`
		StartTime     *string `json:"start_time"`
		EndTime       *string `json:"end_time"`
		Venue         string  `json:"venue"`
		AgeCategory   string  `json:"age_category"`
		AgeCategoryID *int    `json:"age_category_id"`
		CoachUserID   *int    `json:"coach_user_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if strings.TrimSpace(req.Title) != "" {
		session.Title = strings.TrimSpace(req.Title)
	}
	if strings.TrimSpace(req.SessionDate) != "" {
		session.SessionDate = strings.TrimSpace(req.SessionDate)
	}
	if req.StartTime != nil {
		session.StartTime = req.StartTime
	}
	if req.EndTime != nil {
		session.EndTime = req.EndTime
	}
	if req.Venue != "" {
		session.Venue = strings.TrimSpace(req.Venue)
	}
	if req.AgeCategoryID != nil {
		session.AgeCategoryID = req.AgeCategoryID
	} else if strings.TrimSpace(req.AgeCategory) != "" {
		session.AgeCategoryID = getRefSeqnumByName(req.AgeCategory)
	}
	if req.CoachUserID != nil {
		session.CoachUserID = req.CoachUserID
	}

	db.Save(&session)
	db.Preload("AgeCategory").Preload("Coach.Profile").First(&session, session.SessionID)
	return c.JSON(fiber.Map{
		"message": "Training session updated",
		"data":    session,
	})
}

func DeleteTrainingSession(c *fiber.Ctx) error {
	database.Get().Delete(&models.TrainingSession{}, c.Params("id"))
	return c.JSON(fiber.Map{"message": "Training session deleted"})
}

func GetSessionAttendance(c *fiber.Ctx) error {
	sessionID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid session_id"})
	}

	db := database.Get()
	var session models.TrainingSession
	if err := db.Preload("AgeCategory").Preload("Coach.Profile").First(&session, sessionID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Training session not found"})
	}

	playerQuery := db.Model(&models.User{}).
		Preload("Profile").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.user_id").
		Where("users.role_id = ?", models.RefRolePlayer)

	userID, _ := c.Locals("user_id").(int)
	roleID, _ := c.Locals("role_id").(int)
	if roleID == models.RefRoleParent {
		childIDs := getParentChildIDs(userID)
		if len(childIDs) == 0 {
			return c.JSON(fiber.Map{
				"session": session,
				"summary": fiber.Map{"total_players": 0, "present": 0, "absent": 0, "unmarked": 0},
				"players": []fiber.Map{},
			})
		}
		playerQuery = playerQuery.Where("users.user_id IN ?", childIDs)
	} else if roleID == models.RefRolePlayer {
		playerQuery = playerQuery.Where("users.user_id = ?", userID)
	}

	if session.AgeCategoryID != nil && strings.ToLower(c.Query("all_players")) != "true" {
		playerQuery = playerQuery.Where("user_profiles.age_category_id = ?", *session.AgeCategoryID)
	}

	var players []models.User
	playerQuery.Order("user_profiles.last_name ASC, user_profiles.first_name ASC").Find(&players)

	var records []models.PlayerAttendance
	db.Preload("Status").Where("session_id = ?", sessionID).Find(&records)

	attByPlayer := make(map[int]models.PlayerAttendance, len(records))
	for _, r := range records {
		attByPlayer[r.PlayerUserID] = r
	}

	refMap := getRefMap()
	presentCount, absentCount, unmarkedCount := 0, 0, 0

	rows := make([]fiber.Map, 0, len(players))
	for _, p := range players {
		firstName, lastName, avatarURL := "", "", ""
		var jersey *int
		ageCategory, position := "", ""

		if p.Profile != nil {
			firstName = p.Profile.FirstName
			lastName = p.Profile.LastName
			avatarURL = p.Profile.AvatarURL
			jersey = p.Profile.JerseyNumber
			if p.Profile.AgeCategoryID != nil {
				ageCategory = refMap[*p.Profile.AgeCategoryID]
			}
			if p.Profile.PositionID != nil {
				position = refMap[*p.Profile.PositionID]
			}
		}

		var attendanceID *int
		var statusID *int
		statusName := ""
		var checkInTime *time.Time
		remarks := ""

		if rec, ok := attByPlayer[p.UserID]; ok {
			idCopy := rec.AttendanceID
			attendanceID = &idCopy
			statusID = rec.StatusID
			checkInTime = rec.CheckInTime
			remarks = rec.Remarks
			if rec.Status != nil {
				statusName = rec.Status.Details
			} else if rec.StatusID != nil {
				statusName = refMap[*rec.StatusID]
			}
		}

		lowerStatus := strings.ToLower(statusName)
		attended := lowerStatus == "present"

		switch lowerStatus {
		case "present":
			presentCount++
		case "absent":
			absentCount++
		default:
			unmarkedCount++
		}

		rows = append(rows, fiber.Map{
			"player_user_id": p.UserID,
			"username":       p.Username,
			"first_name":     firstName,
			"last_name":      lastName,
			"jersey_number":  jersey,
			"age_category":   ageCategory,
			"position":       position,
			"avatar_url":     avatarURL,
			"attendance_id":  attendanceID,
			"status_id":      statusID,
			"status":         statusName,
			"attended":       attended,
			"check_in_time":  checkInTime,
			"remarks":        remarks,
		})
	}

	return c.JSON(fiber.Map{
		"session": session,
		"summary": fiber.Map{
			"total_players": len(rows),
			"present":       presentCount,
			"absent":        absentCount,
			"unmarked":      unmarkedCount,
		},
		"players": rows,
	})
}

func upsertPlayerAttendance(sessionID, playerUserID int, statusID *int, statusName, remarks string, recordedBy *int) (models.PlayerAttendance, error) {
	resolvedID, lowerStatus, err := resolveAttendanceStatus(statusID, statusName)
	if err != nil {
		return models.PlayerAttendance{}, err
	}

	var checkIn *time.Time
	if lowerStatus == "present" {
		now := time.Now()
		checkIn = &now
	}

	db := database.Get()
	var record models.PlayerAttendance

	err = db.Where("session_id = ? AND player_user_id = ?", sessionID, playerUserID).First(&record).Error
	if err != nil {
		record = models.PlayerAttendance{
			SessionID:    sessionID,
			PlayerUserID: playerUserID,
			StatusID:     resolvedID,
			CheckInTime:  checkIn,
			Remarks:      remarks,
			RecordedBy:   recordedBy,
		}
		err = db.Create(&record).Error
	} else {
		record.StatusID = resolvedID
		record.Remarks = remarks
		record.RecordedBy = recordedBy
		record.CheckInTime = checkIn
		err = db.Save(&record).Error
	}

	return record, err
}

func RecordAttendance(c *fiber.Ctx) error {
	var req struct {
		SessionID    int    `json:"session_id"`
		PlayerUserID int    `json:"player_user_id"`
		StatusID     *int   `json:"status_id"`
		Status       string `json:"status"`
		Remarks      string `json:"remarks"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.SessionID == 0 || req.PlayerUserID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "session_id and player_user_id are required"})
	}

	var recordedBy *int
	if uid, ok := c.Locals("user_id").(int); ok && uid > 0 {
		recordedBy = &uid
	}

	record, err := upsertPlayerAttendance(req.SessionID, req.PlayerUserID, req.StatusID, req.Status, req.Remarks, recordedBy)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	database.Get().Preload("Status").Preload("Player.Profile").First(&record, record.AttendanceID)
	return c.JSON(fiber.Map{
		"message": "Attendance saved",
		"data":    record,
	})
}

func RecordBulkAttendance(c *fiber.Ctx) error {
	var req struct {
		SessionID int `json:"session_id"`
		Records   []struct {
			PlayerUserID int    `json:"player_user_id"`
			StatusID     *int   `json:"status_id"`
			Status       string `json:"status"`
			Remarks      string `json:"remarks"`
		} `json:"records"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.SessionID == 0 || len(req.Records) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "session_id and records are required"})
	}

	var recordedBy *int
	if uid, ok := c.Locals("user_id").(int); ok && uid > 0 {
		recordedBy = &uid
	}

	saved := 0
	for _, item := range req.Records {
		if item.PlayerUserID == 0 {
			continue
		}
		if _, err := upsertPlayerAttendance(req.SessionID, item.PlayerUserID, item.StatusID, item.Status, item.Remarks, recordedBy); err == nil {
			saved++
		}
	}

	return c.JSON(fiber.Map{
		"message": "Bulk attendance saved",
		"saved":   saved,
	})
}

func GetPlayerAttendanceHistory(c *fiber.Ctx) error {
	playerID, err := strconv.Atoi(c.Params("player_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid player_id"})
	}

	if !canAccessPlayer(c, playerID) {
		return c.Status(403).JSON(fiber.Map{"error": "Forbidden: you are not allowed to view this player's attendance"})
	}

	db := database.Get()
	var player models.User
	if err := db.Preload("Profile").First(&player, playerID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Player not found"})
	}

	targetDate := strings.TrimSpace(c.Query("date"))
	month := strings.TrimSpace(c.Query("month"))
	fromDate := strings.TrimSpace(c.Query("from"))
	toDate := strings.TrimSpace(c.Query("to"))
	today := time.Now().Format("2006-01-02")

	query := db.Preload("Session.AgeCategory").
		Preload("Status").
		Joins("JOIN training_sessions ON training_sessions.session_id = player_attendance.session_id").
		Where("player_attendance.player_user_id = ?", playerID)

	if targetDate != "" {
		query = query.Where("training_sessions.session_date = ?", targetDate)
	}
	if month != "" {
		query = query.Where("TO_CHAR(training_sessions.session_date, 'YYYY-MM') = ?", month)
	}
	if fromDate != "" {
		query = query.Where("training_sessions.session_date >= ?", fromDate)
	}
	if toDate != "" {
		query = query.Where("training_sessions.session_date <= ?", toDate)
	}

	var records []models.PlayerAttendance
	query.Order("training_sessions.session_date DESC, player_attendance.created_at DESC").Find(&records)

	refMap := getRefMap()
	firstName, lastName, avatarURL, ageCategory, position := "", "", "", "", ""
	var jersey *int
	if player.Profile != nil {
		firstName = player.Profile.FirstName
		lastName = player.Profile.LastName
		avatarURL = player.Profile.AvatarURL
		jersey = player.Profile.JerseyNumber
		if player.Profile.AgeCategoryID != nil {
			ageCategory = refMap[*player.Profile.AgeCategoryID]
		}
		if player.Profile.PositionID != nil {
			position = refMap[*player.Profile.PositionID]
		}
	}

	present, absent := 0, 0
	attendedToday := false
	todayStatus := ""
	dailyList := make([]fiber.Map, 0, len(records))

	for _, r := range records {
		statusName := ""
		if r.Status != nil {
			statusName = r.Status.Details
		}

		sessionDate, sessionTitle, venue := "", "", ""
		if r.Session != nil {
			sessionDate = r.Session.SessionDate
			if len(sessionDate) >= 10 {
				sessionDate = sessionDate[:10]
			}
			sessionTitle = r.Session.Title
			venue = r.Session.Venue
		}

		lowerStatus := strings.ToLower(statusName)
		if lowerStatus != "present" && lowerStatus != "absent" {
			continue
		}

		attended := lowerStatus == "present"
		if attended {
			present++
		} else {
			absent++
		}

		if sessionDate == today {
			todayStatus = statusName
			attendedToday = attended
		}

		dailyList = append(dailyList, fiber.Map{
			"attendance_id": r.AttendanceID,
			"session_id":    r.SessionID,
			"date":          sessionDate,
			"session_title": sessionTitle,
			"venue":         venue,
			"status_id":     r.StatusID,
			"status":        statusName,
			"attended":      attended,
			"check_in_time": r.CheckInTime,
			"remarks":       r.Remarks,
		})
	}

	totalCount := present + absent
	rate := 0.0
	if totalCount > 0 {
		rate = float64(present) / float64(totalCount) * 100
	}

	return c.JSON(fiber.Map{
		"player": fiber.Map{
			"player_user_id": player.UserID,
			"username":       player.Username,
			"first_name":     firstName,
			"last_name":      lastName,
			"jersey_number":  jersey,
			"age_category":   ageCategory,
			"position":       position,
			"avatar_url":     avatarURL,
		},
		"today":          today,
		"attended_today": attendedToday,
		"today_status":   todayStatus,
		"summary": fiber.Map{
			"total":           totalCount,
			"present":         present,
			"absent":          absent,
			"attendance_rate": rate,
		},
		"history": dailyList,
	})
}

func GetAllPlayersAttendanceHistory(c *fiber.Ctx) error {
	db := database.Get()
	today := time.Now().Format("2006-01-02")

	playerQuery := db.Model(&models.User{}).
		Preload("Profile").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.user_id").
		Where("users.role_id = ?", models.RefRolePlayer)

	userID, _ := c.Locals("user_id").(int)
	roleID, _ := c.Locals("role_id").(int)
	if roleID == models.RefRoleParent {
		childIDs := getParentChildIDs(userID)
		if len(childIDs) == 0 {
			return c.JSON(fiber.Map{"today": today, "count": 0, "players": []fiber.Map{}})
		}
		playerQuery = playerQuery.Where("users.user_id IN ?", childIDs)
	} else if roleID == models.RefRolePlayer {
		playerQuery = playerQuery.Where("users.user_id = ?", userID)
	}

	if ageCat := strings.TrimSpace(c.Query("age_category")); ageCat != "" {
		if seq := getRefSeqnumByName(ageCat); seq != nil {
			playerQuery = playerQuery.Where("user_profiles.age_category_id = ?", *seq)
		}
	} else if ageCatID := strings.TrimSpace(c.Query("age_category_id")); ageCatID != "" {
		playerQuery = playerQuery.Where("user_profiles.age_category_id = ?", ageCatID)
	}

	var players []models.User
	playerQuery.Order("user_profiles.last_name ASC, user_profiles.first_name ASC").Find(&players)

	attQuery := db.Preload("Session").
		Preload("Status").
		Joins("JOIN training_sessions ON training_sessions.session_id = player_attendance.session_id")

	if month := strings.TrimSpace(c.Query("month")); month != "" {
		attQuery = attQuery.Where("TO_CHAR(training_sessions.session_date, 'YYYY-MM') = ?", month)
	}
	if fromDate := strings.TrimSpace(c.Query("from")); fromDate != "" {
		attQuery = attQuery.Where("training_sessions.session_date >= ?", fromDate)
	}
	if toDate := strings.TrimSpace(c.Query("to")); toDate != "" {
		attQuery = attQuery.Where("training_sessions.session_date <= ?", toDate)
	}

	var allRecords []models.PlayerAttendance
	attQuery.Order("training_sessions.session_date DESC, player_attendance.created_at DESC").Find(&allRecords)

	recordsByPlayer := make(map[int][]models.PlayerAttendance)
	for _, r := range allRecords {
		recordsByPlayer[r.PlayerUserID] = append(recordsByPlayer[r.PlayerUserID], r)
	}

	refMap := getRefMap()
	result := make([]fiber.Map, 0, len(players))

	for _, p := range players {
		firstName, lastName, avatarURL, ageCategory, position := "", "", "", "", ""
		var jersey *int
		if p.Profile != nil {
			firstName = p.Profile.FirstName
			lastName = p.Profile.LastName
			avatarURL = p.Profile.AvatarURL
			jersey = p.Profile.JerseyNumber
			if p.Profile.AgeCategoryID != nil {
				ageCategory = refMap[*p.Profile.AgeCategoryID]
			}
			if p.Profile.PositionID != nil {
				position = refMap[*p.Profile.PositionID]
			}
		}

		pRecords := recordsByPlayer[p.UserID]
		present, absent := 0, 0
		attendedToday := false
		todayStatus := ""
		history := make([]fiber.Map, 0, len(pRecords))

		for _, r := range pRecords {
			statusName := ""
			if r.Status != nil {
				statusName = r.Status.Details
			}
			sessionDate, sessionTitle := "", ""
			if r.Session != nil {
				sessionDate = r.Session.SessionDate
				if len(sessionDate) >= 10 {
					sessionDate = sessionDate[:10]
				}
				sessionTitle = r.Session.Title
			}

			lowerStatus := strings.ToLower(statusName)
			if lowerStatus != "present" && lowerStatus != "absent" {
				continue
			}

			attended := lowerStatus == "present"
			if attended {
				present++
			} else {
				absent++
			}

			if sessionDate == today {
				todayStatus = statusName
				attendedToday = attended
			}

			history = append(history, fiber.Map{
				"attendance_id": r.AttendanceID,
				"session_id":    r.SessionID,
				"date":          sessionDate,
				"session_title": sessionTitle,
				"status":        statusName,
				"attended":      attended,
				"check_in_time": r.CheckInTime,
				"remarks":       r.Remarks,
			})
		}

		totalCount := present + absent
		rate := 0.0
		if totalCount > 0 {
			rate = float64(present) / float64(totalCount) * 100
		}

		result = append(result, fiber.Map{
			"player_user_id": p.UserID,
			"username":       p.Username,
			"first_name":     firstName,
			"last_name":      lastName,
			"jersey_number":  jersey,
			"age_category":   ageCategory,
			"position":       position,
			"avatar_url":     avatarURL,
			"attended_today": attendedToday,
			"today_status":   todayStatus,
			"summary": fiber.Map{
				"total":           totalCount,
				"present":         present,
				"absent":          absent,
				"attendance_rate": rate,
			},
			"history": history,
		})
	}

	return c.JSON(fiber.Map{
		"today":   today,
		"count":   len(result),
		"players": result,
	})
}

func GetDailyAttendance(c *fiber.Ctx) error {
	date := strings.TrimSpace(c.Query("date"))
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	db := database.Get()

	playerQuery := db.Model(&models.User{}).
		Preload("Profile").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.user_id").
		Where("users.role_id = ?", models.RefRolePlayer)

	userID, _ := c.Locals("user_id").(int)
	roleID, _ := c.Locals("role_id").(int)
	if roleID == models.RefRoleParent {
		childIDs := getParentChildIDs(userID)
		if len(childIDs) == 0 {
			return c.JSON(fiber.Map{
				"date":       date,
				"session_id": nil,
				"summary":    fiber.Map{"total_players": 0, "present": 0, "absent": 0, "unmarked": 0},
				"players":    []fiber.Map{},
			})
		}
		playerQuery = playerQuery.Where("users.user_id IN ?", childIDs)
	} else if roleID == models.RefRolePlayer {
		playerQuery = playerQuery.Where("users.user_id = ?", userID)
	}

	if ageCat := strings.TrimSpace(c.Query("age_category")); ageCat != "" {
		if seq := getRefSeqnumByName(ageCat); seq != nil {
			playerQuery = playerQuery.Where("user_profiles.age_category_id = ?", *seq)
		}
	} else if ageCatID := strings.TrimSpace(c.Query("age_category_id")); ageCatID != "" {
		playerQuery = playerQuery.Where("user_profiles.age_category_id = ?", ageCatID)
	}

	var players []models.User
	playerQuery.Order("user_profiles.last_name ASC, user_profiles.first_name ASC").Find(&players)

	var sessions []models.TrainingSession
	db.Where("session_date = ?", date).Order("session_id ASC").Find(&sessions)

	attByPlayer := make(map[int]models.PlayerAttendance)
	if len(sessions) > 0 {
		sessionIDs := make([]int, 0, len(sessions))
		for _, s := range sessions {
			sessionIDs = append(sessionIDs, s.SessionID)
		}
		var records []models.PlayerAttendance
		db.Preload("Status").Where("session_id IN ?", sessionIDs).Find(&records)
		for _, r := range records {
			attByPlayer[r.PlayerUserID] = r
		}
	}

	refMap := getRefMap()
	presentCount, absentCount, unmarkedCount := 0, 0, 0

	rows := make([]fiber.Map, 0, len(players))
	for _, p := range players {
		firstName, lastName, avatarURL := "", "", ""
		var jersey *int
		ageCategory, position := "", ""

		if p.Profile != nil {
			firstName = p.Profile.FirstName
			lastName = p.Profile.LastName
			avatarURL = p.Profile.AvatarURL
			jersey = p.Profile.JerseyNumber
			if p.Profile.AgeCategoryID != nil {
				ageCategory = refMap[*p.Profile.AgeCategoryID]
			}
			if p.Profile.PositionID != nil {
				position = refMap[*p.Profile.PositionID]
			}
		}

		var attendanceID *int
		var sessionID *int
		var statusID *int
		statusName := ""
		var checkInTime *time.Time
		remarks := ""

		if rec, ok := attByPlayer[p.UserID]; ok {
			aID := rec.AttendanceID
			sID := rec.SessionID
			attendanceID = &aID
			sessionID = &sID
			statusID = rec.StatusID
			checkInTime = rec.CheckInTime
			remarks = rec.Remarks
			if rec.Status != nil {
				statusName = rec.Status.Details
			} else if rec.StatusID != nil {
				statusName = refMap[*rec.StatusID]
			}
		}

		lowerStatus := strings.ToLower(statusName)
		attended := lowerStatus == "present"

		switch lowerStatus {
		case "present":
			presentCount++
		case "absent":
			absentCount++
		default:
			statusName = ""
			unmarkedCount++
		}

		rows = append(rows, fiber.Map{
			"player_user_id": p.UserID,
			"username":       p.Username,
			"first_name":     firstName,
			"last_name":      lastName,
			"jersey_number":  jersey,
			"age_category":   ageCategory,
			"position":       position,
			"avatar_url":     avatarURL,
			"session_id":     sessionID,
			"attendance_id":  attendanceID,
			"status_id":      statusID,
			"status":         statusName,
			"attended":       attended,
			"check_in_time":  checkInTime,
			"remarks":        remarks,
		})
	}

	var defaultSessionID *int
	if len(sessions) > 0 {
		id := sessions[0].SessionID
		defaultSessionID = &id
	}

	return c.JSON(fiber.Map{
		"date":       date,
		"session_id": defaultSessionID,
		"summary": fiber.Map{
			"total_players": len(rows),
			"present":       presentCount,
			"absent":        absentCount,
			"unmarked":      unmarkedCount,
		},
		"players": rows,
	})
}

func RecordDailyAttendance(c *fiber.Ctx) error {
	var req struct {
		Date         string `json:"date"`
		PlayerUserID int    `json:"player_user_id"`
		StatusID     *int   `json:"status_id"`
		Status       string `json:"status"`
		Remarks      string `json:"remarks"`
		Records      []struct {
			PlayerUserID int    `json:"player_user_id"`
			StatusID     *int   `json:"status_id"`
			Status       string `json:"status"`
			Remarks      string `json:"remarks"`
		} `json:"records"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	date := strings.TrimSpace(req.Date)
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	var recordedBy *int
	if uid, ok := c.Locals("user_id").(int); ok && uid > 0 {
		recordedBy = &uid
	}

	db := database.Get()
	var session models.TrainingSession
	if err := db.Where("session_date = ?", date).Order("session_id ASC").First(&session).Error; err != nil {
		session = models.TrainingSession{
			Title:       "Daily Training - " + date,
			SessionDate: date,
			CoachUserID: recordedBy,
		}
		if err := db.Create(&session).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}

	if len(req.Records) > 0 {
		saved := 0
		for _, item := range req.Records {
			if item.PlayerUserID == 0 {
				continue
			}
			if _, err := upsertPlayerAttendance(session.SessionID, item.PlayerUserID, item.StatusID, item.Status, item.Remarks, recordedBy); err == nil {
				saved++
			}
		}
		return c.JSON(fiber.Map{
			"message":    "Daily attendance saved",
			"date":       date,
			"session_id": session.SessionID,
			"saved":      saved,
		})
	}

	if req.PlayerUserID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "player_user_id or records is required"})
	}

	record, err := upsertPlayerAttendance(session.SessionID, req.PlayerUserID, req.StatusID, req.Status, req.Remarks, recordedBy)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	db.Preload("Status").Preload("Player.Profile").First(&record, record.AttendanceID)
	return c.JSON(fiber.Map{
		"message":    "Daily attendance saved",
		"date":       date,
		"session_id": session.SessionID,
		"data":       record,
	})
}
