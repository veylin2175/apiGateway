package dto

// topic: vote-history-response

type History struct {
	VotingID    string `json:"votingId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CastAt      string `json:"castAt"`
	OptionID    string `json:"optionId"`
	OptionText  string `json:"optionText"`
}
