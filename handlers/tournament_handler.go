package handlers

import (
	"strconv"
	"strings"
	"time"

	"alabapi/database"
	"alabapi/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func resolveAgeCategories(names []string, singleName string, csvIDs string, singleID *int) (string, *int) {
	seen := make(map[int]bool)
	ids := make([]int, 0)

	addID := func(id int) {
		if id > 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	if singleID != nil && *singleID > 0 {
		addID(*singleID)
	}
	if strings.TrimSpace(singleName) != "" {
		if seq := getRefSeqnumByName(singleName); seq != nil {
			addID(*seq)
		}
	}
	for _, name := range names {
		if seq := getRefSeqnumByName(name); seq != nil {
			addID(*seq)
		}
	}
	for _, id := range parseModuleIDs(csvIDs) {
		addID(id)
	}

	if len(ids) == 0 {
		return "", nil
	}

	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.Itoa(id))
	}
	first := ids[0]
	return strings.Join(parts, ","), &first
}

func formatMatch(m models.TournamentMatch, refMap map[int]string) fiber.Map {
	ageCat := ""
	if m.AgeCategory != nil {
		ageCat = m.AgeCategory.Details
	} else if m.AgeCategoryID != nil {
		ageCat = refMap[*m.AgeCategoryID]
	}

	status := ""
	if m.Status != nil {
		status = m.Status.Details
	} else if m.StatusID != nil {
		status = refMap[*m.StatusID]
	}

	matchDate := m.MatchDate
	if len(matchDate) >= 10 {
		matchDate = matchDate[:10]
	}

	return fiber.Map{
		"match_id":        m.MatchID,
		"tournament_id":   m.TournamentID,
		"age_category_id": m.AgeCategoryID,
		"age_category":    ageCat,
		"game_label":      m.GameLabel,
		"opponent":        m.Opponent,
		"match_date":      matchDate,
		"match_time":      m.MatchTime,
		"pitch":           m.Pitch,
		"alab_score":      m.AlabScore,
		"opponent_score":  m.OpponentScore,
		"status_id":       m.StatusID,
		"status":          status,
		"remarks":         m.Remarks,
	}
}

func formatRoster(r models.TournamentRoster, refMap map[int]string) fiber.Map {
	firstName, lastName, ageCat, pos, avatar := "", "", "", "", ""
	var jersey *int
	if r.Player != nil {
		firstName = r.Player.FirstName
		lastName = r.Player.LastName
		jersey = r.Player.JerseyNumber
		avatar = r.Player.AvatarURL
		if r.Player.AgeCategoryID != nil {
			ageCat = refMap[*r.Player.AgeCategoryID]
		}
		if r.Player.PositionID != nil {
			pos = refMap[*r.Player.PositionID]
		}
	}

	return fiber.Map{
		"roster_id":      r.RosterID,
		"tournament_id":  r.TournamentID,
		"player_id":      r.PlayerID,
		"first_name":     firstName,
		"last_name":      lastName,
		"full_name":      strings.TrimSpace(firstName + " " + lastName),
		"jersey_number":  jersey,
		"age_category":   ageCat,
		"position":       pos,
		"avatar_url":     avatar,
		"parent_consent": r.ParentConsent,
		"remarks":        r.Remarks,
	}
}

func formatTournament(t models.Tournament, refMap map[int]string, includeDetails bool) fiber.Map {
	catIDs := parseModuleIDs(t.AgeCategoryIDs)
	if len(catIDs) == 0 && t.AgeCategoryID != nil {
		catIDs = []int{*t.AgeCategoryID}
	}

	catNames := make([]string, 0, len(catIDs))
	for _, id := range catIDs {
		if name, ok := refMap[id]; ok {
			catNames = append(catNames, name)
		}
	}

	status := ""
	if t.Status != nil {
		status = t.Status.Details
	} else if t.StatusID != nil {
		status = refMap[*t.StatusID]
	}

	startDate := t.StartDate
	if len(startDate) >= 10 {
		startDate = startDate[:10]
	}
	var endDate *string
	if t.EndDate != nil {
		ed := *t.EndDate
		if len(ed) >= 10 {
			ed = ed[:10]
		}
		endDate = &ed
	}

	matches := make([]fiber.Map, 0, len(t.Matches))
	matchesByCat := make(map[string][]fiber.Map)
	for _, m := range t.Matches {
		fm := formatMatch(m, refMap)
		matches = append(matches, fm)
		catKey, _ := fm["age_category"].(string)
		if catKey == "" {
			catKey = "Open"
		}
		matchesByCat[catKey] = append(matchesByCat[catKey], fm)
	}

	res := fiber.Map{
		"tournament_id":    t.TournamentID,
		"title":            t.Title,
		"description":      t.Description,
		"venue":            t.Venue,
		"start_date":       startDate,
		"end_date":         endDate,
		"age_category_ids": catIDs,
		"age_categories":   catNames,
		"status_id":        t.StatusID,
		"status":           status,
		"registration_fee": t.RegistrationFee,
		"banner_url":       t.BannerURL,
		"match_count":      len(matches),
		"roster_count":     len(t.Rosters),
		"matches":          matches,
		"created_at":       t.CreatedAt,
	}

	if includeDetails {
		rosters := make([]fiber.Map, 0, len(t.Rosters))
		for _, r := range t.Rosters {
			rosters = append(rosters, formatRoster(r, refMap))
		}
		res["matches_by_category"] = matchesByCat
		res["rosters"] = rosters
	}

	return res
}

