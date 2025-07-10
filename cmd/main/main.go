package main

import (
	"apiGateway/internal/config"
	"apiGateway/internal/dto"
	"apiGateway/internal/http-server/middleware/mwlogger"
	"apiGateway/internal/kafka/producer"
	"apiGateway/internal/lib/logger/handlers/slogpretty"
	"apiGateway/internal/lib/logger/sl"
	"apiGateway/internal/models"
	"context"
	"encoding/json"
	"fmt"
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
	log            *slog.Logger
	kafkaProducer  *producer.Producer
	votings        = make(map[string]models.VoteSession)
	userActivities = make(map[string]models.UserActivity)
	mu             sync.Mutex
	err            error
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log = setupLogger(cfg.Env)

	log.Info("Starting voting service", slog.String("env", cfg.Env))
	log.Debug("Debug messages are enabled")

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)

	//kafkaConsumer := consumer.NewConsumer(cfg.Kafka, "voting-create")

	//go kafkaConsumer.Run(ctx, wg)

	kafkaProducer, err = producer.NewProducer(cfg.Kafka, log)
	if err != nil {
		log.Error("failed to create kafka producer", err)
	}
	defer kafkaProducer.Close()

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

	//router.Post("/voting", CreateVoting)
	router.Get("/voting/{id}", GetVotingByID)
	router.Get("/voting", GetAllVotings)
	router.Post("/user-data", GetUserData)
	router.Post("/vote", SubmitVote)
	router.Post("/connect-wallet", ConnectWalletHandler)
	router.Post("/voting", CreateVotingHandler)

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

	/*votingClient, err := client.NewVotingClient(cfg, log)
	if err != nil {
		log.Error("Failed to create voting client: %v", err)
	}
	fmt.Println("\n--- Testing AddVoteSession ---")
	testTitle := "Тестовое голосование из Go"
	testDesc := "Это голосование создано из Go-бэкенда."
	testStartTime := big.NewInt(time.Now().Unix())
	testEndTime := big.NewInt(time.Now().Add(24 * time.Hour).Unix()) // Через 24 часа
	testMinVotes := big.NewInt(1)
	testIsPrivate := false
	testVoters := []client.Voter{ // Пример одного голосующего
		{Addr: votingClient.FromAddress, HasVoted: false, Choice: "", CanVote: client.VoteAccessHasAccess},
		{Addr: common.HexToAddress("0x70997970C51812dc3A0108C7934CDCc3FbF7b2cc"), HasVoted: false, Choice: "", CanVote: client.VoteAccessHasAccess}, // Пример другого адреса из Hardhat
	}
	testChoices := []string{"Вариант A", "Вариант B", "Вариант C"}

	txHash, err := votingClient.AddVoteSession(
		testTitle,
		testDesc,
		testStartTime,
		testEndTime,
		testMinVotes,
		testIsPrivate,
		testVoters,
		testChoices,
	)
	if err != nil {
		log.Error("Failed to add vote session to blockchain", sl.Err(err))
	} else {
		log.Info("Vote session added to blockchain successfully", slog.String("tx_hash", txHash.Hex()))
		// Вы можете дождаться подтверждения транзакции, если это необходимо
		// bind.WaitMined(context.Background(), votingClient.client, tx)
	}

	// --- Пример вызова метода контракта на чтение (getVotingParticipatedByAddress) ---
	// Ваш существующий код
	fmt.Println("\n--- Testing GetVotingParticipatedByAddress ---")
	address := cfg.Blockchain.WalletAddress
	participated, err := votingClient.GetVotingParticipatedByAddress(address)
	if err != nil {
		log.Error("Failed to get participated votes", sl.Err(err))
	} else {
		if len(participated) == 0 {
			log.Info("No participated votes found for this address")
		} else {
			log.Info(fmt.Sprintf("Found %d participated votes", len(participated)))
			fmt.Println("Participated votes IDs:")
			for _, id := range participated {
				fmt.Println(id.String())
			}
		}
	}

	// --- Пример вызова новой view-функции (GetVotingCount) ---
	fmt.Println("\n--- Testing GetVotingCount ---")
	voteCount, err := votingClient.GetVotingCount()
	if err != nil {
		log.Error("Failed to get vote count", sl.Err(err))
	} else {
		log.Info(fmt.Sprintf("Total vote sessions on contract: %s", voteCount.String()))
	}*/

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

type ConnectWalletRequest struct {
	WalletAddress string `json:"walletAddress"`
}

