package dto

// topic: voting-response

type VotingRes struct {
	VotingID    string `json:"votingId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatorID   string `json:"creatorId"`
	MinVotes    string `json:"minVotes"`
	EndDate     string `json:"endDate"`
	CreatedAt   string `json:"createdAt"`
}

type Options struct {
	OptionID  string `json:"optionId"`
	Text      string `json:"text"`
	VoteCount int    `json:"voteCount"`
}
