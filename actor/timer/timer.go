package timer

import (
	"time"

	"github.com/alfreddobradi/game-vslice/protobuf"
	"github.com/alfreddobradi/game-vslice/transport/nats"
	"github.com/asynkron/protoactor-go/cluster"
	"golang.org/x/exp/slog"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Grain struct{}

func (g *Grain) Init(ctx cluster.GrainContext)           {}
func (g *Grain) Terminate(ctx cluster.GrainContext)      {}
func (g *Grain) ReceiveDefault(ctx cluster.GrainContext) {}

func (g *Grain) CreateTimer(req *protobuf.TimerRequest, ctx cluster.GrainContext) (*protobuf.TimerResponse, error) {
	d, err := time.ParseDuration(req.Duration)
	if err != nil {
		return &protobuf.TimerResponse{
			Status:    "Error",
			Error:     err.Error(),
			Timestamp: timestamppb.Now(),
		}, nil
	}

	deadline := req.Timestamp.AsTime().Add(d)

	slog.Info("starting timer", "trace_id", req.TraceID, "deadline", deadline)
	go g.startTimer(req.Amount, req.Reply, time.Until(deadline))

	return &protobuf.TimerResponse{
		Status:    "OK",
		Deadline:  timestamppb.New(deadline),
		Timestamp: timestamppb.Now(),
	}, nil
}

func (g *Grain) startTimer(amount int64, reply string, dur time.Duration) {
	t := time.NewTicker(dur)
	conn := nats.GetConnection()

	i := int64(1)
	for curTime := range t.C {
		slog.Debug("timer fired", "reply", reply)
		if err := conn.Publish(reply, &protobuf.TimerFired{
			Timestamp: timestamppb.New(curTime),
		}); err != nil {
			slog.Error("failed to send TimerFired message", err)
		}

		if amount > 0 && i >= amount {
			t.Stop()
		}
		i++
	}
}
