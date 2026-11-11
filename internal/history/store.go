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
const fileVersion = 3

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
	// Match history is disposable; incompatible older formats start empty.
	// Availability observations use a separate store and are never reset here.
	if version.Version > 0 && version.Version < fileVersion {
		return s, nil
	}
	var saved file
	if version.Version != fileVersion || json.Unmarshal(data, &saved) != nil || len(saved.Games) > Limit {
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
		data, err = json.Marshal(file{fileVersion, r.ID, all})
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
