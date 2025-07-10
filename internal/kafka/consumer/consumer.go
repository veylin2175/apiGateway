package consumer

import (
	"apiGateway/internal/config"
	"apiGateway/internal/dto"
	"apiGateway/internal/models"
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"log/slog"
	"sync"
	"time"
)

// Consumer инкапсулирует Kafka reader и хранилище данных.
type Consumer struct {
	reader              *kafka.Reader
	Mu                  *sync.RWMutex
	CurrentVotings      map[string]models.VoteSession // Изменил тип на models.VoteSession, как в main
	Log                 *slog.Logger                  // Добавляем логгер
	votingResponseChans map[string]chan models.VoteSession
	UserProfilesHistory map[string][]dto.History
}

// NewConsumer создает новый консюмер Kafka.
// Передаем сюда map, который будем обновлять.
func NewConsumer(cfg config.Kafka, topic string, currentVotings map[string]models.VoteSession, logger *slog.Logger) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     cfg.Brokers,
			Topic:       topic,
			GroupID:     cfg.GroupID,
			StartOffset: kafka.LastOffset, // Начинать чтение с последнего сообщения
			MaxBytes:    10e6,
			Dialer: &kafka.Dialer{ // Добавим таймаут для соединения
				Timeout: 10 * time.Second,
			},
		}),
		Mu:                  &sync.RWMutex{},
		CurrentVotings:      currentVotings, // Получаем ссылку на общую map из main
		Log:                 logger,
		votingResponseChans: make(map[string]chan models.VoteSession),
		UserProfilesHistory: make(map[string][]dto.History),
	}
}

// RunAllVotingsMain запускает консюмер для получения всех голосований на основной странице.
// RunAllVotingsMain запускает консьюмер для получения всех голосований на основной странице.
func (c *Consumer) RunAllVotingsMain(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	c.Log.Info("Starting Kafka consumer for all-votings-response", slog.String("topic", c.reader.Config().Topic), slog.String("group_id", c.reader.Config().GroupID))

	for {
		select {
		case <-ctx.Done():
			c.Log.Info("Stopping Kafka consumer...")
			if err := c.reader.Close(); err != nil {
				c.Log.Error("Failed to close Kafka reader", slog.Any("error", err))
			}
			return
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					c.Log.Debug("Kafka read error during shutdown", slog.Any("error", err))
					continue
				}
				c.Log.Error("Kafka read error", slog.String("error_details", err.Error()), slog.Any("full_error", err))
				continue
			}

			var rawMessage string
			if err := json.Unmarshal(msg.Value, &rawMessage); err != nil {
				c.Log.Error("Failed to unmarshal Kafka message into raw string (likely double-encoded JSON)",
					slog.Any("error", err),
					slog.String("message_value", string(msg.Value)))
				continue
			}

			// Используем новую структуру для анмаршалинга
			var kafkaResponse dto.AllVotingsKafkaResponse // <-- ИЗМЕНЕНО
			if err := json.Unmarshal([]byte(rawMessage), &kafkaResponse); err != nil {
				c.Log.Error("Failed to unmarshal all the votings from raw string (check JSON structure vs DTO)",
					slog.Any("error", err),
					slog.String("raw_message_value", rawMessage))
				continue
			}

			// Теперь получаем слайс голосований из поля Votings новой структуры
			receivedVotings := kafkaResponse.Votings // <-- ИЗМЕНЕНО

			c.Log.Info("Successfully consumed all votings list", slog.Int("count", len(receivedVotings)))

			c.Mu.Lock()
			for k := range c.CurrentVotings {
				delete(c.CurrentVotings, k)
			}

			for _, v := range receivedVotings {
				// Преобразование float64 в int64 перед передачей в time.Unix
				startTime := time.Unix(int64(v.StartDate), 0)
				endTime := time.Unix(int64(v.EndDate), 0)

				newVoting := models.VoteSession{
					ID:              v.VotingID, // Поле "id" в AllVotingRes теперь мапится на VotingID
					Title:           v.Title,
					Description:     v.Description,
					StartTime:       startTime,
					EndTime:         endTime,
					CreatorAddr:     "",
					IsPrivate:       false,
					MinNumberVotes:  0,
					TempNumberVotes: 0,
					Choices:         []models.Choice{},
					Voters:          make(map[string]models.Voter),
					Winner:          []string{},
					Status:          "Upcoming",
				}
				c.CurrentVotings[newVoting.ID] = newVoting
			}
			c.Mu.Unlock()
			c.Log.Info("Global votings map updated from Kafka", slog.Int("new_count", len(c.CurrentVotings)))
		}
	}
} // РАБОТАЕТ