// ConnectWalletHandler - обработчик для подключения MetaMask
func ConnectWalletHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Парсинг адреса кошелька из запроса фронтенда
	var req ConnectWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("Failed to decode connect wallet request", sl.Err(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.WalletAddress == "" {
		log.Warn("ConnectWalletHandler: Received empty wallet address from frontend")
		http.Error(w, "Wallet address is required", http.StatusBadRequest)
		return
	}

	// 2. Получение userID (адрес кошелька)
	userID := req.WalletAddress

	// 3. Вызов метода продюсера для отправки сообщения в Kafka
	// Этот вызов происходит только тогда, когда приходит HTTP-запрос на /connect-wallet
	err := kafkaProducer.UserRegistrationProduce(r.Context(), userID)
	if err != nil {
		// Если Kafka Producer не смог отправить сообщение, логируем ошибку
		// и сообщаем фронтенду об ошибке на бэкенде.
		log.Error("Failed to send user registration event to Kafka", sl.Err(err), slog.String("user_id", userID))
		http.Error(w, "Failed to process user registration event", http.StatusInternalServerError)
		return
	}

	// 4. Успешный ответ фронтенду
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(fmt.Sprintf("User %s registered successfully and event sent to Kafka", userID)))
	if err != nil {
		return
	}
}

