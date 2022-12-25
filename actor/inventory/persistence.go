package inventory

import (
	"bytes"
	"encoding/gob"

	"github.com/alfreddobradi/game-vslice/common"
	"github.com/alfreddobradi/game-vslice/protobuf"
	"github.com/asynkron/protoactor-go/cluster"
)

func (g *Grain) Encode() ([]byte, error) {
	encode := make(map[string]interface{})
	data := make(map[string]interface{})

	data["buildings"] = g.buildings
	encode["data"] = data
	encode["identity"] = g.ctx.Identity()

	buf := bytes.NewBuffer([]byte(""))
	encoder := gob.NewEncoder(buf)

	if err := encoder.Encode(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (g *Grain) Decode(b []byte) error {
	m := make(map[string]interface{})

	buf := bytes.NewBuffer(b)
	decoder := gob.NewDecoder(buf)

	if err := decoder.Decode(&m); err != nil {
		return err
	}

	g.buildings = m["buildings"].(map[common.Building]*BuildingRegister)

	return nil
}

func (g *Grain) Kind() string {
	return "inventory"
}

func (g *Grain) Identity() string {
	return g.ctx.Identity()
}

func (g *Grain) Restore(req *protobuf.RestoreRequest, ctx cluster.GrainContext) (*protobuf.RestoreResponse, error) {
	if err := g.Decode(req.Data); err != nil {
		return &protobuf.RestoreResponse{
			Status: protobuf.Status_Error,
			Error:  err.Error(),
		}, nil
	}

	return &protobuf.RestoreResponse{
		Status: protobuf.Status_OK,
	}, nil
}

func init() {
	m := make(map[common.Building]*BuildingRegister)
	gob.Register(m)
}
