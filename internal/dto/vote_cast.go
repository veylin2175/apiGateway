package dto

// topic: vote-cast

type VoteCast struct {
	VotingID string `json:"votingId"`
	VoterID  string `json:"voterId"`
	OptionID int    `json:"optionId"`
}
