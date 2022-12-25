package inventory

import (
	"fmt"
	"sync"
	"time"

	"github.com/alfreddobradi/game-vslice/common"
	"github.com/alfreddobradi/game-vslice/persistence"
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
	Queue    int
	Finished time.Time
}

type Grain struct {
	ctx cluster.GrainContext

	buildings    map[common.Building]*BuildingRegister
	replySubject string
	subscription *nats.Subscription
}

func (g *Grain) Init(ctx cluster.GrainContext) {
	g.ctx = ctx
	label := fmt.Sprintf("%s-subject", ctx.Identity())
	g.replySubject = uuid.NewSHA1(uuid.NameSpaceOID, []byte(label)).String()

	buildings := make(map[common.Building]*BuildingRegister)

	for _, building := range common.Buildings {
		buildings[building] = &BuildingRegister{
			mx:     &sync.Mutex{},
			Amount: 0,
			Queue:  0,
		}
	}

	g.buildings = buildings

	cb := func(t *protobuf.TimerFired) {
		payload := t.Data.AsMap()
		buildingName := payload["building"].(string)
		building, ok := common.Buildings[common.BuildingName(buildingName)]
		if !ok {
			slog.Error("failed to complete building", fmt.Errorf("Invalid building name: %s", buildingName))
			return
		}
		slog.Debug("finished building", "building", building.Name)
		g.buildings[building].Amount += 1
		g.buildings[building].Queue -= 1
	}

	sub, err := intnats.GetConnection().Subscribe(g.replySubject, cb)
	if err != nil {
		slog.Error("failed to subscribe to reply subject", err)
		return
	}

	g.subscription = sub
}
func (g *Grain) Terminate(ctx cluster.GrainContext) {
	if n, err := persistence.Get().Persist(g); err != nil {
		slog.Error("failed to persist grain", err, "kind", g.Kind(), "identity", ctx.Identity())
	} else {
		slog.Debug("grain successfully persisted", "kind", g.Kind(), "identity", ctx.Identity(), "written", n)
	}

	g.subscription.Unsubscribe()
}

func (g *Grain) ReceiveDefault(ctx cluster.GrainContext) {}

func (g *Grain) Start(req *protobuf.StartRequest, ctx cluster.GrainContext) (*protobuf.StartResponse, error) {
	b, ok := common.Buildings[common.BuildingName(req.Name)]
	if !ok {
		return &protobuf.StartResponse{
			Status:    protobuf.Status_Error,
			Error:     fmt.Sprintf("Invalid building name: %s", req.Name),
			Timestamp: timestamppb.Now(),
		}, nil
	}

	if g.buildings[b].Queue > 0 {
		return &protobuf.StartResponse{
			Status:    protobuf.Status_Error,
			Error:     fmt.Sprintf("Building is already in progress: %s", req.Name),
			Timestamp: timestamppb.Now(),
		}, nil
	}

	slog.Info("requested building", "name", string(b.Name))

	timer := protobuf.GetTimerGrainClient(g.ctx.Cluster(), uuid.New().String())
	timer.CreateTimer(&protobuf.TimerRequest{
		BuildID:  uuid.New().String(),
		Reply:    g.replySubject,
		Duration: b.BuildTime,
		Amount:   req.Amount,
		Data: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"building": structpb.NewStringValue(string(b.Name)),
			},
		},
		Timestamp: timestamppb.Now(),
	})

	g.buildings[b].Queue += int(req.Amount)
	d, _ := time.ParseDuration(b.BuildTime)
	start := time.Now()
	for r := req.Amount; r > 0; r-- {
		start = start.Add(d)
	}
	g.buildings[b].Finished = start

	return &protobuf.StartResponse{
		Status:    protobuf.Status_OK,
		Timestamp: timestamppb.Now(),
	}, nil
}

func (g *Grain) Describe(_ *protobuf.DescribeInventoryRequest, ctx cluster.GrainContext) (*protobuf.DescribeInventoryResponse, error) {
	values := make(map[string]*structpb.Value)

	for building, meta := range g.buildings {
		values[string(building.Name)] = structpb.NewStructValue(&structpb.Struct{
			Fields: map[string]*structpb.Value{
				"amount": structpb.NewNumberValue(float64(meta.Amount)),
				"queue":  structpb.NewNumberValue(float64(meta.Queue)),
				"finish": structpb.NewStringValue(meta.Finished.Format(time.RFC3339)),
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
