package models

import "time"

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
	StartTime       time.Time        `json:"start_date"`  // JSON-тег остался start_date
	EndTime         time.Time        `json:"end_date"`    // JSON-тег остался end_date
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
