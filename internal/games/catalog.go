// Package games is the single registry of supported games and public rules.
package games

import (
	"fmt"
	"github.com/Kirill555dg/mini-games/internal/battleship"
	"github.com/Kirill555dg/mini-games/internal/connectfour"
	"github.com/Kirill555dg/mini-games/internal/dotsboxes"
	"github.com/Kirill555dg/mini-games/internal/engine"
	"github.com/Kirill555dg/mini-games/internal/tictactoe"
	"github.com/Kirill555dg/mini-games/internal/ultimate"
)

var catalog = []engine.Spec{
	{ID: "tictactoe", Title: "Крестики-нолики", Description: "Быстрый раунд 3×3.", Rules: "Ходите по очереди на поле 3×3. Соберите три своих знака подряд по горизонтали, вертикали или диагонали. Заполненное поле без победителя означает ничью. При безошибочной игре ничья неизбежна: это короткий классический режим. Для более глубокой тактики выберите «Девять полей».", MinPlayers: 2, MaxPlayers: 2, Sizes: []int{tictactoe.Size}},
	{ID: "ultimate", Title: "Девять полей", Description: "Управляйте следующим ходом соперника.", Rules: "Девять полей 3×3 образуют одно большое поле. Первый ход — в любую клетку. Положение выбранной клетки внутри малого поля определяет, в какое малое поле должен ходить соперник. Три своих знака в ряд выигрывают малое поле. Выигранное или заполненное поле закрывается; если предыдущий ход отправляет туда, можно выбрать любое открытое поле. Ничейное малое поле не принадлежит никому. Выиграйте три малых поля в ряд по горизонтали, вертикали или диагонали. Когда все малые поля закрыты без такой линии, партия заканчивается ничьей.", MinPlayers: ultimate.Players, MaxPlayers: ultimate.Players},
	{ID: "battleship", Title: "Морской бой", Description: "Потопите флот соперника.", Rules: "Классическое поле 10×10: один четырёхпалубный, два трёхпалубных, три двухпалубных и четыре однопалубных корабля. Быстрый режим 8×8: корабли длиной 4, 3, 2, 2 и 1. Корабли не могут касаться даже углами. Перетаскивайте корабли, поворачивайте или расставьте флот случайно. Нажмите «Флот готов», когда всё расставлено. Попадание даёт ещё один выстрел, промах передаёт ход. Потопленный корабль отмечается целиком; вода вокруг него исключается из выстрелов. Побеждает первый, кто поразит все корабли. Зрители видят только публичные результаты выстрелов.", MinPlayers: 2, MaxPlayers: 2, Sizes: battleship.Sizes()},
	{ID: "connectfour", Title: "Четыре в ряд", Description: "Соберите четыре фишки в ряд.", Rules: "На поле 7×6 фишка падает в нижнюю свободную клетку выбранной колонки. Соберите четыре подряд по горизонтали, вертикали или диагонали. Заполненное поле без победителя означает ничью.", MinPlayers: 2, MaxPlayers: 2},
	{ID: "dotsboxes", Title: "Точки и квадраты", Description: "Квадраты для компании.", Rules: "Выберите от 2 до 4 игроков. Нажимайте на свободные отрезки между точками. Замкнув квадрат, вы получаете очко и дополнительный ход. Когда все отрезки заняты, побеждает игрок с наибольшим счётом. Равный лучший счёт означает ничью.", MinPlayers: 2, MaxPlayers: 4},
}

func init() {
	english := map[string]engine.Translation{
		"tictactoe":   {Title: "Tic-tac-toe", Description: "A quick 3×3 round.", Rules: "Take turns on a 3×3 board. Make three marks in an unbroken horizontal, vertical or diagonal line. A full board without a winner is a draw. Perfect play leads to a draw: this is the short classic mode. Try Nine boards for deeper tactics."},
		"ultimate":    {Title: "Nine boards", Description: "Choose your opponent's next board.", Rules: "Nine 3×3 boards form a larger board. The first move can be anywhere. Your cell's position within its small board sends the opponent to the matching small board. Three marks in a row claim a small board. A won or full board closes; if sent there, choose any open board. A drawn board belongs to neither player. Win three small boards in a row horizontally, vertically or diagonally. If all small boards close without such a line, the game is a draw."},
		"battleship":  {Title: "Sea battle", Description: "Sink the opposing fleet.", Rules: "Classic mode uses a 10×10 board: one four-deck, two three-deck, three two-deck and four single-deck ships. Quick mode is 8×8 with ships of lengths 4, 3, 2, 2 and 1. Ships cannot touch, even diagonally. Drag and rotate ships, or randomize the fleet. Click Fleet ready when placement is complete. A hit grants another shot; a miss passes the turn. Sunk ships are marked in full; the surrounding water is excluded from targeting. Sink the entire opposing fleet to win. Spectators see only public shot results."},
		"connectfour": {Title: "Four in a row", Description: "Connect four discs.", Rules: "On a 7×6 board each disc falls to the lowest empty cell in its column. Connect four horizontally, vertically or diagonally. A full board without a winner is a draw."},
		"dotsboxes":   {Title: "Dots and boxes", Description: "Close boxes with friends.", Rules: "Choose two to four players. Select a free edge between two dots. Closing a box earns one point and another turn. When all edges are filled, the highest score wins. A shared highest score is a draw."},
	}
	for i := range catalog {
		c := &catalog[i]
		c.Translations = map[string]engine.Translation{"en": english[c.ID]}
		c.Sides = map[string][]string{"ru": {"Фиолетовые", "Оранжевые", "Зелёные", "Розовые"}, "en": {"Purple", "Orange", "Green", "Pink"}}
		if c.ID == "tictactoe" || c.ID == "ultimate" {
			c.Sides = map[string][]string{"ru": {"Крестики (X)", "Нолики (O)"}, "en": {"Crosses (X)", "Noughts (O)"}}
		}
		if c.ID == "battleship" {
			c.Sides = map[string][]string{"ru": {"Первый ход", "Второй ход"}, "en": {"First turn", "Second turn"}}
		}
	}
}

func Catalog() []engine.Spec {
	out := append([]engine.Spec(nil), catalog...)
	for i := range out {
		out[i].Sizes = append([]int(nil), out[i].Sizes...)
	}
	return out
}
func Find(id string) (engine.Spec, bool) {
	for _, s := range Catalog() {
		if s.ID == id {
			return s, true
		}
	}
	return engine.Spec{}, false
}
func New(id string, size, players int) (engine.Game, error) {
	s, ok := Find(id)
	if !ok || players < s.MinPlayers || players > s.MaxPlayers {
		return nil, fmt.Errorf("Выберите игру и допустимое число игроков.")
	}
	if size == 0 {
		size = s.DefaultSize()
	}
	if len(s.Sizes) > 0 {
		found := false
		for _, n := range s.Sizes {
			found = found || n == size
		}
		if !found {
			return nil, fmt.Errorf("Выберите доступный размер поля.")
		}
	}
	switch id {
	case "tictactoe":
		return tictactoe.New(), nil
	case "ultimate":
		return ultimate.New(), nil
	case "battleship":
		return battleship.New(size), nil
	case "connectfour":
		return connectfour.New(), nil
	case "dotsboxes":
		return dotsboxes.New(players), nil
	}
	return nil, engine.ErrMove
}
