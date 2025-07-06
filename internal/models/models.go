package models

import "time"

type Model struct {
	User   Users  `json:"user"`
	Voting Voting `json:"voting"`
}

type Users struct {
	ID            string        `json:"id"`
	Username      string        `json:"username"`
	RegisteredAt  time.Time     `json:"registered_at"`
	Subscriptions Subscriptions `json:"subscriptions"`
}

type Voting struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	CreatorID     string          `json:"creator_id"`
	IsPrivate     bool            `json:"is_private"`
	MinVotes      int             `json:"min_votes"`
	EndDate       time.Time       `json:"end_date"`
	CreatedAt     time.Time       `json:"created_at"`
	VotingOptions []VotingOptions `json:"voting_options"`
	Votes         []Votes         `json:"votes"`
}

type VotingOptions struct {
	VotingID string `json:"voting_id"`
	OptionID int8   `json:"option_id"`
	Text     string `json:"text"`
}

type Votes struct {
	VotingID  string    `json:"voting_id"`
	VoterID   string    `json:"voter_id"`
	OptionID  int8      `json:"option_id"`
	TxHash    string    `json:"tx_hash"`
	CreatedAt time.Time `json:"created_at"`
}

type Subscriptions struct {
	UserID    string    `json:"user_id"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}
