package poker

import "testing"

func TestAC01RoyalFlush(t *testing.T) {
	s, err := Evaluate([]Card{{10, 3}, {11, 3}, {12, 3}, {13, 3}, {14, 3}})
	if err != nil {
		t.Fatal(err)
	}
	if s.Category != "同花大顺" {
		t.Fatalf("category = %s", s.Category)
	}
}

func TestAC01TenCategoriesInOrder(t *testing.T) {
	cases := []struct {
		name  string
		cards []Card
	}{
		{"同花大顺", []Card{{10, 3}, {11, 3}, {12, 3}, {13, 3}, {14, 3}}},
		{"同花顺", []Card{{5, 2}, {6, 2}, {7, 2}, {8, 2}, {9, 2}}},
		{"四条", []Card{{13, 3}, {13, 2}, {13, 1}, {13, 0}, {2, 3}}},
		{"葫芦", []Card{{12, 3}, {12, 2}, {12, 1}, {5, 3}, {5, 2}}},
		{"同花", []Card{{14, 1}, {11, 1}, {8, 1}, {4, 1}, {2, 1}}},
		{"顺子", []Card{{5, 3}, {6, 2}, {7, 1}, {8, 0}, {9, 3}}},
		{"三条", []Card{{8, 3}, {8, 2}, {8, 1}, {13, 0}, {2, 3}}},
		{"两对", []Card{{11, 3}, {11, 2}, {4, 3}, {4, 0}, {14, 2}}},
		{"一对", []Card{{13, 3}, {13, 2}, {14, 3}, {9, 0}, {3, 1}}},
		{"高牌", []Card{{14, 3}, {13, 2}, {9, 1}, {6, 0}, {2, 3}}},
	}
	var previous Strength
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := Evaluate(c.cards)
			if err != nil {
				t.Fatal(err)
			}
			if s.Category != c.name {
				t.Fatalf("got %s", s.Category)
			}
			if i > 0 && previous.Compare(s) <= 0 {
				t.Fatal("category order incorrect")
			}
			previous = s
		})
	}
}
