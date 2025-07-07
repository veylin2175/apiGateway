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

// UserActivity хранит информацию о пользовательской активности
type UserActivity struct {
	CreatedVotings      []string       `json:"created_votings"`      // IDs созданных голосований
	ParticipatedVotings map[string]int `json:"participated_votings"` // ID голосования -> Индекс выбранного варианта (0-based)
}

// Global data stores (temporary, using in-memory maps)
var (
	votings = make(map[string]Voting)
	// userActivities key: wallet address (lowercase), value: UserActivity
	userActivities = make(map[string]UserActivity)
	mu             sync.Mutex // Mutex for concurrent map access
)

type Voting struct {
	ID             string   `json:"voting_id"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	IsPrivate      bool     `json:"is_private"`
	MinVotes       int      `json:"min_votes"`
	StartDate      string   `json:"start_date"` // <--- НОВОЕ ПОЛЕ
	EndDate        string   `json:"end_date"`
	Options        []string `json:"options"`
	CreatorAddress string   `json:"creator_address"`
	VotesCount     int      `json:"votes_count"`
}

// UserDataResponse структура ответа для получения данных пользователя
type UserDataResponse struct {
	WalletAddress            string             `json:"wallet_address"`
	CreatedVotingsCount      int                `json:"created_votings_count"`
	ParticipatedVotingsCount int                `json:"participated_votings_count"`
	Votings                  []UserVotingDetail `json:"votings"` // Детали голосований, связанных с пользователем
}

// UserVotingDetail представляет одно голосование для профиля пользователя
type UserVotingDetail struct {
	ID             string `json:"voting_id"`
	Title          string `json:"title"`
	StartDate      string `json:"start_date"` // <--- НОВОЕ ПОЛЕ
	EndDate        string `json:"end_date"`
	IsPrivate      bool   `json:"is_private"`
	CreatorAddress string `json:"creator_address"`
	VotesCount     int    `json:"votes_count"`
	UserVote       *int   `json:"user_vote,omitempty"`
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

	_, cancel := context.WithCancel(context.Background())
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
	var newVoting Voting
	err := json.NewDecoder(r.Body).Decode(&newVoting)
	if err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// Валидация дат
	startDate, err := time.Parse(time.RFC3339, newVoting.StartDate)
	if err != nil {
		http.Error(w, "Invalid start date format", http.StatusBadRequest)
		slog.Error("CreateVoting: Invalid start date format", sl.Err(err))
		return
	}
	endDate, err := time.Parse(time.RFC3339, newVoting.EndDate)
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

	newVoting.ID = strconv.Itoa(len(votings) + 1)
	newVoting.VotesCount = 0
	votings[newVoting.ID] = newVoting

	creatorAddressLower := strings.ToLower(newVoting.CreatorAddress)
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
		http.Error(w, "Voting not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(voting)
	if err != nil {
		slog.Error("Failed to encode response for GetVotingByID", sl.Err(err))
		return
	}
}

func GetAllVotings(w http.ResponseWriter, r *http.Request) {
	var filteredVotings []Voting
	showAll := r.URL.Query().Get("type") == "all" // Используется для отображения приватных голосований

	mu.Lock()
	for _, v := range votings {
		// Фильтруем приватные голосования, если showAll не установлен
		if showAll || !v.IsPrivate {
			filteredVotings = append(filteredVotings, v)
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
		// Если пользователь является создателем голосования
		if strings.ToLower(voting.CreatorAddress) == userAddressLower {
			userVoteIndex, ok := activity.ParticipatedVotings[voting.ID]
			var userVotePtr *int
			if ok {
				userVotePtr = &userVoteIndex
			}

			userVotings = append(userVotings, UserVotingDetail{
				ID:             voting.ID,
				Title:          voting.Title,
				StartDate:      voting.StartDate, // <--- Добавлено
				EndDate:        voting.EndDate,
				IsPrivate:      voting.IsPrivate,
				CreatorAddress: voting.CreatorAddress,
				VotesCount:     voting.VotesCount,
				UserVote:       userVotePtr,
			})
			seenVotingIDs[voting.ID] = true
		}
	}

	for votingID, userVoteIndex := range activity.ParticipatedVotings {
		if !seenVotingIDs[votingID] {
			voting, ok := votings[votingID]
			if ok {
				voteIndex := userVoteIndex
				userVotings = append(userVotings, UserVotingDetail{
					ID:             voting.ID,
					Title:          voting.Title,
					StartDate:      voting.StartDate, // <--- Добавлено
					EndDate:        voting.EndDate,
					IsPrivate:      voting.IsPrivate,
					CreatorAddress: voting.CreatorAddress,
					VotesCount:     voting.VotesCount,
					UserVote:       &voteIndex,
				})
				seenVotingIDs[voting.ID] = true
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
		http.Error(w, "Voting not found", http.StatusNotFound)
		slog.Error("SubmitVote: Voting not found", slog.String("voting_id", req.VotingID))
		return
	}

	// Проверяем, началось ли голосование
	startDate, err := time.Parse(time.RFC3339, voting.StartDate) // <--- Проверка даты начала
	if err != nil {
		http.Error(w, "Invalid start date format for voting", http.StatusInternalServerError)
		slog.Error("SubmitVote: Invalid start date format", sl.Err(err), slog.String("voting_id", req.VotingID))
		return
	}
	if time.Now().Before(startDate) { // <--- Проверка на будущую дату
		http.Error(w, "Voting has not started yet", http.StatusForbidden)
		slog.Warn("SubmitVote: Voting has not started", slog.String("voting_id", req.VotingID))
		return
	}

	// Проверяем, закончилось ли голосование
	endDate, err := time.Parse(time.RFC3339, voting.EndDate)
	if err != nil {
		http.Error(w, "Invalid end date format for voting", http.StatusInternalServerError)
		slog.Error("SubmitVote: Invalid end date format", sl.Err(err), slog.String("voting_id", req.VotingID))
		return
	}
	if time.Now().After(endDate) {
		http.Error(w, "Voting has already ended", http.StatusForbidden)
		slog.Warn("SubmitVote: Voting has ended", slog.String("voting_id", req.VotingID))
		return
	}

	// Проверяем валидность выбранной опции
	if req.SelectedOptionIndex < 0 || req.SelectedOptionIndex >= len(voting.Options) {
		http.Error(w, "Invalid option selected", http.StatusBadRequest)
		slog.Warn("SubmitVote: Invalid option index", slog.Int("option_index", req.SelectedOptionIndex), slog.String("voting_id", req.VotingID))
		return
	}

	userAddressLower := strings.ToLower(req.UserAddress)

	// Проверяем, голосовал ли пользователь уже
	activity := userActivities[userAddressLower]
	if activity.ParticipatedVotings == nil {
		activity.ParticipatedVotings = make(map[string]int)
	}
	if _, alreadyVoted := activity.ParticipatedVotings[req.VotingID]; alreadyVoted {
		http.Error(w, "You have already voted in this poll", http.StatusConflict) // 409 Conflict
		slog.Warn("SubmitVote: User already voted", slog.String("user_address", req.UserAddress), slog.String("voting_id", req.VotingID))
		return
	}

	// Регистрируем голос
	activity.ParticipatedVotings[req.VotingID] = req.SelectedOptionIndex
	userActivities[userAddressLower] = activity

	// Увеличиваем счетчик голосов для голосования
	voting.VotesCount++
	votings[req.VotingID] = voting

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Vote successfully recorded"})
	slog.Info("Vote recorded", slog.String("voting_id", req.VotingID), slog.String("user_address", req.UserAddress), slog.Int("option_index", req.SelectedOptionIndex))
}

// For testing purposes, add some dummy data
func addDummyData() {
	mu.Lock()
	defer mu.Unlock()

	votings = make(map[string]Voting)
	userActivities = make(map[string]UserActivity)

	user1 := "0x1234567890abcdef1234567890abcdef12345678"
	user2 := "0x9876543210fedcba9876543210fedcba98765432"
	user3 := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	now := time.Now()
	futureStartDate := now.Add(48 * time.Hour)  // Голосование начнется через 2 дня
	pastEndDate := now.Add(-1 * time.Hour)      // Голосование уже закончилось
	activeStartDate := now.Add(-24 * time.Hour) // Голосование началось вчера

	voting1 := Voting{
		ID: "1", Title: "Выбор цвета логотипа", Description: "Голосуем за основной цвет нашего нового логотипа.", IsPrivate: false,
		MinVotes: 1, StartDate: activeStartDate.Format(time.RFC3339), EndDate: now.Add(24 * time.Hour).Format(time.RFC3339), Options: []string{"Синий", "Зеленый", "Красный"},
		CreatorAddress: user1, VotesCount: 5,
	}
	voting2 := Voting{
		ID: "2", Title: "Приватное голосование команды A", Description: "Решение по внутреннему проекту.", IsPrivate: true,
		MinVotes: 3, StartDate: activeStartDate.Format(time.RFC3339), EndDate: now.Add(48 * time.Hour).Format(time.RFC3339), Options: []string{"Да", "Нет"},
		CreatorAddress: user1, VotesCount: 2,
	}
	voting3 := Voting{
		ID: "3", Title: "Когда провести митинг?", Description: "Выбираем удобное время для еженедельного митинга.", IsPrivate: false,
		MinVotes: 1, StartDate: now.Add(-72 * time.Hour).Format(time.RFC3339), EndDate: pastEndDate.Format(time.RFC3339), Options: []string{"Понедельник", "Среда", "Пятница"},
		CreatorAddress: user2, VotesCount: 8, // Finished voting
	}
	voting4 := Voting{
		ID: "4", Title: "Будущий функционал", Description: "Какую функцию добавить следующей?", IsPrivate: false,
		MinVotes: 1, StartDate: futureStartDate.Format(time.RFC3339), EndDate: futureStartDate.Add(72 * time.Hour).Format(time.RFC3339), Options: []string{"Чат", "Опросы", "Форум"},
		CreatorAddress: user2, VotesCount: 0, // Upcoming voting
	}
	voting5 := Voting{
		ID: "5", Title: "Идея для следующего хакатона", Description: "На чем сфокусируемся?", IsPrivate: false,
		MinVotes: 1, StartDate: activeStartDate.Format(time.RFC3339), EndDate: now.Add(96 * time.Hour).Format(time.RFC3339), Options: []string{"AI", "Web3", "IoT"},
		CreatorAddress: user1, VotesCount: 1,
	}
	voting6 := Voting{
		ID: "6", Title: "Любимый цвет", Description: "Какой ваш любимый цвет?", IsPrivate: false,
		MinVotes: 1, StartDate: activeStartDate.Format(time.RFC3339), EndDate: now.Add(120 * time.Hour).Format(time.RFC3339), Options: []string{"Синий", "Зеленый", "Желтый", "Фиолетовый"},
		CreatorAddress: user1, VotesCount: 0,
	}

	votings[voting1.ID] = voting1
	votings[voting2.ID] = voting2
	votings[voting3.ID] = voting3
	votings[voting4.ID] = voting4
	votings[voting5.ID] = voting5
	votings[voting6.ID] = voting6

	userActivities[strings.ToLower(user1)] = UserActivity{
		CreatedVotings:      []string{"1", "2", "5", "6"},
		ParticipatedVotings: map[string]int{"3": 0},
	}

	userActivities[strings.ToLower(user2)] = UserActivity{
		CreatedVotings:      []string{"3", "4"},
		ParticipatedVotings: map[string]int{"1": 1, "2": 0},
	}

	userActivities[strings.ToLower(user3)] = UserActivity{
		CreatedVotings:      []string{},
		ParticipatedVotings: make(map[string]int),
	}
}
