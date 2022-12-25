package dummy

import (
	"github.com/alfreddobradi/game-vslice/persistence/contract"
)

type Persister struct {
}

func (p *Persister) Persist(item contract.Persistable) (int, error) {
	raw, err := item.Encode()
	if err != nil {
		return 0, err
	}

	return len(raw), nil
}

func (p *Persister) Restore(key string) ([]byte, error) {
	// 	raw, err := item.Encode()
	// 	if err != nil {
	// 		return 0, err
	// 	}

	// 	return len(raw), nil
	return nil, nil
}