// --- НОВЫЙ КОНСЮМЕР ДЛЯ ОДНОГО ГОЛОСОВАНИЯ ---
func (c *Consumer) RunVotingByIdMain(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	c.Log.Info("Starting Kafka consumer for voting-response", slog.String("topic", c.reader.Config().Topic), slog.String("group_id", c.reader.Config().GroupID))

	for {
		select {
		case <-ctx.Done():
			c.Log.Info("Stopping Kafka consumer for voting-response...")
			if err := c.reader.Close(); err != nil {
				c.Log.Error("Failed to close Kafka reader for voting-response", slog.Any("error", err))
			}
			return
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					c.Log.Debug("Kafka read error during shutdown for voting-response", slog.Any("error", err))
					continue
				}
				c.Log.Error("Kafka read error for voting-response", slog.String("error_details", err.Error()), slog.Any("full_error", err))
				continue
			}

			// Применяем двойной Unmarshal для обработки двойной сериализации JSON
			var rawMessage string
			if err := json.Unmarshal(msg.Value, &rawMessage); err != nil {
				c.Log.Error("Failed to unmarshal Kafka message into raw string (likely double-encoded JSON) for voting-response",
					slog.Any("error", err),
					slog.String("message_value", string(msg.Value)))
				continue
			}

			var receivedVoting dto.VotingKafkaResponse
			if err := json.Unmarshal([]byte(rawMessage), &receivedVoting); err != nil {
				c.Log.Error("Failed to unmarshal single voting response from raw string (check JSON structure vs DTO) for voting-response",
					slog.Any("error", err),
					slog.String("raw_message_value", rawMessage))
				continue
			}

			var actualVotesCount int64
			if receivedVoting.VotesCount != "" { // Проверяем, что не пустая строка
				val, err := receivedVoting.VotesCount.Int64()
				if err != nil {
					c.Log.Error("Failed to parse VotesCount from json.Number to int64",
						slog.Any("error", err),
						slog.String("votes_count_json_number", receivedVoting.VotesCount.String()))
					// Продолжаем с 0 или пропускаем, в зависимости от желаемого поведения
					continue // Пропускаем сообщение, если не можем распарсить
				}
				actualVotesCount = val
			}

			c.Log.Info("Successfully unmarshalled VotingKafkaResponse",
				slog.String("voting_id_from_dto", receivedVoting.VotingID),
				slog.Int64("votes_count_from_dto", actualVotesCount)) // Используем actualVotesCount здесь

			// *****************************************************************
			// --- Извлекаем ID голосования из тела JSON-сообщения ---
			// *****************************************************************
			votingID := receivedVoting.VotingID // <--- ИЗМЕНЕНО: теперь берем ID из DTO
			if votingID == "" {
				// Это критическая ошибка, так как без ID мы не можем обновить мапу.
				c.Log.Error("Received single voting response with empty VotingID in JSON body. Message cannot be processed.", slog.String("message_value", rawMessage))
				continue // Сообщение игнорируется, так как не можем его идентифицировать
			}

			var calculatedTotalVotes int64
			for _, opt := range receivedVoting.Options {
				calculatedTotalVotes += int64(opt.VoteCount)
			}

			if actualVotesCount == 0 {
				c.Log.Error("ERROR: Votes count is equal to 0 (after conversion)",
					slog.String("voting_id", receivedVoting.VotingID))
			} else {
				c.Log.Info("INFO: Votes count is NOT zero (after conversion)",
					slog.String("voting_id", receivedVoting.VotingID),
					slog.Int64("votes_count", actualVotesCount))
			}

			c.Log.Info("Successfully consumed single voting response", slog.String("voting_id", votingID), slog.String("title", receivedVoting.Title))

			// Преобразование float64 UNIX timestamp в time.Time
			startTime := time.Unix(int64(receivedVoting.StartDate), 0)
			endDate := time.Unix(int64(receivedVoting.EndDate), 0)

			// Преобразование OptionRes в models.Choice
			var choices []models.Choice
			for _, opt := range receivedVoting.Options {
				choices = append(choices, models.Choice{
					Title:      opt.Text,
					CountVotes: int64(opt.VoteCount),
				})
			}

			c.Mu.Lock()
			currentVoting, exists := c.CurrentVotings[votingID]
			if !exists {
				c.Log.Debug("Creating new VoteSession entry for received single voting response", slog.String("voting_id", votingID))
				currentVoting = models.VoteSession{
					ID: votingID, // ID берем из поля DTO
					// Инициализация остальных полей по умолчанию
					IsPrivate: false,
					Voters:    make(map[string]models.Voter),
					Winner:    []string{},
					Status:    "Upcoming",
				}
			}

			// Обновляем поля VoteSession
			currentVoting.Title = receivedVoting.Title
			currentVoting.Description = receivedVoting.Description
			currentVoting.CreatorAddr = receivedVoting.CreatorID
			currentVoting.MinNumberVotes = receivedVoting.MinVotes
			currentVoting.StartTime = startTime
			currentVoting.EndTime = endDate
			currentVoting.TempNumberVotes = calculatedTotalVotes
			currentVoting.Choices = choices

			// Поля, которые не обновляются этим сообщением, сохраняют свои значения.
			// Если эти поля уже были установлены ранее, они останутся без изменений.

			c.CurrentVotings[votingID] = currentVoting
			c.Mu.Unlock()
			c.Log.Info("Global votings map updated from Kafka with single voting data", slog.String("voting_id", votingID))

			c.Mu.Lock()                                        // Блокируем мьютекс для votingResponseChans
			respChan, found := c.votingResponseChans[votingID] // Ищем канал по votingID
			if found {
				select {
				case respChan <- currentVoting: // Отправляем обновленное голосование в канал
					c.Log.Debug("Sent updated voting data to response channel", slog.String("voting_id", votingID))
				case <-time.After(100 * time.Millisecond): // Таймаут на случай, если канал уже неактивен
					c.Log.Warn("Timeout sending voting data to response channel, channel likely closed", slog.String("voting_id", votingID))
				}
				delete(c.votingResponseChans, votingID) // Удаляем канал после отправки/таймаута
			} else {
				c.Log.Debug("No active response channel found for voting ID. Data updated in map only.", slog.String("voting_id", votingID))
			}
			c.Mu.Unlock() // Отпускаем мьютекс responseMu
		}
	}
}

