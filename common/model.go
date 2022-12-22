package common

import "time"

type BuildingName string

const (
	House BuildingName = "house"
)

var Buildings map[BuildingName]Building = map[BuildingName]Building{
	House: {Name: House, BuildTime: 10 * time.Second},
}

type Building struct {
	Name      BuildingName
	BuildTime time.Duration
}
