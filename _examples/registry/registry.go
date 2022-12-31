package main

import (
	"github.com/alfreddobradi/game-vslice/blueprints"
	"github.com/alfreddobradi/game-vslice/blueprints/registry"
	"github.com/alfreddobradi/game-vslice/common"
)

func main() {
	buildingName := "house"
	id := common.GetBuildingID(buildingName)
	i := &blueprints.Building{
		ID:   id,
		Name: buildingName,
		Cost: map[string]int64{
			"wood": 100,
		},
		Generates: map[string]int64{
			"pops": 1,
		},
		BuildTime: "10s",
	}

	if err := registry.Push("building", i, true); err != nil {
		panic(err)
	}

}