// RunVoteHistoryConsumer теперь будет просто обновлять UserProfilesHistory
func (c *Consumer) RunVoteHistoryConsumer(ctx context.Context, wg *sync.WaitGroup) {
	type JavaHistoryMessage struct {
		UserID  string        `json:"userId"`
		History []dto.History `json:"history"`
	}

	defer wg.Done()

	// Создаем новый ридер для этого топика, как и раньше
	historyReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092"},             // Или ваш адрес Kafka-брокера
		Topic:    "vote-history-response",                // Топик для истории профиля
		GroupID:  "go-app-profile-history-updater-group", // Уникальная группа
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer func() {
		if err := historyReader.Close(); err != nil {
			c.Log.Error("Failed to close Kafka reader for vote-history-response", slog.Any("error", err))
		}
	}()

	c.Log.Info("Starting Kafka consumer for vote-history-response",
		slog.String("topic", historyReader.Config().Topic),
		slog.String("group_id", historyReader.Config().GroupID))

	for {
		select {
		case <-ctx.Done():
			c.Log.Info("Stopping Kafka consumer for vote-history-response...")
			return
		default:
			msg, err := historyReader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					c.Log.Debug("Kafka read error during shutdown for vote-history-response", slog.Any("error", err))
					continue
				}
				c.Log.Error("Kafka read error for vote-history-response", slog.String("error_details", err.Error()), slog.Any("full_error", err))
				continue
			}

			var rawMessage string
			if err := json.Unmarshal(msg.Value, &rawMessage); err != nil {
				c.Log.Error("Failed to unmarshal Kafka message into raw string (likely double-encoded JSON)",
					slog.Any("error", err),
					slog.String("message_value", string(msg.Value)))
				continue
			}

			var javaMsg JavaHistoryMessage
			if err := json.Unmarshal([]byte(rawMessage), &javaMsg); err != nil {
				c.Log.Error("Failed to unmarshal Java history message (check JSON structure vs DTO)",
					slog.Any("error", err),
					slog.String("raw_message_value", rawMessage))
				continue
			}

			userAddress := javaMsg.UserID // <--- ПОЛУЧАЕМ userAddress ИЗ ЗНАЧЕНИЯ СООБЩЕНИЯ
			if userAddress == "" {
				c.Log.Warn("Received vote history message with empty userId in value. Skipping.",
					slog.String("message_value", string(msg.Value)))
				continue
			}

			receivedHistory := javaMsg.History

			c.Log.Info("Successfully consumed vote history events",
				slog.String("user_address", userAddress),
				slog.Int("num_entries", len(receivedHistory)))

			// ОБНОВЛЯЕМ ГЛОБАЛЬНУЮ МАПУ ИСТОРИИ ПРОФИЛЯ
			c.Mu.Lock()
			c.UserProfilesHistory[userAddress] = receivedHistory
			c.Mu.Unlock()

			c.Log.Info("User profile history map updated",
				slog.String("user_address", userAddress),
				slog.Int("history_count_in_map", len(c.UserProfilesHistory[userAddress])))
		}
	}
}
