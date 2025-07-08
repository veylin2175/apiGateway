package main

import (
	"apiGateway/internal/config"
	"apiGateway/internal/http-server/middleware/mwlogger"
	"apiGateway/internal/lib/logger/handlers/slogpretty"
	"apiGateway/internal/lib/logger/sl"
	"context"
	"encoding/json"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	votings        = make(map[string]VoteSession)
	userActivities = make(map[string]UserActivity)
	mu             sync.Mutex
)

// UserActivity хранит информацию о пользовательской активности
type UserActivity struct {
	CreatedVotings      []string       `json:"created_votings"`
	ParticipatedVotings map[string]int `json:"participated_votings"`
}

type Choice struct {
	Title      string `json:"title"`
	CountVotes int64  `json:"countVotes"`
}

type Voter struct {
	Address string `json:"address"`
	IsVoted bool   `json:"is_voted"`
	Choice  int    `json:"choice_index"` // Изменено: храним индекс выбора
	CanVote bool   `json:"can_vote"`     // Это поле может быть вычислено, но для контракта оставим
}

type VoteSession struct {
	ID              string           `json:"voting_id"`
	CreatorAddr     string           `json:"creator_address"` // JSON-тег остался creator_address
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	StartTime       string           `json:"start_date"`  // JSON-тег остался start_date
	EndTime         string           `json:"end_date"`    // JSON-тег остался end_date
	MinNumberVotes  int64            `json:"min_votes"`   // JSON-тег остался min_votes
	TempNumberVotes int64            `json:"votes_count"` // JSON-тег остался votes_count
	IsPrivate       bool             `json:"is_private"`
	Choices         []Choice         `json:"options"` // JSON-тег остался options
	Voters          map[string]Voter `json:"voters"`
	Winner          []string         `json:"winner"`
	Status          string           `json:"status"` // "Upcoming", "Active", "Finished", "Rejected"
}

// UserDataResponse структура ответа для получения данных пользователя
type UserDataResponse struct {
	WalletAddress            string             `json:"wallet_address"`
	CreatedVotingsCount      int                `json:"created_votings_count"`
	ParticipatedVotingsCount int                `json:"participated_votings_count"`
	Votings                  []UserVotingDetail `json:"votings"`
}

