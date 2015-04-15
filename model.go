package hunt

type Tag struct {
	Id string `json:"id"`
}

type Question struct {
	Question      string   `json:"question"`
	Answers       []string `json:"answers"`
	CorrectAnswer int      `json:"correctAnswer"`
	WrongMsg      string   `json:"wrongMessage"`
	RightMsg      string   `json:"rightMessage"`
}

type Clue struct {
	Id           string    `json:"id"`
	Type         string    `json:"type"`
	ShuffleGroup int       `json:"shufflegroup"`
	DisplayName  string    `json:"displayName"`
	DisplayText  string    `json:"displayText"`
	DisplayImage string    `json:"displayImage"`
	Tags         []*Tag    `json:"tags,omitempty"`
	Questions    *Question `json:"question,omitempty"`
}

type Hunt struct {
	Id          string  `json:"id" datastore:"id"`
	Type        string  `json:"type" datastore:"type"`
	DisplayName string  `json:"displayName" datastore:"displayName"`
	ImageUrl    string  `json:"imageUrl" datastore:"imageUrl"`
	Clues       []*Clue `json:"clues,omitempty" datastore:"clues"`
}

type HuntsList struct {
	Hunts []*Hunt `json:"mapResult" datastore:"mapResult"`
}
