package models

import "time"

type Users struct {
	UserID             string   `json:"user_id"`             // Адрес кошелька
	CreatedVoting      []Voting `json:"create_voting"`       // Список всех созданных пользователем голосований
	ParticipatedVoting []Voting `json:"participated_voting"` // Список голосований, в которых участвовал пользователь
	SubscribedVoting   []Voting `json:"subscribed_voting"`   // (опционально) Подписки
}

type Voting struct {
	VotingID      string          `json:"voting_id"`      // ID голосования
	Title         string          `json:"title"`          // Название
	Description   string          `json:"description"`    // Описание
	CreatorID     string          `json:"creator_id"`     // Адрес кошелька создателя голосования
	IsPrivate     bool            `json:"is_private"`     // Приватное / публичное
	MinVotes      int             `json:"min_votes"`      // Мин кол-во голосов
	EndDate       time.Time       `json:"end_date"`       // Дата время окончания
	CreatedAt     time.Time       `json:"created_at"`     // Дата создания / публикации
	VotingOptions []VotingOptions `json:"voting_options"` // Список вариантов ответа
	Votes         []Votes         `json:"votes"`          // Список проголосовавших
}

type VotingOptions struct {
	VotingID string `json:"voting_id"` // ID голосования
	OptionID int8   `json:"option_id"` // ID варианта ответа
	Text     string `json:"text"`      // Текст варианта ответа
}

type Votes struct {
	VotingID string `json:"voting_id"` // ID голосования
	VoterID  string `json:"voter_id"`  // ID проголосовавшего
	OptionID int8   `json:"option_id"` // ID варианта ответа
}

type Subscriptions struct { // Опционально
	UserID   string `json:"user_id"`   // ID того, кто подписан
	AuthorID string `json:"author_id"` // ID того, НА кого подписан
}