// UserVotingDetail представляет одно голосование для профиля пользователя
type UserVotingDetail struct {
	ID             string `json:"voting_id"`
	Title          string `json:"title"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	IsPrivate      bool   `json:"is_private"`
	CreatorAddress string `json:"creator_address"`
	VotesCount     int64  `json:"votes_count"`
	UserVote       *int   `json:"user_vote,omitempty"`
	Status         string `json:"status"` // Добавлено поле Status для UserVotingDetail
}

// VoteRequest структура для приема запроса на голосование
type VoteRequest struct {
	VotingID            string `json:"voting_id"`
	UserAddress         string `json:"user_address"`
	SelectedOptionIndex int    `json:"selected_option_index"`
}

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("Starting voting service", slog.String("env", cfg.Env))
	log.Debug("Debug messages are enabled")

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(mwlogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})
	router.Get("/profile", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/profile.html")
	})
	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	router.Post("/voting", CreateVoting)
	router.Get("/voting/{id}", GetVotingByID)
	router.Get("/voting", GetAllVotings)
	router.Post("/user-data", GetUserData)
	router.Post("/vote", SubmitVote)

	log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	// Initialize some dummy data for testing
	addDummyData()

	// Горутина для регулярного обновления статусов голосований
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second) // Проверяем каждые 30 секунд
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				UpdateAllVotingStatuses()
			case <-ctx.Done(): // Используем ctx для graceful shutdown
				log.Info("Voting status update goroutine stopped")
				return
			}
		}
	}()

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server", sl.Err(err))
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

	sign := <-stop

	log.Info("application stopping", slog.String("signal", sign.String()))
	cancel()
	wg.Wait()

	log.Info("application stopped")
}

// setupLogger создает логгер с различными хендерами и уровнями логирования в зависимости от окружения
func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}

// setupPrettySlog создает логгер с удобным выводом данных для локала
func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	h := opts.NewPrettyHandler(os.Stdout)

	return slog.New(h)
}

func CreateVoting(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		Title          string   `json:"title"`
		Description    string   `json:"description"`
		IsPrivate      bool     `json:"is_private"`
		MinNumberVotes int64    `json:"min_votes"`
		StartTime      string   `json:"start_date"`
		EndTime        string   `json:"end_date"`
		Choices        []string `json:"options"` // Принимаем массив строк
		CreatorAddress string   `json:"creator_address"`
	}
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// Валидация дат
	startDate, err := time.Parse(time.RFC3339, requestData.StartTime)
	if err != nil {
		http.Error(w, "Invalid start date format", http.StatusBadRequest)
		slog.Error("CreateVoting: Invalid start date format", sl.Err(err))
		return
	}
	endDate, err := time.Parse(time.RFC3339, requestData.EndTime)
	if err != nil {
		http.Error(w, "Invalid end date format", http.StatusBadRequest)
		slog.Error("CreateVoting: Invalid end date format", sl.Err(err))
		return
	}
	if startDate.After(endDate) {
		http.Error(w, "Start date cannot be after end date", http.StatusBadRequest)
		slog.Error("CreateVoting: Start date after end date")
		return
	}

	// Преобразование []string в []Choice
	choices := make([]Choice, len(requestData.Choices))
	for i, title := range requestData.Choices {
		choices[i] = Choice{Title: title, CountVotes: 0}
	}

	newVoting := VoteSession{
		ID:              strconv.Itoa(len(votings) + 1),
		CreatorAddr:     requestData.CreatorAddress,
		Title:           requestData.Title,
		Description:     requestData.Description,
		StartTime:       requestData.StartTime,
		EndTime:         requestData.EndTime,
		MinNumberVotes:  requestData.MinNumberVotes,
		TempNumberVotes: 0,
		IsPrivate:       requestData.IsPrivate,
		Choices:         choices,                // Используем преобразованный срез
		Voters:          make(map[string]Voter), // Инициализируем пустую мапу
		Winner:          []string{},
		Status:          "Upcoming", // Изначальный статус
	}

	votings[newVoting.ID] = newVoting

	creatorAddressLower := strings.ToLower(newVoting.CreatorAddr)
	activity := userActivities[creatorAddressLower]
	activity.CreatedVotings = append(activity.CreatedVotings, newVoting.ID)
	userActivities[creatorAddressLower] = activity

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]string{
		"voting_id": newVoting.ID,
	})
	if err != nil {
		slog.Error("Failed to encode response for CreateVoting", sl.Err(err))
		return
	}
}

func GetVotingByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	mu.Lock()
	voting, ok := votings[id]
	mu.Unlock()

	if !ok {
		http.Error(w, "VoteSession not found", http.StatusNotFound)
		return
	}

	// Обновляем статус голосования перед отправкой
	// Это гарантирует, что при запросе деталей всегда будет актуальный статус
	UpdateVotingStatusAndWinner(id)

	mu.Lock()            // Перезаблокируем для чтения обновленных данных
	voting = votings[id] // Получаем обновленную версию
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(voting)
	if err != nil {
		slog.Error("Failed to encode response for GetVotingByID", sl.Err(err))
		return
	}
}

func GetAllVotings(w http.ResponseWriter, r *http.Request) {
	var filteredVotings []VoteSession
	showAll := r.URL.Query().Get("type") == "all" // Используется для отображения приватных голосований

	mu.Lock()
	for _, v := range votings {
		// Обновляем статус голосования перед добавлением в список
		UpdateVotingStatusAndWinner(v.ID) // Обновляем в цикле
		updatedVoting := votings[v.ID]    // Получаем обновленную версию

		// Фильтруем приватные голосования, если showAll не установлен
		if showAll || !updatedVoting.IsPrivate {
			filteredVotings = append(filteredVotings, updatedVoting)
		}
	}
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(filteredVotings)
	if err != nil {
		slog.Error("Failed to encode response for GetAllVotings", sl.Err(err))
		return
	}
}

func GetUserData(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		UserAddress string `json:"user_address"`
	}
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	userAddressLower := strings.ToLower(requestBody.UserAddress)

	mu.Lock()
	activity, exists := userActivities[userAddressLower]
	if !exists {
		activity = UserActivity{
			CreatedVotings:      []string{},
			ParticipatedVotings: make(map[string]int),
		}
	}

	var userVotings []UserVotingDetail
	seenVotingIDs := make(map[string]bool)

	for _, voting := range votings {
		// Обновляем статус голосования перед добавлением в список
		UpdateVotingStatusAndWinner(voting.ID) // Обновляем в цикле
		updatedVoting := votings[voting.ID]    // Получаем обновленную версию

		// Если пользователь является создателем голосования
		if strings.ToLower(updatedVoting.CreatorAddr) == userAddressLower {
			userVoteIndex, ok := activity.ParticipatedVotings[updatedVoting.ID]
			var userVotePtr *int
			if ok {
				userVotePtr = &userVoteIndex
			}

			userVotings = append(userVotings, UserVotingDetail{
				ID:             updatedVoting.ID,
				Title:          updatedVoting.Title,
				StartDate:      updatedVoting.StartTime,
				EndDate:        updatedVoting.EndTime,
				IsPrivate:      updatedVoting.IsPrivate,
				CreatorAddress: updatedVoting.CreatorAddr,
				VotesCount:     updatedVoting.TempNumberVotes,
				UserVote:       userVotePtr,
				Status:         updatedVoting.Status, // Добавлено
			})
			seenVotingIDs[updatedVoting.ID] = true
		}
	}

	for votingID, userVoteIndex := range activity.ParticipatedVotings {
		if !seenVotingIDs[votingID] {
			voting, ok := votings[votingID]
			if ok {
				UpdateVotingStatusAndWinner(voting.ID) // Обновляем статус перед использованием
				updatedVoting := votings[voting.ID]

				voteIndex := userVoteIndex
				userVotings = append(userVotings, UserVotingDetail{
					ID:             updatedVoting.ID,
					Title:          updatedVoting.Title,
					StartDate:      updatedVoting.StartTime,
					EndDate:        updatedVoting.EndTime,
					IsPrivate:      updatedVoting.IsPrivate,
					CreatorAddress: updatedVoting.CreatorAddr,
					VotesCount:     updatedVoting.TempNumberVotes,
					UserVote:       &voteIndex,
					Status:         updatedVoting.Status, // Добавлено
				})
				seenVotingIDs[updatedVoting.ID] = true
			}
		}
	}
	mu.Unlock()

	createdCount := 0
	for _, uv := range userVotings {
		if strings.ToLower(uv.CreatorAddress) == userAddressLower {
			createdCount++
		}
	}

	response := UserDataResponse{
		WalletAddress:            requestBody.UserAddress,
		CreatedVotingsCount:      createdCount,
		ParticipatedVotingsCount: len(activity.ParticipatedVotings),
		Votings:                  userVotings,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		slog.Error("Failed to encode response for GetUserData", sl.Err(err))
		return
	}
}

// SubmitVote обрабатывает запрос на голосование
func SubmitVote(w http.ResponseWriter, r *http.Request) {
	var req VoteRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		slog.Error("SubmitVote: Invalid request payload", sl.Err(err))
		return
	}

	mu.Lock()
	defer mu.Unlock()

	voting, ok := votings[req.VotingID]
	if !ok {
		http.Error(w, "VoteSession not found", http.StatusNotFound)
		slog.Error("SubmitVote: VoteSession not found", slog.String("voting_id", req.VotingID))
		return
	}

	// Проверяем, началось ли голосование
	startDate, err := time.Parse(time.RFC3339, voting.StartTime)
	if err != nil {
		http.Error(w, "Invalid start date format for voting", http.StatusInternalServerError)
		slog.Error("SubmitVote: Invalid start date format", sl.Err(err), slog.String("voting_id", req.VotingID))
		return
	}
	if time.Now().Before(startDate) {
		http.Error(w, "VoteSession has not started yet", http.StatusForbidden)
		slog.Warn("SubmitVote: VoteSession has not started", slog.String("voting_id", req.VotingID))
		return
	}

	// Проверяем, закончилось ли голосование
	endDate, err := time.Parse(time.RFC3339, voting.EndTime)
	if err != nil {
		http.Error(w, "Invalid end date format for voting", http.StatusInternalServerError)
		slog.Error("SubmitVote: Invalid end date format", sl.Err(err), slog.String("voting_id", req.VotingID))
		return
	}
	if time.Now().After(endDate) {
		http.Error(w, "VoteSession has already ended", http.StatusForbidden)
		slog.Warn("SubmitVote: VoteSession has ended", slog.String("voting_id", req.VotingID))
		return
	}

	// Проверяем валидность выбранной опции
	if req.SelectedOptionIndex < 0 || req.SelectedOptionIndex >= len(voting.Choices) {
		http.Error(w, "Invalid option selected", http.StatusBadRequest)
		slog.Warn("SubmitVote: Invalid option index", slog.Int("option_index", req.SelectedOptionIndex), slog.String("voting_id", req.VotingID))
		return
	}

	userAddressLower := strings.ToLower(req.UserAddress)

	// Проверяем, голосовал ли пользователь уже (через Voters)
	if voter, exists := voting.Voters[userAddressLower]; exists && voter.IsVoted {
		http.Error(w, "You have already voted in this poll", http.StatusConflict) // 409 Conflict
		slog.Warn("SubmitVote: User already voted via Voters map", slog.String("user_address", req.UserAddress), slog.String("voting_id", req.VotingID))
		return
	}

	// Проверяем, голосовал ли пользователь уже (через UserActivity - для обратной совместимости или если нужно учитывать централизованно)
	activity := userActivities[userAddressLower]
	if activity.ParticipatedVotings == nil {
		activity.ParticipatedVotings = make(map[string]int)
	}
	if _, alreadyVoted := activity.ParticipatedVotings[req.VotingID]; alreadyVoted {
		http.Error(w, "You have already voted in this poll", http.StatusConflict) // 409 Conflict
		slog.Warn("SubmitVote: User already voted via UserActivity map", slog.String("user_address", req.UserAddress), slog.String("voting_id", req.VotingID))
		return
	}

	// Регистрируем голос в UserActivity
	activity.ParticipatedVotings[req.VotingID] = req.SelectedOptionIndex
	userActivities[userAddressLower] = activity

	// Регистрируем голос в VoteSession.Voters
	voting.Voters[userAddressLower] = Voter{
		Address: req.UserAddress,
		IsVoted: true,
		Choice:  req.SelectedOptionIndex,
		CanVote: true, // Это поле здесь не играет роли, но сохраним для структуры
	}

	// Увеличиваем счетчик голосов для выбранной опции
	voting.Choices[req.SelectedOptionIndex].CountVotes++

	// Увеличиваем общий счетчик голосов для голосования
	voting.TempNumberVotes++
	votings[req.VotingID] = voting

	// Обновляем статус голосования сразу после голосования (опционально, но полезно)
	UpdateVotingStatusAndWinner(req.VotingID)

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(map[string]string{"message": "Vote successfully recorded"})
	if err != nil {
		return
	}
	slog.Info("Vote recorded", slog.String("voting_id", req.VotingID), slog.String("user_address", req.UserAddress), slog.Int("option_index", req.SelectedOptionIndex))
}

// UpdateVotingStatusAndWinner обновляет статус голосования и определяет победителя
// Должен вызываться под мьютексом
func UpdateVotingStatusAndWinner(votingID string) {
	voting, ok := votings[votingID]
	if !ok {
		return // Голосование не найдено
	}

	now := time.Now()
	startDate, _ := time.Parse(time.RFC3339, voting.StartTime)
	endDate, _ := time.Parse(time.RFC3339, voting.EndTime)

	if now.Before(startDate) {
		voting.Status = "Upcoming"
	} else if now.After(endDate) {
		// Голосование завершено
		if voting.TempNumberVotes < voting.MinNumberVotes {
			voting.Status = "Rejected" // Отклонено, если не набрано мин.голосов
			voting.Winner = []string{}
		} else {
			voting.Status = "Finished" // Закончено, если набрано

			maxVotes := int64(-1)
			var winners []string

			for _, choice := range voting.Choices {
				if choice.CountVotes > maxVotes {
					maxVotes = choice.CountVotes
					winners = []string{choice.Title}
				} else if choice.CountVotes == maxVotes && maxVotes != -1 {
					winners = append(winners, choice.Title)
				}
			}
			voting.Winner = winners
		}
	} else {
		voting.Status = "Active" // Активное
	}

	votings[votingID] = voting // Сохраняем обновленное голосование
}

// UpdateAllVotingStatuses проходит по всем голосованиям и обновляет их статус
func UpdateAllVotingStatuses() {
	mu.Lock()
	defer mu.Unlock()
	slog.Debug("Updating all voting statuses...")
	for id := range votings {
		UpdateVotingStatusAndWinner(id)
	}
	slog.Debug("All voting statuses updated.")
}

// For testing purposes, add some dummy data
func addDummyData() {
	votings = make(map[string]VoteSession)
	userActivities = make(map[string]UserActivity)

	user1 := "0x1234567890abcdef1234567890abcdef12345678"
	user2 := "0x9876543210fedcba9876543210fedcba98765432"
	user3 := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	now := time.Now()
	futureStartDate := now.Add(48 * time.Hour)
	pastEndDate := now.Add(-1 * time.Hour)
	activeStartDate := now.Add(-24 * time.Hour)

	// Helper to create Choice slice
	createChoices := func(titles []string) []Choice {
		c := make([]Choice, len(titles))
		for i, t := range titles {
			c[i] = Choice{Title: t, CountVotes: 0}
		}
		return c
	}

	// Helper to create Voters map
	createVoters := func(votes map[string]int, votingChoices []Choice) map[string]Voter {
		votersMap := make(map[string]Voter)
		for addr, choiceIdx := range votes {
			votersMap[strings.ToLower(addr)] = Voter{
				Address: addr,
				IsVoted: true,
				Choice:  choiceIdx,
				CanVote: true,
			}
			if choiceIdx >= 0 && choiceIdx < len(votingChoices) {
				// Increment count here for dummy data
				votingChoices[choiceIdx].CountVotes++
			}
		}
		return votersMap
	}

	// Voting 1: Active, with votes
	choices1 := createChoices([]string{"Синий", "Зеленый", "Красный"})
	voters1 := createVoters(map[string]int{
		user1: 0,
		user2: 1,
		user3: 0,
		"0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb": 2,
		"0xcccccccccccccccccccccccccccccccccccccccc": 0,
	}, choices1)
	voting1 := VoteSession{
		ID: "1", Title: "Выбор цвета логотипа", Description: "Голосуем за основной цвет нашего нового логотипа.", IsPrivate: false,
		MinNumberVotes: 1, StartTime: activeStartDate.Format(time.RFC3339), EndTime: now.Add(24 * time.Hour).Format(time.RFC3339), Choices: choices1,
		CreatorAddr: user1, TempNumberVotes: int64(len(voters1)), Voters: voters1, Winner: []string{}, Status: "Active",
	}

	// Voting 2: Private, Active
	choices2 := createChoices([]string{"Да", "Нет"})
	voters2 := createVoters(map[string]int{
		user1: 0,
		user2: 1,
	}, choices2)
	voting2 := VoteSession{
		ID: "2", Title: "Приватное голосование команды A", Description: "Решение по внутреннему проекту.", IsPrivate: true,
		MinNumberVotes: 3, StartTime: activeStartDate.Format(time.RFC3339), EndTime: now.Add(48 * time.Hour).Format(time.RFC3339), Choices: choices2,
		CreatorAddr: user1, TempNumberVotes: int64(len(voters2)), Voters: voters2, Winner: []string{}, Status: "Active",
	}

	// Voting 3: Finished, has winner
	choices3 := createChoices([]string{"Понедельник", "Среда", "Пятница"})
	voters3 := createVoters(map[string]int{
		user1: 0,
		user2: 1,
		user3: 1,
		"0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee": 1,
		"0xffffffffffffffffffffffffffffffffffffffff": 2,
		"0xdddddddddddddddddddddddddddddddddddddddd": 0,
		"0xgggggggggggggggggggggggggggggggggggggggg": 1,
		"0xhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh": 2,
	}, choices3)
	voting3 := VoteSession{
		ID: "3", Title: "Когда провести митинг?", Description: "Выбираем удобное время для еженедельного митинга.", IsPrivate: false,
		MinNumberVotes: 1, StartTime: now.Add(-72 * time.Hour).Format(time.RFC3339), EndTime: pastEndDate.Format(time.RFC3339), Choices: choices3,
		CreatorAddr: user2, TempNumberVotes: int64(len(voters3)), Voters: voters3, Winner: []string{}, Status: "Finished", // Status will be updated by goroutine
	}

	// Voting 4: Upcoming
	choices4 := createChoices([]string{"Чат", "Опросы", "Форум"})
	voting4 := VoteSession{
		ID: "4", Title: "Будущий функционал", Description: "Какую функцию добавить следующей?", IsPrivate: false,
		MinNumberVotes: 1, StartTime: futureStartDate.Format(time.RFC3339), EndTime: futureStartDate.Add(72 * time.Hour).Format(time.RFC3339), Choices: choices4,
		CreatorAddr: user2, TempNumberVotes: 0, Voters: make(map[string]Voter), Winner: []string{}, Status: "Upcoming",
	}

	// Voting 5: Active, low votes (will be Rejected after end)
	choices5 := createChoices([]string{"AI", "Web3", "IoT"})
	voters5 := createVoters(map[string]int{
		user1: 0,
	}, choices5)
	voting5 := VoteSession{
		ID: "5", Title: "Идея для следующего хакатона", Description: "На чем сфокусируемся?", IsPrivate: false,
		MinNumberVotes: 5, StartTime: activeStartDate.Format(time.RFC3339), EndTime: now.Add(96 * time.Hour).Format(time.RFC3339), Choices: choices5,
		CreatorAddr: user1, TempNumberVotes: int64(len(voters5)), Voters: voters5, Winner: []string{}, Status: "Active",
	}

	// Voting 6: Active, multiple winners possible
	choices6 := createChoices([]string{"Синий", "Зеленый", "Желтый", "Фиолетовый"})
	voters6 := createVoters(map[string]int{
		"0xaaa": 0, "0xbbb": 0,
		"0xccc": 1, "0xddd": 1,
		"0xeee": 2,
	}, choices6)
	voting6 := VoteSession{
		ID: "6", Title: "Любимый цвет", Description: "Какой ваш любимый цвет?", IsPrivate: false,
		MinNumberVotes: 1, StartTime: activeStartDate.Format(time.RFC3339), EndTime: now.Add(120 * time.Hour).Format(time.RFC3339), Choices: choices6,
		CreatorAddr: user1, TempNumberVotes: int64(len(voters6)), Voters: voters6, Winner: []string{}, Status: "Active",
	}

	votings[voting1.ID] = voting1
	votings[voting2.ID] = voting2
	votings[voting3.ID] = voting3
	votings[voting4.ID] = voting4
	votings[voting5.ID] = voting5
	votings[voting6.ID] = voting6

	// Update initial statuses and winners for dummy data
	UpdateAllVotingStatuses()

	userActivities[strings.ToLower(user1)] = UserActivity{
		CreatedVotings:      []string{"1", "2", "5", "6"},
		ParticipatedVotings: map[string]int{"3": 0, "1": 0}, // User1 voted in 3 and 1
	}

	userActivities[strings.ToLower(user2)] = UserActivity{
		CreatedVotings:      []string{"3", "4"},
		ParticipatedVotings: map[string]int{"1": 1, "2": 0}, // User2 voted in 1 and 2
	}

	userActivities[strings.ToLower(user3)] = UserActivity{
		CreatedVotings:      []string{},
		ParticipatedVotings: map[string]int{"1": 0, "3": 1}, // User3 voted in 1 and 3
	}
}
