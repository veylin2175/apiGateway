package dto

// topic: all-votings-response

type AllVotingRes struct {
	VotingID    string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	StartDate   float64 `json:"startDate"`
	EndDate     float64 `json:"endDate"`
}

type AllVotingsKafkaResponse struct {
	Votings []AllVotingRes `json:"votings"`
}
