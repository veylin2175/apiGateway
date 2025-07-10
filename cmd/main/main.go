package main

import (
	"apiGateway/internal/config"
	"apiGateway/internal/dto"
	"apiGateway/internal/http-server/middleware/mwlogger"
	"apiGateway/internal/kafka/consumer"
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

func init() {}

func main() {
	cfg := config.MustLoad()

	log = setupLogger(cfg.Env)

	log.Info("Starting voting service", slog.String("env", cfg.Env))
	log.Debug("Debug messages are enabled")

	ctx, cancel := context.WithCancel(context.Background())
	rwmu := &sync.RWMutex{}
	wg := &sync.WaitGroup{}
	wg.Add(1)

	allVotingConsumer := consumer.NewConsumer(cfg.Kafka, "all-votings-response", votings, log)
	wg.Add(1)
	go allVotingConsumer.RunAllVotingsMain(ctx, wg)

	votingConsumer := consumer.NewConsumer(cfg.Kafka, "voting-response", votings, log)
	wg.Add(1)
	go votingConsumer.RunVotingByIdMain(ctx, wg)

	historyConsumer := consumer.NewConsumer(cfg.Kafka, "vote-history-response", votings, log)
	wg.Add(1)
	go historyConsumer.RunVoteHistoryConsumer(ctx, wg)

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
	router.Post("/user-data", GetUserData(log, historyConsumer, kafkaProducer, userActivities, votings, rwmu))
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
	log.Info("Kafka consumers stopped.")

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
	err = json.NewEncoder(w).Encode(responseBody)
	if err != nil {
		return
	}
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
	startDate := voting.StartTime
	if time.Now().Before(startDate) {
		http.Error(w, "VoteSession has not started yet", http.StatusForbidden)
		slog.Warn("SubmitVote: VoteSession has not started", slog.String("voting_id", req.VotingID))
		return
	}

	// Проверяем, закончилось ли голосование
	endDate := voting.EndTime
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
	var optionIDToKafka string
	if req.SelectedOptionIndex >= 0 && req.SelectedOptionIndex < len(voting.Choices) {
		// Предполагаем, что models.Choice имеет поле ID.
		// Если нет, и OptionID хранится как `votingID_opt_index`, то генерируем:
		optionIDToKafka = fmt.Sprintf("%s_opt_%d", req.VotingID, req.SelectedOptionIndex)
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
			slog.String("option_id", voteEvent.OptionID))
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

// GetUserData теперь является функцией, которая возвращает http.HandlerFunc.
// Она принимает все необходимые зависимости как аргументы.
func GetUserData(
	log *slog.Logger,
	consumerInstance *consumer.Consumer, // Экземпляр Consumer
	kafkaProducer *producer.Producer, // Экземпляр Producer
	userActivities map[string]models.UserActivity, // Глобальная мапа userActivities
	votings map[string]models.VoteSession, // Глобальная мапа votings
	mu *sync.RWMutex, // Глобальный мьютекс для votings и userActivities
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestPayload struct {
			UserAddress string `json:"user_address"`
		}

		err := json.NewDecoder(r.Body).Decode(&requestPayload)
		if err != nil {
			log.Error("GetUserData: Failed to decode user data request", sl.Err(err))
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

		// --- ОТПРАВКА ЗАПРОСА ИСТОРИИ ГОЛОСОВАНИЙ В KAFKA (fire-and-forget) ---
		// Это инициирует Java-сервис отправить историю в топик `vote-history-response`,
		// которую консьюмер Go-сервиса сохранит в consumerInstance.UserProfilesHistory.
		err = kafkaProducer.VoteHistoryRequestProduce(r.Context(), userAddress)
		if err != nil {
			log.Error("Failed to send vote history request to Kafka", sl.Err(err), slog.String("user_address", userAddress))
			// Продолжаем выполнять запрос, даже если триггер не сработал.
		} else {
			log.Info("Vote history request sent to Kafka for user", slog.String("user_address", userAddress))
		}
		// --- КОНЕЦ БЛОКА ОТПРАВКИ ТРИГГЕРА ---

		// --- ВАША СУЩЕСТВУЮЩАЯ ЛОГИКА ПОЛУЧЕНИЯ ДАННЫХ ПОЛЬЗОВАТЕЛЯ ---
		mu.Lock()
		defer mu.Unlock()

		activity, activityExists := userActivities[strings.ToLower(userAddress)]
		if !activityExists {
			activity = models.UserActivity{
				CreatedVotings:      []string{},
				ParticipatedVotings: make(map[string]int),
			}
		}

		createdCount := 0
		participatedCount := len(activity.ParticipatedVotings)

		userVotings := []models.VoteSession{}
		for _, voting := range votings {
			if strings.EqualFold(voting.CreatorAddr, userAddress) {
				createdCount++
				userVotings = append(userVotings, voting)
			} else if _, ok := activity.ParticipatedVotings[voting.ID]; ok {
				userVotings = append(userVotings, voting)
			}
		}

		// --- ПОЛУЧАЕМ ИСТОРИЮ ИЗ ПАМЯТИ ---
		consumerInstance.Mu.Lock() // Блокируем consumerInstance.UserProfilesHistory
		historyData, found := consumerInstance.UserProfilesHistory[userAddress]
		if !found {
			historyData = []dto.History{} // Если истории нет, возвращаем пустой слайс
			log.Info("GetUserData: No history found in memory for user", slog.String("user_address", userAddress))
		}
		consumerInstance.Mu.Unlock()

		// --- ФОРМИРУЕМ ОТВЕТ ДЛЯ ФРОНТЕНДА ---
		type UserProfileResponse struct {
			UserAddress              string               `json:"user_address"`
			CreatedVotingsCount      int                  `json:"created_votings_count"`
			ParticipatedVotingsCount int                  `json:"participated_votings_count"`
			Votings                  []models.VoteSession `json:"votings"`
			History                  []dto.History        `json:"history"` // Добавляем историю из памяти
		}

		responsePayload := UserProfileResponse{
			UserAddress:              requestPayload.UserAddress,
			CreatedVotingsCount:      createdCount,
			ParticipatedVotingsCount: participatedCount,
			Votings:                  userVotings,
			History:                  historyData, // <-- Прямое использование данных из памяти
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(responsePayload)
		if err != nil {
			log.Error("GetUserData: Failed to encode responsePayload", sl.Err(err))
			return
		}
	}
}

// UpdateVotingStatusAndWinner обновляет статус голосования и определяет победителя
// Должен вызываться под мьютексом
func UpdateVotingStatusAndWinner(votingID string) {
	voting, ok := votings[votingID]
	if !ok {
		return // Голосование не найдено
	}

	now := time.Now()
	startDate := voting.StartTime
	endDate := voting.EndTime

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
