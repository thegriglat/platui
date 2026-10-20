package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/google/renameio/v2"
)

// FileStore keeps values in <Dir>/<key>.json.
type FileStore struct{ Dir string }

var _ Store = (*FileStore)(nil)

// NewFileStore creates a store; the directory is created on the first Save.
func NewFileStore(dir string) *FileStore { return &FileStore{Dir: dir} }

type envelope struct {
	V     int             `json:"v"`
	Saved time.Time       `json:"saved"`
	Data  json.RawMessage `json:"data"`
}

func (s *FileStore) path(key string) string {
	return filepath.Join(s.Dir, filepath.FromSlash(key)+".json")
}

// Load reads a value; a missing file, corruption or a foreign version give ok=false.
func (s *FileStore) Load(key string, v any) (time.Time, bool, error) {
	if err := checkKey(key); err != nil {
		return time.Time{}, false, err
	}
	b, err := os.ReadFile(s.path(key))
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	var e envelope
	if json.Unmarshal(b, &e) != nil || e.V != version || e.Data == nil {
		return time.Time{}, false, nil
	}
	if json.Unmarshal(e.Data, v) != nil {
		return time.Time{}, false, nil
	}
	return e.Saved, true, nil
}

// Save writes a value atomically (renameio: temp+fsync+rename, mode 0600).
func (s *FileStore) Save(key string, v any) error {
	if err := checkKey(key); err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b, err := json.Marshal(envelope{V: version, Saved: time.Now().UTC(), Data: data})
	if err != nil {
		return err
	}
	p := s.path(key)
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return renameio.WriteFile(p, b, 0o600)
}
