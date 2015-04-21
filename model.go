package hunt

type Tag struct {
	Id string `json:"id"`
}

type Question struct {
	Question      string   `json:"question"`
	Answers       []string `json:"answers" datastore:"-"`
	CorrectAnswer int      `json:"correctAnswer"`
	WrongMsg      string   `json:"wrongMessage"`
	RightMsg      string   `json:"rightMessage"`
}

type Clue struct {
	Id           string     `json:"id"`
	Type         string     `json:"type"`
	ShuffleGroup int        `json:"shufflegroup"`
	DisplayName  string     `json:"displayName"`
	DisplayText  string     `json:"displayText"`
	DisplayImage string     `json:"displayImage"`
	Tags         []Tag      `json:"tags" datastore:"-"`
	Questions    []Question `json:"question" datastore:"-"`
}

type Hunt struct {
	Key         string `json:"-" datastore:"-"`
	Id          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"displayName"`
	ImageUrl    string `json:"imageUrl"`
	Clues       []Clue `json:"clues" datastore:"-"`
}
