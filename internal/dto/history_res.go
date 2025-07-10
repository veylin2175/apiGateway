package dto

// topic: vote-history-response -- отображение в профиле

type History struct {
	Title       string `json:"title"`
	VotersCount int    `json:"votersCount"`
	IsPrivate   bool   `json:"private"`
	OptionText  string `json:"optionText"`
}