// CreateVotingHandler - обработчик HTTP для создания голосования
func CreateVotingHandler(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Title          string   `json:"title"`
		Description    string   `json:"description"`
		IsPrivate      bool     `json:"is_private"`
		MinNumberVotes int64    `json:"min_votes"`
		StartTime      string   `json:"start_date"`
		EndTime        string   `json:"end_date"`
		Choices        []string `json:"options"`
		CreatorAddress string   `json:"creator_address"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestPayload)
	if err != nil {
		log.Error("Failed to decode create voting request", sl.Err(err))
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// --- Генерируем фиктивные ID для голосования и транзакции ---
	// Это будет использовано в ответе фронтенду и в Kafka сообщении
	votingID := fmt.Sprintf("vote_%d", time.Now().UnixNano()) // Уникальный ID голосования
	txHash := fmt.Sprintf("0x%x", time.Now().UnixNano()*1000) // Фиктивный хеш транзакции

	log.Info("Received request to create voting (blockchain skipped)",
		slog.String("voting_id_generated", votingID),
		slog.String("title", requestPayload.Title))

	// --- БЛОКЧЕЙН ЛОГИКА ВРЕМЕННО УДАЛЕНА ИЛИ ЗАКОММЕНТИРОВАНА ---
	// Здесь раньше был вызов:
	// txHash, err := votingClient.AddVoteSession(...)
	// if err != nil { ... }
	// logger.Info("Vote session added to blockchain", slog.String("tx_hash", txHash.Hex()))
	// -----------------------------------------------------------

	// Подготовка данных для Kafka в точном формате dto.VotingReq
	optionsForKafka := make([]dto.Option, len(requestPayload.Choices))
	for i, choiceText := range requestPayload.Choices {
		optionsForKafka[i] = dto.Option{
			OptionID: fmt.Sprintf("%s_opt_%d", votingID, i+1), // Генерация уникального ID для каждой опции
			Text:     choiceText,
		}
	}

	// EndDate берется напрямую из запроса как строка
	votingEvent := dto.VotingReq{
		ID:          votingID,
		Title:       requestPayload.Title,
		Description: requestPayload.Description,
		CreatorID:   requestPayload.CreatorAddress,
		Private:     requestPayload.IsPrivate,
		MinVotes:    int(requestPayload.MinNumberVotes),
		EndDate:     requestPayload.EndTime,
		StartDate:   requestPayload.StartTime,
		Options:     optionsForKafka,
	}

	// Отправка сообщения в Kafka
	err = kafkaProducer.VotingCreateProduce(r.Context(), votingEvent)
	if err != nil {
		log.Error("Failed to send voting creation event to Kafka", sl.Err(err), slog.String("voting_id", votingEvent.ID))
		// Здесь решаем, что делать, если Kafka не сработала.
		// Для простоты, пока просто логируем.
	} else {
		log.Info("Voting creation event sent to Kafka", slog.String("voting_id", votingEvent.ID))
	}

	// Возвращаем JSON-ответ фронтенду
	responseBody := map[string]string{
		"voting_id": votingID,
		"tx_hash":   txHash, // Используем фиктивный хеш
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(responseBody)
}

// SubmitVote - хендлер для обработки голосования пользователя
func SubmitVote(w http.ResponseWriter, r *http.Request) {
	// Используем models.VoteRequest из вашего старого кода
	var req models.VoteRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		slog.Error("SubmitVote: Invalid request payload", sl.Err(err))
		return
	}

	mu.Lock() // Блокируем доступ к общим данным
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

	// --- ЛОГИКА ОБНОВЛЕНИЯ СОСТОЯНИЯ В ПАМЯТИ (ИЗ СТАРОГО КОДА) ---
	// Регистрируем голос в UserActivity
	activity.ParticipatedVotings[req.VotingID] = req.SelectedOptionIndex
	userActivities[userAddressLower] = activity

	// Регистрируем голос в VoteSession.Voters
	voting.Voters[userAddressLower] = models.Voter{
		Address: req.UserAddress,
		IsVoted: true,
		Choice:  req.SelectedOptionIndex,
		CanVote: true, // Это поле здесь не играет роли, но сохраним для структуры
	}

	// Увеличиваем счетчик голосов для выбранной опции
	voting.Choices[req.SelectedOptionIndex].CountVotes++

	// Увеличиваем общий счетчик голосов для голосования
	voting.TempNumberVotes++
	votings[req.VotingID] = voting // Убедитесь, что голосование сохраняется обратно в map

	// Обновляем статус голосования сразу после голосования (опционально, но полезно)
	UpdateVotingStatusAndWinner(req.VotingID)
	// --- КОНЕЦ ЛОГИКИ ОБНОВЛЕНИЯ СОСТОЯНИЯ В ПАМЯТИ ---

	// --- ЛОГИКА ОТПРАВКИ В KAFKA (НОВАЯ) ---
	// Нам нужен OptionID. Если Choice в models.VoteSession.Choices
	// имеет поле ID, используем его. Если нет, генерируем, как раньше.
	var optionIDToKafka int
	if req.SelectedOptionIndex >= 0 && req.SelectedOptionIndex < len(voting.Choices) {
		// Предполагаем, что models.Choice имеет поле ID.
		// Если нет, и OptionID хранится как `votingID_opt_index`, то генерируем:
		// optionIDToKafka = fmt.Sprintf("%s_opt_%d", req.VotingID, req.SelectedOptionIndex)
		// Если Choice.ID существует:
	} else {
		slog.Error("SubmitVote: Internal error, invalid option index after checks",
			slog.String("voting_id", req.VotingID), slog.Int("index", req.SelectedOptionIndex))
	}

	// Подготовка данных для Kafka в точном формате dto.VoteCast
	voteEvent := dto.VoteCast{
		VotingID: req.VotingID,
		VoterID:  req.UserAddress,
		OptionID: optionIDToKafka, // Используем полученный/сгенерированный ID
	}

	// Отправка сообщения в Kafka
	err = kafkaProducer.VoteCastProduce(r.Context(), voteEvent)
	if err != nil {
		slog.Error("Failed to send vote cast event to Kafka",
			sl.Err(err),
			slog.String("voting_id", voteEvent.VotingID),
			slog.String("voter_id", voteEvent.VoterID))
		// Опять же, решаем, что делать, если Kafka не сработала.
		// В данном случае, голос уже записан в in-memory, так что это только проблема с Kafka.
	} else {
		slog.Info("Vote cast event sent to Kafka",
			slog.String("voting_id", voteEvent.VotingID),
			slog.String("voter_id", voteEvent.VoterID),
			slog.Int("option_id", voteEvent.OptionID))
	}
	// --- КОНЕЦ ЛОГИКИ ОТПРАВКИ В KAFKA ---

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(map[string]string{"message": "Vote successfully recorded"})
	if err != nil {
		slog.Error("SubmitVote: Failed to encode response", sl.Err(err))
		return
	}
	slog.Info("Vote recorded", slog.String("voting_id", req.VotingID), slog.String("user_address", req.UserAddress), slog.Int("option_index", req.SelectedOptionIndex))
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
	choices := make([]models.Choice, len(requestData.Choices))
	for i, title := range requestData.Choices {
		choices[i] = models.Choice{Title: title, CountVotes: 0}
	}

	newVoting := models.VoteSession{
		ID:              strconv.Itoa(len(votings) + 1),
		CreatorAddr:     requestData.CreatorAddress,
		Title:           requestData.Title,
		Description:     requestData.Description,
		StartTime:       requestData.StartTime,
		EndTime:         requestData.EndTime,
		MinNumberVotes:  requestData.MinNumberVotes,
		TempNumberVotes: 0,
		IsPrivate:       requestData.IsPrivate,
		Choices:         choices,                       // Используем преобразованный срез
		Voters:          make(map[string]models.Voter), // Инициализируем пустую мапу
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

// GetVotingByID - ОБНОВЛЕНО для отправки запроса деталей голосования в Kafka
func GetVotingByID(w http.ResponseWriter, r *http.Request) {
	votingID := chi.URLParam(r, "id") // Получаем ID голосования из URL

	if votingID == "" {
		http.Error(w, "Voting ID is required", http.StatusBadRequest)
		slog.Warn("GetVotingByID: Empty voting ID received")
		return
	}

	log.Info("Accessed GetVotingByID endpoint", slog.String("voting_id", votingID))

	// --- НОВОЕ: Отправка запроса деталей голосования в Kafka ---
	err := kafkaProducer.VotingRequestProduce(r.Context(), votingID)
	if err != nil {
		log.Error("Failed to send voting details request to Kafka", sl.Err(err), slog.String("voting_id", votingID))
		// Можно не возвращать ошибку фронтенду, так как это не критично для отображения голосования.
		// Просто логируем и продолжаем.
	} else {
		log.Info("Voting details request event sent to Kafka", slog.String("voting_id", votingID))
	}
	// --- КОНЕЦ НОВОГО БЛОКА ---

	mu.Lock()
	defer mu.Unlock()

	_, ok := votings[votingID]
	if !ok {
		http.Error(w, "VoteSession not found", http.StatusNotFound)
		slog.Warn("GetVotingByID: VoteSession not found", slog.String("voting_id", votingID))
		return
	}

	// Обновляем статус голосования перед отправкой
	UpdateVotingStatusAndWinner(votingID)
	updatedVoting := votings[votingID] // Получаем обновленную версию

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(updatedVoting)
	if err != nil {
		slog.Error("Failed to encode response for GetVotingByID", sl.Err(err), slog.String("voting_id", votingID))
		return
	}
}

// GetAllVotings - ОБНОВЛЕНО для отправки триггера в Kafka
func GetAllVotings(w http.ResponseWriter, r *http.Request) {
	log.Info("Accessed GetAllVotings endpoint")

	// --- НОВОЕ: Отправка триггера в Kafka ---
	err := kafkaProducer.TriggerAllVotingsProduce(r.Context())
	if err != nil {
		log.Error("Failed to send trigger all votings event to Kafka from GetAllVotings", sl.Err(err))
		// Можно не возвращать ошибку фронтенду, так как это не критично для отображения списка голосований.
		// Просто логируем и продолжаем.
	} else {
		log.Info("All votings trigger event sent to Kafka from GetAllVotings")
	}
	// --- КОНЕЦ НОВОГО БЛОКА ---

	var filteredVotings []models.VoteSession
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
	err = json.NewEncoder(w).Encode(filteredVotings)
	if err != nil {
		slog.Error("Failed to encode response for GetAllVotings", sl.Err(err))
		return
	}
}

func GetUserData(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		UserAddress string `json:"user_address"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestPayload)
	if err != nil {
		log.Error("Failed to decode user data request", sl.Err(err))
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	userAddress := requestPayload.UserAddress
	if userAddress == "" {
		log.Warn("GetUserData: Received empty user address")
		http.Error(w, "User address is required", http.StatusBadRequest)
		return
	}

	log.Info("Received request for user data", slog.String("user_address", userAddress))

	// --- НОВОЕ: Отправка запроса истории голосований в Kafka ---
	err = kafkaProducer.VoteHistoryRequestProduce(r.Context(), userAddress)
	if err != nil {
		log.Error("Failed to send vote history request to Kafka", sl.Err(err), slog.String("user_address", userAddress))
		// Здесь решаем, что делать, если Kafka не сработала.
		// Возможно, это не критично для отображения профиля, но стоит залогировать.
	} else {
		log.Info("Vote history request sent to Kafka for user", slog.String("user_address", userAddress))
	}
	// --- КОНЕЦ НОВОГО БЛОКА ---

	// --- Ваша существующая логика получения данных пользователя ---
	// Здесь вы должны получить реальные данные пользователя из вашего хранилища.
	// Например, из `userActivities` и `votings` map.
	mu.Lock() // Блокируем доступ к общим данным
	defer mu.Unlock()

	activity, activityExists := userActivities[strings.ToLower(userAddress)]
	if !activityExists {
		activity = models.UserActivity{
			CreatedVotings:      []string{},
			ParticipatedVotings: make(map[string]int),
		}
		// Если пользователя нет, можно его "зарегистрировать" или просто вернуть 0
		// userActivities[strings.ToLower(userAddress)] = activity // Если хотим создать запись при первом запросе
	}

	createdCount := 0
	participatedCount := len(activity.ParticipatedVotings)

	// Проходимся по всем голосованиям, чтобы определить, сколько создал данный пользователь
	userVotings := []models.VoteSession{}
	for _, voting := range votings {
		if strings.EqualFold(voting.CreatorAddr, userAddress) {
			createdCount++
			userVotings = append(userVotings, voting)
		} else if _, ok := activity.ParticipatedVotings[voting.ID]; ok {
			// Если пользователь участвовал в этом голосовании, добавляем его в список
			userVotings = append(userVotings, voting)
		}
	}

	// Формируем ответ для фронтенда
	responsePayload := map[string]interface{}{
		"user_address":               requestPayload.UserAddress,
		"created_votings_count":      createdCount,
		"participated_votings_count": participatedCount,
		"votings":                    userVotings, // Отправляем список голосований, связанных с пользователем
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responsePayload)
}