func GetTournaments(c *fiber.Ctx) error {
	db := database.Get()
	query := db.Model(&models.Tournament{}).
		Preload("Status").
		Preload("Matches", func(tx *gorm.DB) *gorm.DB {
			return tx.Preload("AgeCategory").Preload("Status").Order("match_date ASC, match_time ASC, match_id ASC")
		}).
		Preload("Rosters")

	if status := strings.TrimSpace(c.Query("status")); status != "" {
		if seq := getRefSeqnumByName(status); seq != nil {
			query = query.Where("status_id = ?", *seq)
		}
	}

	var tournaments []models.Tournament
	query.Order("start_date ASC, tournament_id DESC").Find(&tournaments)

	filterCat := strings.TrimSpace(c.Query("age_category"))
	var filterCatID int
	if filterCat != "" {
		if seq := getRefSeqnumByName(filterCat); seq != nil {
			filterCatID = *seq
		}
	}

	refMap := getRefMap()
	result := make([]fiber.Map, 0, len(tournaments))
	for _, t := range tournaments {
		if filterCatID > 0 {
			matched := false
			if t.AgeCategoryID != nil && *t.AgeCategoryID == filterCatID {
				matched = true
			}
			for _, id := range parseModuleIDs(t.AgeCategoryIDs) {
				if id == filterCatID {
					matched = true
					break
				}
			}
			for _, m := range t.Matches {
				if m.AgeCategoryID != nil && *m.AgeCategoryID == filterCatID {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		result = append(result, formatTournament(t, refMap, false))
	}

	return c.JSON(result)
}

func GetTournamentByID(c *fiber.Ctx) error {
	db := database.Get()
	filterCat := strings.TrimSpace(c.Query("age_category"))
	var filterCatID *int
	if filterCat != "" {
		filterCatID = getRefSeqnumByName(filterCat)
	}

	var t models.Tournament
	err := db.Preload("Status").
		Preload("Matches", func(tx *gorm.DB) *gorm.DB {
			q := tx.Preload("AgeCategory").Preload("Status").Order("match_date ASC, match_time ASC, match_id ASC")
			if filterCatID != nil {
				q = q.Where("age_category_id = ?", *filterCatID)
			}
			return q
		}).
		Preload("Rosters.Player").
		First(&t, c.Params("id")).Error

	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Tournament not found"})
	}

	if filterCatID != nil {
		filteredRosters := make([]models.TournamentRoster, 0)
		for _, r := range t.Rosters {
			if r.Player != nil && r.Player.AgeCategoryID != nil && *r.Player.AgeCategoryID == *filterCatID {
				filteredRosters = append(filteredRosters, r)
			}
		}
		t.Rosters = filteredRosters
	}

	return c.JSON(formatTournament(t, getRefMap(), true))
}

type MatchInput struct {
	AgeCategory   string  `json:"age_category"`
	AgeCategoryID *int    `json:"age_category_id"`
	GameLabel     string  `json:"game_label"`
	Opponent      string  `json:"opponent"`
	MatchDate     string  `json:"match_date"`
	MatchTime     *string `json:"match_time"`
	Pitch         string  `json:"pitch"`
	AlabScore     *int    `json:"alab_score"`
	OpponentScore *int    `json:"opponent_score"`
	Status        string  `json:"status"`
	StatusID      *int    `json:"status_id"`
	Remarks       string  `json:"remarks"`
}

func CreateTournament(c *fiber.Ctx) error {
	var req struct {
		Title           string       `json:"title"`
		Description     string       `json:"description"`
		Venue           string       `json:"venue"`
		StartDate       string       `json:"start_date"`
		EndDate         *string      `json:"end_date"`
		AgeCategory     string       `json:"age_category"`
		AgeCategoryID   *int         `json:"age_category_id"`
		AgeCategories   []string     `json:"age_categories"`
		AgeCategoryIDs  string       `json:"age_category_ids"`
		Status          string       `json:"status"`
		StatusID        *int         `json:"status_id"`
		RegistrationFee float64      `json:"registration_fee"`
		BannerURL       string       `json:"banner_url"`
		Matches         []MatchInput `json:"matches"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Venue) == "" || strings.TrimSpace(req.StartDate) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "title, venue, and start_date are required"})
	}

	catCSV, firstCatID := resolveAgeCategories(req.AgeCategories, req.AgeCategory, req.AgeCategoryIDs, req.AgeCategoryID)

	statusID := req.StatusID
	if statusID == nil && strings.TrimSpace(req.Status) != "" {
		statusID = getRefSeqnumByName(req.Status)
	}
	if statusID == nil {
		upcoming := 37
		statusID = &upcoming
	}

	t := models.Tournament{
		Title:           strings.TrimSpace(req.Title),
		Description:     strings.TrimSpace(req.Description),
		Venue:           strings.TrimSpace(req.Venue),
		StartDate:       strings.TrimSpace(req.StartDate),
		EndDate:         req.EndDate,
		AgeCategoryID:   firstCatID,
		AgeCategoryIDs:  catCSV,
		StatusID:        statusID,
		RegistrationFee: req.RegistrationFee,
		BannerURL:       strings.TrimSpace(req.BannerURL),
	}

	db := database.Get()
	if err := db.Create(&t).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	for _, m := range req.Matches {
		if strings.TrimSpace(m.Opponent) == "" {
			continue
		}
		mCatID := m.AgeCategoryID
		if mCatID == nil && strings.TrimSpace(m.AgeCategory) != "" {
			mCatID = getRefSeqnumByName(m.AgeCategory)
		}
		mStatusID := m.StatusID
		if mStatusID == nil && strings.TrimSpace(m.Status) != "" {
			mStatusID = getRefSeqnumByName(m.Status)
		}
		if mStatusID == nil {
			upcoming := 37
			mStatusID = &upcoming
		}
		mDate := strings.TrimSpace(m.MatchDate)
		if mDate == "" {
			mDate = t.StartDate
		}
		label := strings.TrimSpace(m.GameLabel)
		if label == "" {
			label = "Game 1"
		}
		db.Create(&models.TournamentMatch{
			TournamentID:  t.TournamentID,
			AgeCategoryID: mCatID,
			GameLabel:     label,
			Opponent:      strings.TrimSpace(m.Opponent),
			MatchDate:     mDate,
			MatchTime:     m.MatchTime,
			Pitch:         strings.TrimSpace(m.Pitch),
			AlabScore:     m.AlabScore,
			OpponentScore: m.OpponentScore,
			StatusID:      mStatusID,
			Remarks:       strings.TrimSpace(m.Remarks),
		})
	}

	db.Preload("Status").
		Preload("Matches", func(tx *gorm.DB) *gorm.DB {
			return tx.Preload("AgeCategory").Preload("Status").Order("match_date ASC, match_time ASC, match_id ASC")
		}).
		Preload("Rosters.Player").
		First(&t, t.TournamentID)

	return c.Status(201).JSON(fiber.Map{
		"message": "Tournament created",
		"data":    formatTournament(t, getRefMap(), true),
	})
}

func UpdateTournament(c *fiber.Ctx) error {
	db := database.Get()
	var t models.Tournament
	if err := db.First(&t, c.Params("id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Tournament not found"})
	}

	var req struct {
		Title           string   `json:"title"`
		Description     string   `json:"description"`
		Venue           string   `json:"venue"`
		StartDate       string   `json:"start_date"`
		EndDate         *string  `json:"end_date"`
		AgeCategory     string   `json:"age_category"`
		AgeCategoryID   *int     `json:"age_category_id"`
		AgeCategories   []string `json:"age_categories"`
		AgeCategoryIDs  string   `json:"age_category_ids"`
		Status          string   `json:"status"`
		StatusID        *int     `json:"status_id"`
		RegistrationFee *float64 `json:"registration_fee"`
		BannerURL       string   `json:"banner_url"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if strings.TrimSpace(req.Title) != "" {
		t.Title = strings.TrimSpace(req.Title)
	}
	if req.Description != "" {
		t.Description = strings.TrimSpace(req.Description)
	}
	if strings.TrimSpace(req.Venue) != "" {
		t.Venue = strings.TrimSpace(req.Venue)
	}
	if strings.TrimSpace(req.StartDate) != "" {
		t.StartDate = strings.TrimSpace(req.StartDate)
	}
	if req.EndDate != nil {
		t.EndDate = req.EndDate
	}
	if len(req.AgeCategories) > 0 || strings.TrimSpace(req.AgeCategory) != "" || strings.TrimSpace(req.AgeCategoryIDs) != "" || req.AgeCategoryID != nil {
		catCSV, firstCatID := resolveAgeCategories(req.AgeCategories, req.AgeCategory, req.AgeCategoryIDs, req.AgeCategoryID)
		t.AgeCategoryIDs = catCSV
		t.AgeCategoryID = firstCatID
	}
	if req.StatusID != nil {
		t.StatusID = req.StatusID
	} else if strings.TrimSpace(req.Status) != "" {
		t.StatusID = getRefSeqnumByName(req.Status)
	}
	if req.RegistrationFee != nil {
		t.RegistrationFee = *req.RegistrationFee
	}
	if req.BannerURL != "" {
		t.BannerURL = strings.TrimSpace(req.BannerURL)
	}
	t.UpdatedAt = time.Now()

	db.Save(&t)
	db.Preload("Status").Preload("Matches.AgeCategory").Preload("Matches.Status").Preload("Rosters.Player").First(&t, t.TournamentID)

	return c.JSON(fiber.Map{
		"message": "Tournament updated",
		"data":    formatTournament(t, getRefMap(), true),
	})
}

func DeleteTournament(c *fiber.Ctx) error {
	tournamentID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid tournament_id"})
	}

	var deletedRows int64
	err = database.Get().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tournament_id = ?", tournamentID).Delete(&models.TournamentMatch{}).Error; err != nil {
			return err
		}
		if err := tx.Where("tournament_id = ?", tournamentID).Delete(&models.TournamentRoster{}).Error; err != nil {
			return err
		}
		res := tx.Where("tournament_id = ?", tournamentID).Delete(&models.Tournament{})
		if res.Error != nil {
			return res.Error
		}
		deletedRows = res.RowsAffected
		return nil
	})

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if deletedRows == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Tournament not found"})
	}

	return c.JSON(fiber.Map{
		"message":       "Tournament deleted",
		"tournament_id": tournamentID,
	})
}

