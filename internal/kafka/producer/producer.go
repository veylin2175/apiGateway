package producer

import (
	"apiGateway/internal/config"
	"apiGateway/internal/dto" // Убедитесь, что dto.UserIdReq определен здесь
	"context"
	"encoding/json"
	"fmt"  // Для использования fmt.Errorf
	"time" // Для BatchTimeout

	"github.com/segmentio/kafka-go"
	"log/slog" // Используем slog
)

// Producer обертка для Segmentio Kafka Writer.
type Producer struct {
	writer *kafka.Writer
	log    *slog.Logger // Добавляем логгер
}

// NewProducer создает и возвращает новый экземпляр Producer.
// Теперь принимает slog.Logger и использует более полную конфигурацию Kafka.
func NewProducer(cfg config.Kafka, log *slog.Logger) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers not provided in configuration")
	}

	writer := kafka.Writer{
		Addr:        kafka.TCP(cfg.Brokers...), // Использование varargs для нескольких брокеров
		Balancer:    &kafka.LeastBytes{},       // Балансировщик
		Logger:      kafka.LoggerFunc(func(msg string, args ...interface{}) { log.Debug(msg, args...) }),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) { log.Error(msg, args...) }),
	}

	p := &Producer{
		writer: &writer,
		log:    log,
	}

	log.Info("Kafka producer initialized successfully", slog.Any("brokers", cfg.Brokers))
	return p, nil
}

// UserRegistrationProduce отправляет userID в топик "user-registration".
// Возвращает ошибку, чтобы вызывающая сторона могла ее обработать.
func (p *Producer) UserRegistrationProduce(ctx context.Context, userID string) error {
	user := dto.UserIdReq{
		UserID: userID,
	}

	value, err := json.Marshal(user)
	if err != nil {
		p.log.Error("Failed to marshal UserIdReq", slog.String("user_id", userID), slog.Any("error", err))
		return fmt.Errorf("failed to marshal user ID: %w", err)
	}

	message := kafka.Message{
		Topic: "user-registrations",
		Key:   []byte(userID), // Используем userID как ключ, чтобы все сообщения от одного пользователя шли в одну партицию
		Value: value,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte("UserRegistered")},
			{Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339))},
		},
	}

	err = p.writer.WriteMessages(ctx, message)
	if err != nil {
		p.log.Error("Failed to write message to Kafka", slog.String("topic", message.Topic), slog.String("user_id", userID), slog.Any("error", err))
		return fmt.Errorf("error writing to kafka topic %s: %w", message.Topic, err)
	}

	p.log.Info("Message sent successfully to Kafka", slog.String("topic", message.Topic), slog.String("user_id", userID))
	return nil
} // РАБОТАЕТ

// VoteHistoryRequestProduce отправляет userID в топик "vote-history-request".
// Возвращает ошибку, чтобы вызывающая сторона могла ее обработать.
func (p *Producer) VoteHistoryRequestProduce(ctx context.Context, userID string) error {
	user := dto.UserIdReq{
		UserID: userID,
	}

	value, err := json.Marshal(user)
	if err != nil {
		p.log.Error("Failed to marshal UserIdReq", slog.String("user_id", userID), slog.Any("error", err))
		return fmt.Errorf("failed to marshal user ID: %w", err)
	}

	message := kafka.Message{
		Topic: "vote-history-request",
		Key:   []byte(userID), // Используем userID как ключ, чтобы все сообщения от одного пользователя шли в одну партицию
		Value: value,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte("VoteHistoryRequested")},
			{Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339))},
		},
	}

	err = p.writer.WriteMessages(ctx, message)
	if err != nil {
		p.log.Error("Failed to write message to Kafka", slog.String("topic", message.Topic), slog.String("user_id", userID), slog.Any("error", err))
		return fmt.Errorf("error writing to kafka topic %s: %w", message.Topic, err)
	}

	p.log.Info("Message sent successfully to Kafka", slog.String("topic", message.Topic), slog.String("user_id", userID))
	return nil
} // РАБОТАЕТ

// TriggerAllVotingsProduce отправляет пустое сообщение в топик "trigger-all-votings".
// Это служит триггером для других сервисов обновить информацию обо всех голосованиях.
func (p *Producer) TriggerAllVotingsProduce(ctx context.Context) error {
	// Пустой JSON как значение сообщения
	value := []byte("{}")

	message := kafka.Message{
		Topic: "trigger-all-votings",
		Key:   nil,
		Value: value,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte("AllVotingsTriggered")},
			{Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339))},
		},
	}

	err := p.writer.WriteMessages(ctx, message)
	if err != nil {
		p.log.Error("Failed to write trigger all votings message to Kafka",
			slog.String("topic", message.Topic),
			slog.Any("error", err))
		return fmt.Errorf("error writing trigger all votings event to kafka topic %s: %w", message.Topic, err)
	}

	p.log.Info("Trigger all votings message sent successfully to Kafka", slog.String("topic", message.Topic))
	return nil
} // РАБОТАЕТ