/*func GetUserData(w http.ResponseWriter, r *http.Request) {
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
		activity = models.UserActivity{
			CreatedVotings:      []string{},
			ParticipatedVotings: make(map[string]int),
		}
	}

	var userVotings []models.UserVotingDetail
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

			userVotings = append(userVotings, models.UserVotingDetail{
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
				userVotings = append(userVotings, models.UserVotingDetail{
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

	response := models.UserDataResponse{
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
}*/

// SubmitVote обрабатывает запрос на голосование
/*func SubmitVote(w http.ResponseWriter, r *http.Request) {
	var req models.VoteRequest
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
	voting.Voters[userAddressLower] = models.Voter{
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
}*/

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
	votings = make(map[string]models.VoteSession)
	userActivities = make(map[string]models.UserActivity)

	user1 := "0x1234567890abcdef1234567890abcdef12345678"
	user2 := "0x9876543210fedcba9876543210fedcba98765432"
	user3 := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	now := time.Now()
	futureStartDate := now.Add(48 * time.Hour)
	pastEndDate := now.Add(-1 * time.Hour)
	activeStartDate := now.Add(-24 * time.Hour)

	// Helper to create Choice slice
	createChoices := func(titles []string) []models.Choice {
		c := make([]models.Choice, len(titles))
		for i, t := range titles {
			c[i] = models.Choice{Title: t, CountVotes: 0}
		}
		return c
	}

	// Helper to create Voters map
	createVoters := func(votes map[string]int, votingChoices []models.Choice) map[string]models.Voter {
		votersMap := make(map[string]models.Voter)
		for addr, choiceIdx := range votes {
			votersMap[strings.ToLower(addr)] = models.Voter{
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
	voting1 := models.VoteSession{
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
	voting2 := models.VoteSession{
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
	voting3 := models.VoteSession{
		ID: "3", Title: "Когда провести митинг?", Description: "Выбираем удобное время для еженедельного митинга.", IsPrivate: false,
		MinNumberVotes: 1, StartTime: now.Add(-72 * time.Hour).Format(time.RFC3339), EndTime: pastEndDate.Format(time.RFC3339), Choices: choices3,
		CreatorAddr: user2, TempNumberVotes: int64(len(voters3)), Voters: voters3, Winner: []string{}, Status: "Finished", // Status will be updated by goroutine
	}

	// Voting 4: Upcoming
	choices4 := createChoices([]string{"Чат", "Опросы", "Форум"})
	voting4 := models.VoteSession{
		ID: "4", Title: "Будущий функционал", Description: "Какую функцию добавить следующей?", IsPrivate: false,
		MinNumberVotes: 1, StartTime: futureStartDate.Format(time.RFC3339), EndTime: futureStartDate.Add(72 * time.Hour).Format(time.RFC3339), Choices: choices4,
		CreatorAddr: user2, TempNumberVotes: 0, Voters: make(map[string]models.Voter), Winner: []string{}, Status: "Upcoming",
	}

	// Voting 5: Active, low votes (will be Rejected after end)
	choices5 := createChoices([]string{"AI", "Web3", "IoT"})
	voters5 := createVoters(map[string]int{
		user1: 0,
	}, choices5)
	voting5 := models.VoteSession{
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
	voting6 := models.VoteSession{
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

	userActivities[strings.ToLower(user1)] = models.UserActivity{
		CreatedVotings:      []string{"1", "2", "5", "6"},
		ParticipatedVotings: map[string]int{"3": 0, "1": 0}, // User1 voted in 3 and 1
	}

	userActivities[strings.ToLower(user2)] = models.UserActivity{
		CreatedVotings:      []string{"3", "4"},
		ParticipatedVotings: map[string]int{"1": 1, "2": 0}, // User2 voted in 1 and 2
	}

	userActivities[strings.ToLower(user3)] = models.UserActivity{
		CreatedVotings:      []string{},
		ParticipatedVotings: map[string]int{"1": 0, "3": 1}, // User3 voted in 1 and 3
	}
}
