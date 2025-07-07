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
	EndDate        string   `json:"end_date"`
	Options        []string `json:"options"`
	CreatorAddress string   `json:"creator_address"` // Новый поле: адрес создателя
	VotesCount     int      `json:"votes_count"`     // Количество проголосовавших (для простоты, общее число)
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
	EndDate        string `json:"end_date"`
	IsPrivate      bool   `json:"is_private"`
	CreatorAddress string `json:"creator_address"`
	VotesCount     int    `json:"votes_count"`         // Общее количество проголосовавших в этом голосовании
	UserVote       *int   `json:"user_vote,omitempty"` // Индекс варианта, за который проголосовал пользователь (null если не голосовал)
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
	router.Get("/profile", func(w http.ResponseWriter, r *http.Request) { // Маршрут для профиля
		http.ServeFile(w, r, "./static/profile.html")
	})
	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	router.Post("/voting", CreateVoting)
	router.Get("/voting/{id}", GetVotingByID)
	router.Get("/voting", GetAllVotings)
	router.Post("/user-data", GetUserData) // Маршрут для получения данных пользователя

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

	newVoting.ID = strconv.Itoa(len(votings) + 1) // simple ID
	newVoting.VotesCount = 0                      // Initialize votes count
	votings[newVoting.ID] = newVoting

	// Update user activity for the creator
	// Это место не нужно менять, так как оно корректно добавляет ID в CreatedVotings
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
	showAll := r.URL.Query().Get("type") == "all"

	mu.Lock()
	for _, v := range votings {
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
	// Получаем или инициализируем активность пользователя
	activity, exists := userActivities[userAddressLower]
	if !exists {
		activity = UserActivity{
			CreatedVotings:      []string{},
			ParticipatedVotings: make(map[string]int),
		}
		// Важно: если активности не было, то и созданных/участвующих голосований тоже нет.
		// Но мы все равно хотим пройти по ВСЕМ голосованиям, чтобы найти созданные.
	}

	var userVotings []UserVotingDetail
	seenVotingIDs := make(map[string]bool) // Для отслеживания уже добавленных голосований

	// Перебираем ВСЕ голосования, чтобы найти созданные текущим пользователем
	for _, voting := range votings {
		if strings.ToLower(voting.CreatorAddress) == userAddressLower {
			userVotings = append(userVotings, UserVotingDetail{
				ID:             voting.ID,
				Title:          voting.Title,
				EndDate:        voting.EndDate,
				IsPrivate:      voting.IsPrivate,
				CreatorAddress: voting.CreatorAddress,
				VotesCount:     voting.VotesCount,
				UserVote:       nil, // Создатель обычно не голосует за своё
			})
			seenVotingIDs[voting.ID] = true
		}
	}

	// Добавляем голосования, в которых пользователь участвовал, если они еще не были добавлены
	for votingID, userVoteIndex := range activity.ParticipatedVotings {
		if !seenVotingIDs[votingID] { // Проверяем, не было ли уже добавлено это голосование (например, если пользователь создал и проголосовал)
			voting, ok := votings[votingID]
			if ok {
				voteIndex := userVoteIndex // Need a pointer to int for JSON omitempty
				userVotings = append(userVotings, UserVotingDetail{
					ID:             voting.ID,
					Title:          voting.Title,
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
	mu.Unlock() // Разблокируем мьютекс после всех операций с map votings и userActivities

	// Обновляем CreatedVotingsCount на основе фактически найденных созданных голосований
	createdCount := 0
	for _, voting := range userVotings {
		if strings.ToLower(voting.CreatorAddress) == userAddressLower {
			createdCount++
		}
	}

	response := UserDataResponse{
		WalletAddress:            requestBody.UserAddress,
		CreatedVotingsCount:      createdCount, // Обновлено
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

// For testing purposes, add some dummy data
func addDummyData() {
	mu.Lock()
	defer mu.Unlock()

	// Ensure we start with fresh data for each run in development
	votings = make(map[string]Voting)
	userActivities = make(map[string]UserActivity)

	// Dummy user addresses (lowercase for consistency)
	user1 := "0x1234567890abcdef1234567890abcdef12345678" // User A
	user2 := "0x9876543210fedcba9876543210fedcba98765432" // User B

	// Create some dummy votings
	voting1 := Voting{
		ID: "1", Title: "Выбор цвета логотипа", Description: "Голосуем за основной цвет нашего нового логотипа.", IsPrivate: false,
		MinVotes: 1, EndDate: time.Now().Add(24 * time.Hour).Format(time.RFC3339), Options: []string{"Синий", "Зеленый", "Красный"},
		CreatorAddress: user1, VotesCount: 5,
	}
	voting2 := Voting{
		ID: "2", Title: "Приватное голосование команды A", Description: "Решение по внутреннему проекту.", IsPrivate: true,
		MinVotes: 3, EndDate: time.Now().Add(48 * time.Hour).Format(time.RFC3339), Options: []string{"Да", "Нет"},
		CreatorAddress: user1, VotesCount: 2,
	}
	voting3 := Voting{
		ID: "3", Title: "Когда провести митинг?", Description: "Выбираем удобное время для еженедельного митинга.", IsPrivate: false,
		MinVotes: 1, EndDate: time.Now().Add(-1 * time.Hour).Format(time.RFC3339), Options: []string{"Понедельник", "Среда", "Пятница"},
		CreatorAddress: user2, VotesCount: 8, // Finished voting
	}
	voting4 := Voting{
		ID: "4", Title: "Будущий функционал", Description: "Какую функцию добавить следующей?", IsPrivate: false,
		MinVotes: 1, EndDate: time.Now().Add(72 * time.Hour).Format(time.RFC3339), Options: []string{"Чат", "Опросы", "Форум"},
		CreatorAddress: user2, VotesCount: 0,
	}
	voting5 := Voting{ // Добавим ещё одно голосование от user1
		ID: "5", Title: "Идея для следующего хакатона", Description: "На чем сфокусируемся?", IsPrivate: false,
		MinVotes: 1, EndDate: time.Now().Add(96 * time.Hour).Format(time.RFC3339), Options: []string{"AI", "Web3", "IoT"},
		CreatorAddress: user1, VotesCount: 1,
	}

	votings[voting1.ID] = voting1
	votings[voting2.ID] = voting2
	votings[voting3.ID] = voting3
	votings[voting4.ID] = voting4
	votings[voting5.ID] = voting5

	// Populate user activities (Ensure creator addresses are lowercase)
	userActivities[strings.ToLower(user1)] = UserActivity{
		CreatedVotings:      []string{"1", "2", "5"}, // Обновлено
		ParticipatedVotings: map[string]int{"3": 0},  // User1 voted for option 0 in voting 3
	}

	userActivities[strings.ToLower(user2)] = UserActivity{
		CreatedVotings:      []string{"3", "4"},
		ParticipatedVotings: map[string]int{"1": 1, "2": 0}, // User2 voted for option 1 in voting 1, and option 0 in voting 2
	}
}
