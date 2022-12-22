package inventory

import (
	"fmt"
	"sync"

	"github.com/alfreddobradi/game-vslice/common"
	"github.com/alfreddobradi/game-vslice/protobuf"
	intnats "github.com/alfreddobradi/game-vslice/transport/nats"
	"github.com/asynkron/protoactor-go/cluster"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"golang.org/x/exp/slog"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BuildingRegister struct {
	mx *sync.Mutex

	Amount   int
	Building bool
}

type Grain struct {
	ctx cluster.GrainContext

	buildings map[common.Building]*BuildingRegister
}

func (g *Grain) Init(ctx cluster.GrainContext) {
	g.ctx = ctx

	buildings := make(map[common.Building]*BuildingRegister)

	for _, building := range common.Buildings {
		buildings[building] = &BuildingRegister{
			mx:       &sync.Mutex{},
			Amount:   0,
			Building: false,
		}
	}

	g.buildings = buildings
}
func (g *Grain) Terminate(ctx cluster.GrainContext)      {}
func (g *Grain) ReceiveDefault(ctx cluster.GrainContext) {}

func (g *Grain) Start(req *protobuf.StartRequest, ctx cluster.GrainContext) (*protobuf.StartResponse, error) {
	b, ok := common.Buildings[common.BuildingName(req.Name)]
	if !ok {
		return &protobuf.StartResponse{
			Status:    "Error",
			Error:     fmt.Sprintf("Invalid building name: %s", req.Name),
			Timestamp: timestamppb.Now(),
		}, nil
	}

	if g.buildings[b].Building {
		return &protobuf.StartResponse{
			Status:    "Error",
			Error:     fmt.Sprintf("Building is already in progress: %s", req.Name),
			Timestamp: timestamppb.Now(),
		}, nil
	}

	reply := nats.NewInbox()

	slog.Info("requested building", "name", string(b.Name))

	go g.startSubscription(int(req.Amount), reply, b)

	timer := protobuf.GetTimerGrainClient(g.ctx.Cluster(), uuid.New().String())
	timer.CreateTimer(&protobuf.TimerRequest{
		BuildID:   uuid.New().String(),
		Reply:     reply,
		Duration:  "10s",
		Amount:    req.Amount,
		Timestamp: timestamppb.Now(),
	})

	g.buildings[b].Building = true

	return &protobuf.StartResponse{
		Status:    "OK",
		Timestamp: timestamppb.Now(),
	}, nil
}

func (g *Grain) Describe(_ *protobuf.DescribeInventoryRequest, ctx cluster.GrainContext) (*protobuf.DescribeInventoryResponse, error) {
	values := make(map[string]*structpb.Value)

	for building, meta := range g.buildings {
		values[string(building.Name)] = structpb.NewStructValue(&structpb.Struct{
			Fields: map[string]*structpb.Value{
				"amount":   structpb.NewNumberValue(float64(meta.Amount)),
				"building": structpb.NewBoolValue(meta.Building),
			},
		})
	}

	inventory := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"buildings": structpb.NewStructValue(&structpb.Struct{
				Fields: values,
			}),
		},
	}

	return &protobuf.DescribeInventoryResponse{
		Inventory: inventory,
		Timestamp: timestamppb.Now(),
	}, nil
}

func (g *Grain) startSubscription(amount int, reply string, b common.Building) {
	i := 1

	cb := func(t *protobuf.TimerFired) {
		slog.Debug("finished building", "building", b.Name)
		g.buildings[b].Amount += 1
		if amount > 0 && i >= amount {
			g.buildings[b].Building = false
		}
		i++
	}

	sub, _ := intnats.GetConnection().Subscribe(reply, cb)
	if amount > 0 {
		sub.AutoUnsubscribe(amount)
	}
}
