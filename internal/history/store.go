// Package history stores completed public match records with atomic replacement.
package history

import (
	"encoding/json"
	"errors"
	"github.com/Kirill555dg/mini-games/internal/engine"
	"github.com/Kirill555dg/mini-games/internal/games"
	"io"
	"os"
	"path/filepath"
	"time"
)

const Limit = 100
const MaxBytes = 1024 * 1024

type Record struct {
	ID         uint64       `json:"id"`
	FinishedAt time.Time    `json:"finishedAt"`
	Code       string       `json:"code"`
	Game       string       `json:"game"`
	Size       int          `json:"size"`
	Players    []string     `json:"players"`
	Winner     int          `json:"winner"`
	Reason     string       `json:"reason,omitempty"`
	State      engine.State `json:"state"`
}
type file struct {
	Version int      `json:"version"`
	Serial  uint64   `json:"serial"`
	Games   []Record `json:"games"`
}
type Store struct {
	path    string
	serial  uint64
	records []Record
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, records: []Record{}}
	if path == "" {
		return s, nil
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, errors.New("cannot open match history")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if err != nil || len(data) > MaxBytes {
		return nil, errors.New("match history exceeds limit")
	}
	var version struct {
		Version int `json:"version"`
	}
	if json.Unmarshal(data, &version) != nil {
		return nil, errors.New("invalid match history; preserve existing file")
	}
	var saved file
	switch version.Version {
	case 1:
		saved, err = migrate(data)
	case 2:
		err = json.Unmarshal(data, &saved)
	default:
		err = errors.New("unsupported history version")
	}
	if err != nil || len(saved.Games) > Limit {
		return nil, errors.New("invalid match history; preserve existing file")
	}
	var previous uint64
	for _, r := range saved.Games {
		spec, ok := games.Find(r.Game)
		if !ok || r.ID <= previous || r.ID > saved.Serial || len(r.Players) < spec.MinPlayers || len(r.Players) > spec.MaxPlayers || r.Winner < engine.Draw || r.Winner == engine.NoWinner || r.Winner >= len(r.Players) || r.State.Phase != engine.Finished || r.State.Winner != r.Winner || len(r.State.Moves) > 256 {
			return nil, errors.New("invalid saved match")
		}
		for _, name := range r.Players {
			if name == "" || len([]rune(name)) > 24 {
				return nil, errors.New("invalid saved player")
			}
		}
		previous = r.ID
	}
	s.records = saved.Games
	s.serial = saved.Serial
	return s, nil
}

// List returns detached records: callers may serialize them without holding a room lock.
func (s *Store) List() []Record {
	out := append([]Record{}, s.records...)
	for i := range out {
		out[i].Players = append([]string(nil), out[i].Players...)
		out[i].State = engine.Copy(out[i].State)
	}
	return out
}

func (s *Store) Append(r Record) error {
	r.ID = s.serial + 1
	all := append(s.List(), r)
	if len(all) > Limit {
		all = all[len(all)-Limit:]
	}
	var data []byte
	var err error
	for {
		data, err = json.Marshal(file{2, r.ID, all})
		if err != nil {
			return err
		}
		if len(data) <= MaxBytes {
			break
		}
		if len(all) <= 1 {
			return errors.New("match exceeds history limit")
		}
		all = all[1:]
	}
	if s.path != "" {
		f, err := os.CreateTemp(filepath.Dir(s.path), ".stats-*")
		if err != nil {
			return errors.New("cannot create match history")
		}
		name := f.Name()
		defer os.Remove(name)
		_, err = f.Write(data)
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			return errors.New("cannot write match history")
		}
		if os.Rename(name, s.path) != nil {
			return errors.New("cannot replace match history")
		}
	}
	s.records = all
	s.serial = r.ID
	return nil
}

func migrate(data []byte) (file, error) {
	var old struct {
		Version int    `json:"version"`
		Serial  uint64 `json:"serial"`
		Games   []struct {
			ID         uint64    `json:"id"`
			FinishedAt time.Time `json:"finishedAt"`
			Code       string    `json:"code"`
			Size       int       `json:"size"`
			X          string    `json:"x"`
			O          string    `json:"o"`
			Result     string    `json:"result"`
			Reason     string    `json:"reason"`
			Moves      []struct {
				Mark string `json:"mark"`
				Cell int    `json:"cell"`
			} `json:"moves"`
		} `json:"games"`
	}
	if err := json.Unmarshal(data, &old); err != nil {
		return file{}, err
	}
	out := file{Version: 2, Serial: old.Serial, Games: []Record{}}
	for _, r := range old.Games {
		if r.Size == 0 {
			r.Size = 3
		}
		g, err := games.New("tictactoe", r.Size, 2)
		if err != nil {
			return file{}, err
		}
		for i, m := range r.Moves {
			mark := "X"
			if i%2 == 1 {
				mark = "O"
			}
			if m.Mark != mark || g.Apply(i%2, engine.Action{Kind: "move", Cell: m.Cell}) != nil {
				return file{}, errors.New("invalid legacy moves")
			}
		}
		winner := engine.Draw
		switch r.Result {
		case "X":
			winner = 0
		case "O":
			winner = 1
		case "draw", "abandoned":
		default:
			return file{}, errors.New("invalid legacy result")
		}
		state := g.View(-1)
		state.Phase = engine.Finished
		state.Winner = winner
		out.Games = append(out.Games, Record{ID: r.ID, FinishedAt: r.FinishedAt, Code: r.Code, Game: "tictactoe", Size: r.Size, Players: []string{r.X, r.O}, Winner: winner, Reason: r.Reason, State: state})
	}
	return out, nil
}
