package persistence

import (
	"github.com/alfreddobradi/game-vslice/persistence/contract"
	"github.com/alfreddobradi/game-vslice/persistence/postgres"
	"github.com/asynkron/protoactor-go/cluster"
)

var persister contract.PersisterRestorer

func Create(c *cluster.Cluster) {
	if persister == nil {
		persister = postgres.NewPersister(c)
	}
}

func Get() contract.PersisterRestorer {
	return persister
}
