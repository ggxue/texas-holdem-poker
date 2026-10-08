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
	Hand     *hand
}

type view struct {
	Version int64      `json:"version"`
	You     string     `json:"you"`
	Host    string     `json:"host"`
	Seats   [2]*player `json:"seats"`
	Bot     player     `json:"bot"`
	Error   string     `json:"error,omitempty"`
	Hand    *handView  `json:"hand,omitempty"`
}

func (s room) visibleTo(id string) view {
	v := view{Version: s.Version, You: id, Host: s.Host, Bot: s.Bot}
	if s.Hand != nil {
		v.Hand = s.Hand.visibleTo(id)
	}
	for i, occupant := range s.Seats {
		if occupant != "" {
			p := s.Accounts[occupant]
			v.Seats[i] = &p
		}
	}
	return v
}

// clone creates the rollback boundary for one atomic in-memory command.
func (s room) clone() room {
	c := s
	c.Accounts = make(map[string]player, len(s.Accounts))
	for id, p := range s.Accounts {
		c.Accounts[id] = p
	}
	if s.Hand != nil {
		h := *s.Hand
		h.Players = append([]participant(nil), s.Hand.Players...)
		h.Board = append([]Card(nil), s.Hand.Board...)
		c.Hand = &h
	}
	return c
}
