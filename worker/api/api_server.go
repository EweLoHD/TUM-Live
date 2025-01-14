package api

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/joschahenningsen/TUM-Live/worker/cfg"
	"github.com/joschahenningsen/TUM-Live/worker/pb"
	"github.com/joschahenningsen/TUM-Live/worker/worker"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedToWorkerServer
}

// RequestCut is a gRPC endpoint for the worker to Cut a video
func (s server) RequestCut(ctx context.Context, request *pb.CutRequest) (*pb.CutResponse, error) {
	return nil, errors.New("not implemented")
}

// RequestWaveform is a gRPC endpoint for the worker to generate a waveform
func (s server) RequestWaveform(ctx context.Context, request *pb.WaveformRequest) (*pb.WaveFormResponse, error) {
	if request.WorkerId != cfg.WorkerID {
		return nil, errors.New("unauthenticated: wrong worker id")
	}
	waveform, err := worker.GetWaveform(request)
	return &pb.WaveFormResponse{Waveform: waveform}, err
}

func (s server) RequestStream(ctx context.Context, request *pb.StreamRequest) (*pb.Status, error) {
	if request.WorkerId != cfg.WorkerID {
		log.Info("Rejected request to stream")
		return &pb.Status{Ok: false}, errors.New("unauthenticated: wrong worker id")
	}
	go worker.HandleStreamRequest(request)
	return &pb.Status{Ok: true}, nil
}

func (s server) RequestPremiere(ctx context.Context, request *pb.PremiereRequest) (*pb.Status, error) {
	if request.WorkerID != cfg.WorkerID {
		log.Info("Rejected request for premiere")
		return &pb.Status{Ok: false}, errors.New("unauthenticated: wrong worker id")
	}
	go worker.HandlePremiere(request)
	return &pb.Status{Ok: true}, nil
}

func (s server) RequestStreamEnd(ctx context.Context, request *pb.EndStreamRequest) (*pb.Status, error) {
	if request.WorkerID != cfg.WorkerID {
		log.Info("Rejected request to end stream")
		return &pb.Status{Ok: false}, errors.New("unauthenticated: wrong worker id")
	}
	go worker.HandleStreamEndRequest(request)
	return &pb.Status{Ok: true}, nil
}

//InitApi Initializes api endpoints
//addr: port to run on, e.g. ":8080"
func InitApi(addr string) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.WithError(err).Fatal("failed to listen")
	}
	grpcServer := grpc.NewServer(grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle:     time.Minute,
		MaxConnectionAge:      time.Minute,
		MaxConnectionAgeGrace: time.Second * 5,
		Time:                  time.Minute * 10,
		Timeout:               time.Second * 20,
	}))
	pb.RegisterToWorkerServer(grpcServer, &server{})

	reflection.Register(grpcServer)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
