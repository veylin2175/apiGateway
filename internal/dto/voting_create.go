package dto

// topic: voting-create

type VotingReq struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	CreatorID   string   `json:"creatorId"`
	Private     bool     `json:"private"`
	MinVotes    int      `json:"minVotes"`
	EndDate     string   `json:"endDate"`
	StartDate   string   `json:"startDate"`
	Options     []Option `json:"options"`
}

type Option struct {
	OptionID string `json:"optionId"`
	Text     string `json:"text"`
}
