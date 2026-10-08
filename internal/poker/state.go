package poker

type player struct {
	ID    string `json:"id"`
	Chips int64  `json:"chips"`
}

type room struct {
	Version  int64             `json:"version"`
	Accounts map[string]player `json:"accounts"`
	Seats    [2]string         `json:"seats"`
	Host     string            `json:"host"`
	Bot      player            `json:"bot"`
}

type view struct {
	Version int64      `json:"version"`
	You     string     `json:"you"`
	Host    string     `json:"host"`
	Seats   [2]*player `json:"seats"`
	Bot     player     `json:"bot"`
	Error   string     `json:"error,omitempty"`
}

func (s room) visibleTo(id string) view {
	v := view{Version: s.Version, You: id, Host: s.Host, Bot: s.Bot}
	for i, occupant := range s.Seats {
		if occupant != "" {
			p := s.Accounts[occupant]
			v.Seats[i] = &p
		}
	}
	return v
}
