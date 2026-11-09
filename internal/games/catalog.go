// Package games is the single registry of supported games and public rules.
package games

import (
	"fmt"
	"github.com/Kirill555dg/mini-games/internal/battleship"
	"github.com/Kirill555dg/mini-games/internal/connectfour"
	"github.com/Kirill555dg/mini-games/internal/dotsboxes"
	"github.com/Kirill555dg/mini-games/internal/engine"
	"github.com/Kirill555dg/mini-games/internal/tictactoe"
)

var catalog = []engine.Spec{
	{ID: "tictactoe", Title: "Крестики-нолики", Description: "Классическая дуэль на поле от 3×3 до 7×7.", Rules: "Ходите по очереди. Соберите непрерывный ряд по горизонтали, вертикали или диагонали: три на поле 3×3, четыре на 4×4 и пять на больших полях. Заполненное поле без победителя означает ничью.", MinPlayers: 2, MaxPlayers: 2, Sizes: []int{3, 4, 5, 6, 7}},
	{ID: "battleship", Title: "Морской бой", Description: "Спрячьте флот, найдите корабли соперника.", Rules: "Поле 8×8. Разместите корабли длиной 4, 3, 2, 2 и 1; они не могут касаться даже углами. Можно расставить флот случайно. Нажмите «Готов», когда всё расставлено. Попадание даёт ещё один выстрел, промах передаёт ход. Побеждает первый, кто поразит все корабли. Зрители видят только выстрелы.", MinPlayers: 2, MaxPlayers: 2},
	{ID: "connectfour", Title: "Четыре в ряд", Description: "Опускайте фишки в колонку и стройте четвёрку.", Rules: "На поле 7×6 фишка падает в нижнюю свободную клетку выбранной колонки. Соберите четыре подряд по горизонтали, вертикали или диагонали. Заполненное поле без победителя означает ничью.", MinPlayers: 2, MaxPlayers: 2},
	{ID: "dotsboxes", Title: "Точки и квадраты", Description: "Замыкайте квадраты. Для компании до четырёх.", Rules: "Выберите от 2 до 4 игроков. Нажимайте на свободные отрезки между точками. Замкнув квадрат, вы получаете очко и дополнительный ход. Когда все отрезки заняты, побеждает игрок с наибольшим счётом. Равный лучший счёт означает ничью.", MinPlayers: 2, MaxPlayers: 4},
}

func init() {
	english := map[string]engine.Translation{
		"tictactoe":   {Title: "Tic-tac-toe", Description: "A classic duel, from 3×3 to 7×7.", Rules: "Take turns placing marks. Make an unbroken horizontal, vertical or diagonal line: three on 3×3, four on 4×4, and five on larger boards. A full board without a winning line is a draw."},
		"battleship":  {Title: "Sea battle", Description: "Hide your fleet. Find your opponent's ships.", Rules: "The board is 8×8. Place five ships of lengths 4, 3, 2, 2 and 1. Ships cannot touch, even diagonally. You can randomize the fleet. Click Ready when placement is complete. A hit grants another shot; a miss passes the turn. Sink the entire opposing fleet to win. Spectators see shots only."},
		"connectfour": {Title: "Four in a row", Description: "Drop your discs and connect four.", Rules: "On a 7×6 board each disc falls to the lowest empty cell in its column. Connect four horizontally, vertically or diagonally. A full board without a winner is a draw."},
		"dotsboxes":   {Title: "Dots and boxes", Description: "Close the boxes. Bring up to four friends.", Rules: "Choose two to four players. Select a free edge between two dots. Closing a box earns one point and another turn. When all edges are filled, the highest score wins. A shared highest score is a draw."},
	}
	for i := range catalog {
		c := &catalog[i]
		c.Translations = map[string]engine.Translation{"en": english[c.ID]}
		c.Sides = map[string][]string{"ru": {"Фиолетовые", "Оранжевые", "Зелёные", "Розовые"}, "en": {"Purple", "Orange", "Green", "Pink"}}
		if c.ID == "tictactoe" {
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
		return tictactoe.New(size), nil
	case "battleship":
		return battleship.New(), nil
	case "connectfour":
		return connectfour.New(), nil
	case "dotsboxes":
		return dotsboxes.New(players), nil
	}
	return nil, engine.ErrMove
}