func GetTournamentMatches(c *fiber.Ctx) error {
	tournamentID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid tournament_id"})
	}

	db := database.Get()
	query := db.Preload("AgeCategory").Preload("Status").Where("tournament_id = ?", tournamentID)

	if ageCat := strings.TrimSpace(c.Query("age_category")); ageCat != "" {
		if seq := getRefSeqnumByName(ageCat); seq != nil {
			query = query.Where("age_category_id = ?", *seq)
		}
	}
	if date := strings.TrimSpace(c.Query("date")); date != "" {
		query = query.Where("match_date = ?", date)
	}

	var matches []models.TournamentMatch
	query.Order("match_date ASC, match_time ASC, match_id ASC").Find(&matches)

	refMap := getRefMap()
	result := make([]fiber.Map, 0, len(matches))
	for _, m := range matches {
		result = append(result, formatMatch(m, refMap))
	}

	return c.JSON(result)
}

func AddTournamentMatches(c *fiber.Ctx) error {
	tournamentID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid tournament_id"})
	}

	db := database.Get()
	var t models.Tournament
	if err := db.First(&t, tournamentID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Tournament not found"})
	}

	var req struct {
		AgeCategory   string       `json:"age_category"`
		AgeCategoryID *int         `json:"age_category_id"`
		GameLabel     string       `json:"game_label"`
		Opponent      string       `json:"opponent"`
		MatchDate     string       `json:"match_date"`
		MatchTime     *string      `json:"match_time"`
		Pitch         string       `json:"pitch"`
		AlabScore     *int         `json:"alab_score"`
		OpponentScore *int         `json:"opponent_score"`
		Status        string       `json:"status"`
		StatusID      *int         `json:"status_id"`
		Remarks       string       `json:"remarks"`
		Matches       []MatchInput `json:"matches"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	inputs := req.Matches
	if len(inputs) == 0 && strings.TrimSpace(req.Opponent) != "" {
		inputs = []MatchInput{{
			AgeCategory:   req.AgeCategory,
			AgeCategoryID: req.AgeCategoryID,
			GameLabel:     req.GameLabel,
			Opponent:      req.Opponent,
			MatchDate:     req.MatchDate,
			MatchTime:     req.MatchTime,
			Pitch:         req.Pitch,
			AlabScore:     req.AlabScore,
			OpponentScore: req.OpponentScore,
			Status:        req.Status,
			StatusID:      req.StatusID,
			Remarks:       req.Remarks,
		}}
	}

	if len(inputs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "opponent or matches list is required"})
	}

	refMap := getRefMap()
	created := make([]fiber.Map, 0, len(inputs))

	for _, m := range inputs {
		if strings.TrimSpace(m.Opponent) == "" {
			continue
		}
		mCatID := m.AgeCategoryID
		if mCatID == nil && strings.TrimSpace(m.AgeCategory) != "" {
			mCatID = getRefSeqnumByName(m.AgeCategory)
		}
		mStatusID := m.StatusID
		if mStatusID == nil && strings.TrimSpace(m.Status) != "" {
			mStatusID = getRefSeqnumByName(m.Status)
		}
		if mStatusID == nil {
			upcoming := 37
			mStatusID = &upcoming
		}
		mDate := strings.TrimSpace(m.MatchDate)
		if mDate == "" {
			mDate = t.StartDate
		}
		label := strings.TrimSpace(m.GameLabel)
		if label == "" {
			label = "Game 1"
		}

		match := models.TournamentMatch{
			TournamentID:  tournamentID,
			AgeCategoryID: mCatID,
			GameLabel:     label,
			Opponent:      strings.TrimSpace(m.Opponent),
			MatchDate:     mDate,
			MatchTime:     m.MatchTime,
			Pitch:         strings.TrimSpace(m.Pitch),
			AlabScore:     m.AlabScore,
			OpponentScore: m.OpponentScore,
			StatusID:      mStatusID,
			Remarks:       strings.TrimSpace(m.Remarks),
		}
		if err := db.Create(&match).Error; err == nil {
			created = append(created, formatMatch(match, refMap))
		}
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Match schedule(s) added",
		"count":   len(created),
		"matches": created,
	})
}

func UpdateTournamentMatch(c *fiber.Ctx) error {
	db := database.Get()
	var match models.TournamentMatch
	if err := db.First(&match, c.Params("match_id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Match not found"})
	}

	var req MatchInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.AgeCategoryID != nil {
		match.AgeCategoryID = req.AgeCategoryID
	} else if strings.TrimSpace(req.AgeCategory) != "" {
		match.AgeCategoryID = getRefSeqnumByName(req.AgeCategory)
	}
	if strings.TrimSpace(req.GameLabel) != "" {
		match.GameLabel = strings.TrimSpace(req.GameLabel)
	}
	if strings.TrimSpace(req.Opponent) != "" {
		match.Opponent = strings.TrimSpace(req.Opponent)
	}
	if strings.TrimSpace(req.MatchDate) != "" {
		match.MatchDate = strings.TrimSpace(req.MatchDate)
	}
	if req.MatchTime != nil {
		match.MatchTime = req.MatchTime
	}
	if req.Pitch != "" {
		match.Pitch = strings.TrimSpace(req.Pitch)
	}
	if req.AlabScore != nil {
		match.AlabScore = req.AlabScore
	}
	if req.OpponentScore != nil {
		match.OpponentScore = req.OpponentScore
	}
	if req.StatusID != nil {
		match.StatusID = req.StatusID
	} else if strings.TrimSpace(req.Status) != "" {
		match.StatusID = getRefSeqnumByName(req.Status)
	}
	if req.Remarks != "" {
		match.Remarks = strings.TrimSpace(req.Remarks)
	}

	db.Save(&match)
	return c.JSON(fiber.Map{
		"message": "Match updated",
		"data":    formatMatch(match, getRefMap()),
	})
}

func DeleteTournamentMatch(c *fiber.Ctx) error {
	matchID, err := strconv.Atoi(c.Params("match_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid match_id"})
	}

	res := database.Get().Where("match_id = ?", matchID).Delete(&models.TournamentMatch{})
	if res.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Match not found"})
	}

	return c.JSON(fiber.Map{
		"message":  "Match deleted",
		"match_id": matchID,
	})
}

func GetTournamentRoster(c *fiber.Ctx) error {
	tournamentID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid tournament_id"})
	}

	var rosters []models.TournamentRoster
	database.Get().
		Preload("Player").
		Where("tournament_id = ?", tournamentID).
		Order("roster_id ASC").
		Find(&rosters)

	filterCat := strings.TrimSpace(c.Query("age_category"))
	var filterCatID int
	if filterCat != "" {
		if seq := getRefSeqnumByName(filterCat); seq != nil {
			filterCatID = *seq
		}
	}

	refMap := getRefMap()
	result := make([]fiber.Map, 0, len(rosters))
	for _, r := range rosters {
		if filterCatID > 0 {
			if r.Player == nil || r.Player.AgeCategoryID == nil || *r.Player.AgeCategoryID != filterCatID {
				continue
			}
		}
		result = append(result, formatRoster(r, refMap))
	}

	return c.JSON(result)
}

func AddTournamentRoster(c *fiber.Ctx) error {
	tournamentID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid tournament_id"})
	}

	var req struct {
		PlayerID      int      `json:"player_id"`
		PlayerIDs     []int    `json:"player_ids"`
		PlayerUserID  int      `json:"player_user_id"`
		PlayerUserIDs []int    `json:"player_user_ids"`
		PlayerNames   []string `json:"player_names"`
		Remarks       string   `json:"remarks"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	seen := make(map[int]bool)
	playerIDs := make([]int, 0)
	addPlayer := func(id int) {
		if id > 0 && !seen[id] {
			seen[id] = true
			playerIDs = append(playerIDs, id)
		}
	}

	addPlayer(req.PlayerID)
	addPlayer(req.PlayerUserID)
	for _, id := range req.PlayerIDs {
		addPlayer(id)
	}
	for _, id := range req.PlayerUserIDs {
		addPlayer(id)
	}
	for _, name := range req.PlayerNames {
		if pid, err := findPlayerIDByName(name); err == nil {
			addPlayer(pid)
		}
	}

	if len(playerIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "At least one player is required"})
	}

	db := database.Get()
	added := 0
	for _, pid := range playerIDs {
		var existing models.TournamentRoster
		if err := db.Where("tournament_id = ? AND player_id = ?", tournamentID, pid).First(&existing).Error; err == nil {
			continue
		}
		roster := models.TournamentRoster{
			TournamentID:  tournamentID,
			PlayerID:      pid,
			ParentConsent: false,
			Remarks:       strings.TrimSpace(req.Remarks),
		}
		if err := db.Create(&roster).Error; err == nil {
			added++
		}
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Player(s) added to tournament roster",
		"added":   added,
	})
}

