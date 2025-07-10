package dto

import "encoding/json"

// topic: voting-response -- отображение одного голосования

type OptionRes struct {
	OptionID  string `json:"optionId"`
	Text      string `json:"text"`
	VoteCount int    `json:"voteCount"`
}

type VotingKafkaResponse struct {
	VotingID    string      `json:"votingId"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	CreatorID   string      `json:"creatorId"`  // Соответствует CreatorID из Java
	MinVotes    int64       `json:"minVotes"`   // В Go нужно будет парсить в int64
	EndDate     float64     `json:"endDate"`    // В Go нужно будет парсить в time.Time
	StartDate   float64     `json:"startDate"`  // В Go нужно будет парсить в time.Time
	VotesCount  json.Number `json:"votesCount"` // Изменил на int64, чтобы избежать переполнения, если вдруг очень много голосов.
	Options     []OptionRes `json:"options"`    // Список опций
}
