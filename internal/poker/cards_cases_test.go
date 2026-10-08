package poker_test

import (
	"reflect"
	"testing"
	"texas-poker/internal/poker"
)

func strength(t *testing.T, cards ...poker.Card) poker.Strength {
	t.Helper()
	s, e := poker.Evaluate(cards)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func TestAC02FullPointPriority(t *testing.T) {
	cases := [][2][]poker.Card{
		{{{13, 0}, {13, 1}, {14, 0}, {9, 0}, {3, 0}}, {{13, 2}, {13, 3}, {12, 3}, {11, 3}, {8, 3}}},
		{{{8, 0}, {8, 1}, {8, 2}, {14, 0}, {2, 0}}, {{8, 0}, {8, 1}, {8, 3}, {13, 3}, {12, 3}}},
		{{{7, 0}, {7, 1}, {7, 2}, {7, 3}, {14, 0}}, {{7, 0}, {7, 1}, {7, 2}, {7, 3}, {13, 3}}},
		{{{12, 0}, {12, 1}, {3, 0}, {3, 1}, {2, 0}}, {{11, 2}, {11, 3}, {10, 2}, {10, 3}, {14, 3}}},
		{{{8, 0}, {8, 1}, {8, 2}, {2, 0}, {2, 1}}, {{7, 0}, {7, 1}, {7, 2}, {14, 0}, {14, 1}}},
		{{{14, 0}, {12, 0}, {9, 0}, {6, 0}, {3, 0}}, {{14, 3}, {12, 3}, {9, 3}, {6, 3}, {2, 3}}},
		{{{14, 0}, {12, 1}, {9, 0}, {6, 0}, {3, 0}}, {{14, 3}, {12, 2}, {9, 3}, {6, 3}, {2, 3}}},
	}
	for i, pair := range cases {
		if strength(t, pair[0]...).Compare(strength(t, pair[1]...)) <= 0 {
			t.Fatalf("point priority case %d", i)
		}
	}
}
func TestAC03WheelAndNoWraparound(t *testing.T) {
	for _, flush := range []bool{false, true} {
		suit := func(i int) int {
			if flush {
				return 2
			}
			return i % 4
		}
		wheel, six, royal, wrap := []poker.Card{}, []poker.Card{}, []poker.Card{}, []poker.Card{}
		for i, r := range []int{14, 2, 3, 4, 5} {
			wheel = append(wheel, poker.Card{Rank: r, Suit: suit(i)})
		}
		for i, r := range []int{2, 3, 4, 5, 6} {
			six = append(six, poker.Card{Rank: r, Suit: suit(i)})
		}
		for i, r := range []int{10, 11, 12, 13, 14} {
			royal = append(royal, poker.Card{Rank: r, Suit: suit(i)})
		}
		for i, r := range []int{12, 13, 14, 2, 3} {
			wrap = append(wrap, poker.Card{Rank: r, Suit: suit(i)})
		}
		w, s, r, b := strength(t, wheel...), strength(t, six...), strength(t, royal...), strength(t, wrap...)
		if w.Compare(s) >= 0 || r.Compare(s) <= 0 || b.Category == "顺子" || b.Category == "同花顺" || w.Cards[4].Rank != 14 {
			t.Fatalf("A handling: %+v %+v %+v %+v", w, s, r, b)
		}
	}
}
func TestAC04FirstDifferentSuit(t *testing.T) {
	a := strength(t, poker.Card{14, 2}, poker.Card{13, 3}, poker.Card{9, 1}, poker.Card{6, 0}, poker.Card{2, 3})
	b := strength(t, poker.Card{14, 3}, poker.Card{13, 0}, poker.Card{9, 1}, poker.Card{6, 0}, poker.Card{2, 3})
	if b.Compare(a) <= 0 {
		t.Fatal("A suit must decide before K suit")
	}
}
func TestAC05And06BestFiveAndSharedBoardTie(t *testing.T) {
	cases := []struct {
		cards []poker.Card
		want  [5]poker.Card
	}{
		{[]poker.Card{{10, 3}, {11, 3}, {12, 3}, {13, 3}, {14, 3}, {2, 2}, {3, 0}}, [5]poker.Card{{14, 3}, {13, 3}, {12, 3}, {11, 3}, {10, 3}}},
		{[]poker.Card{{14, 3}, {14, 0}, {13, 1}, {12, 2}, {2, 3}, {14, 2}, {3, 0}}, [5]poker.Card{{14, 3}, {14, 2}, {14, 0}, {13, 1}, {12, 2}}},
		{[]poker.Card{{2, 3}, {4, 2}, {7, 0}, {9, 1}, {11, 3}, {14, 2}, {14, 1}}, [5]poker.Card{{14, 2}, {14, 1}, {11, 3}, {9, 1}, {7, 0}}},
		{[]poker.Card{{14, 0}, {14, 3}, {13, 2}, {12, 1}, {11, 0}, {10, 2}, {2, 0}}, [5]poker.Card{{14, 3}, {13, 2}, {12, 1}, {11, 0}, {10, 2}}},
	}
	for i, c := range cases {
		got := strength(t, c.cards...)
		if !reflect.DeepEqual(got.Cards, c.want) {
			t.Fatalf("best five %d: %v want %v", i, got.Cards, c.want)
		}
	}
	board := cases[0].cards[:5]
	a := strength(t, append(append([]poker.Card{}, board...), poker.Card{2, 2}, poker.Card{3, 0})...)
	b := strength(t, append(append([]poker.Card{}, board...), poker.Card{8, 3}, poker.Card{9, 3})...)
	if a.Compare(b) != 0 {
		t.Fatal("unused hole cards broke tie")
	}
}