func UpdateParentConsent(c *fiber.Ctx) error {
	db := database.Get()
	var roster models.TournamentRoster
	if err := db.First(&roster, c.Params("roster_id")).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Roster entry not found"})
	}

	if !canAccessPlayer(c, roster.PlayerID) {
		return c.Status(403).JSON(fiber.Map{"error": "Forbidden: you can only give consent for your own child"})
	}

	var req struct {
		ParentConsent bool   `json:"parent_consent"`
		Remarks       string `json:"remarks"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	roster.ParentConsent = req.ParentConsent
	if req.Remarks != "" {
		roster.Remarks = strings.TrimSpace(req.Remarks)
	}
	db.Save(&roster)

	return c.JSON(fiber.Map{
		"message": "Parent consent updated",
		"data":    roster,
	})
}

func RemoveTournamentRoster(c *fiber.Ctx) error {
	rosterID, err := strconv.Atoi(c.Params("roster_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid roster_id"})
	}

	res := database.Get().Where("roster_id = ?", rosterID).Delete(&models.TournamentRoster{})
	if res.Error != nil {
		return c.Status(500).JSON(fiber.Map{"error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Roster entry not found"})
	}

	return c.JSON(fiber.Map{
		"message":   "Player removed from tournament roster",
		"roster_id": rosterID,
	})
}