// VotingRequestProduce отправляет votingID в топик "voting-request".
// Возвращает ошибку, чтобы вызывающая сторона могла ее обработать.
func (p *Producer) VotingRequestProduce(ctx context.Context, VotingID string) error {
	voting := dto.VotingRequest{
		VotingID: VotingID,
	}

	value, err := json.Marshal(voting)
	if err != nil {
		p.log.Error("Failed to marshal VotingIdReq", slog.String("voting_id", VotingID), slog.Any("error", err))
		return fmt.Errorf("failed to marshal voting ID: %w", err)
	}

	message := kafka.Message{
		Topic: "voting-request",
		Key:   []byte(VotingID), // Используем VotingID как ключ, чтобы все сообщения от одного пользователя шли в одну партицию
		Value: value,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte("VotingDetailsRequested")},
			{Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339))},
		},
	}

	err = p.writer.WriteMessages(ctx, message)
	if err != nil {
		p.log.Error("Failed to write message to Kafka", slog.String("topic", message.Topic), slog.String("user_id", VotingID), slog.Any("error", err))
		return fmt.Errorf("error writing to kafka topic %s: %w", message.Topic, err)
	}

	p.log.Info("Message sent successfully to Kafka", slog.String("topic", message.Topic), slog.String("user_id", VotingID))
	return nil
} // РАБОТАЕТ

// VotingCreateProduce отправляет сообщение о создании нового голосования в Kafka.
// Она принимает контекст и структуру dto.VotingReq.
func (p *Producer) VotingCreateProduce(ctx context.Context, votingData dto.VotingReq) error {
	// Сериализуем структуру VotingReq в JSON.
	// JSON-тэги (`json:"..."`) гарантируют правильные имена полей в JSON-выводе.
	value, err := json.Marshal(votingData)
	if err != nil {
		p.log.Error("Failed to marshal VotingReq", slog.String("voting_id", votingData.ID), slog.Any("error", err))
		return fmt.Errorf("failed to marshal voting data: %w", err)
	}

	// Используем ID голосования в качестве ключа сообщения.
	// Это гарантирует, что все события, относящиеся к одному голосованию,
	// будут попадать в одну и ту же партицию, сохраняя порядок.
	key := []byte(votingData.ID)

	// Добавляем заголовки для лучшей идентификации события.
	headers := []kafka.Header{
		{Key: "event_type", Value: []byte("VotingCreated")},
		{Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339))},
	}

	message := kafka.Message{
		Topic:   "voting-create",
		Key:     key,
		Value:   value,
		Headers: headers,
	}

	// Отправляем сообщение в Kafka.
	err = p.writer.WriteMessages(ctx, message)
	if err != nil {
		p.log.Error("Failed to write voting creation message to Kafka",
			slog.String("topic", message.Topic),
			slog.String("voting_id", votingData.ID),
			slog.Any("error", err))
		return fmt.Errorf("error writing voting creation event to kafka topic %s: %w", message.Topic, err)
	}

	p.log.Info("Voting creation message sent successfully to Kafka",
		slog.String("topic", message.Topic),
		slog.String("voting_id", votingData.ID))
	return nil
} // РАБОТАЕТ

// VoteCastProduce отправляет сообщение о голосовании пользователя в Kafka.
// Она принимает контекст и структуру dto.VoteCast.
func (p *Producer) VoteCastProduce(ctx context.Context, voteData dto.VoteCast) error {
	// Сериализуем структуру VoteCast в JSON.
	value, err := json.Marshal(voteData)
	if err != nil {
		p.log.Error("Failed to marshal VoteCast",
			slog.String("voting_id", voteData.VotingID),
			slog.String("voter_id", voteData.VoterID),
			slog.Int("option_id", voteData.OptionID),
			slog.Any("error", err))
		return fmt.Errorf("failed to marshal vote cast data: %w", err)
	}

	// Используем VotingID + VoterID в качестве ключа сообщения для обеспечения порядка
	// событий от одного пользователя в рамках одного голосования, или просто VotingID
	// если порядок внутри голосования не так критичен, но все события голосования
	// должны быть вместе.
	// Для уникальности и распределения по партициям, можно использовать конкатенацию.
	key := []byte(fmt.Sprintf("%s-%s", voteData.VotingID, voteData.VoterID))

	// Добавляем заголовки для лучшей идентификации события.
	headers := []kafka.Header{
		{Key: "event_type", Value: []byte("VoteCast")},
		{Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339))},
		{Key: "source_service", Value: []byte("api-gateway")},
	}

	message := kafka.Message{
		Topic:   "vote-cast",
		Key:     key,
		Value:   value,
		Headers: headers,
	}

	// Отправляем сообщение в Kafka.
	err = p.writer.WriteMessages(ctx, message)
	if err != nil {
		p.log.Error("Failed to write vote cast message to Kafka",
			slog.String("topic", message.Topic),
			slog.String("voting_id", voteData.VotingID),
			slog.String("voter_id", voteData.VoterID),
			slog.Any("error", err))
		return fmt.Errorf("error writing vote cast event to kafka topic %s: %w", message.Topic, err)
	}

	p.log.Info("Vote cast message sent successfully to Kafka",
		slog.String("topic", message.Topic),
		slog.String("voting_id", voteData.VotingID),
		slog.String("voter_id", voteData.VoterID))
	return nil
} // НЕ ТЕСТИЛИ

// Close закрывает продюсер и очищает ресурсы.
func (p *Producer) Close() {
	p.log.Info("closing kafka producer")
	if err := p.writer.Close(); err != nil {
		p.log.Error("failed to close Kafka writer", slog.Any("error", err))
	} else {
		p.log.Info("kafka producer closed.")
	}
}
