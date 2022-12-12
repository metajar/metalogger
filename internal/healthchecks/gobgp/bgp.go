package gobgp

import (
	"context"
	"fmt"
	"github.com/metajar/metalogger/internal/logger"
	api "github.com/osrg/gobgp/v3/api"
	apipb "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/log"
	"github.com/osrg/gobgp/v3/pkg/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	apb "google.golang.org/protobuf/types/known/anypb"
)

func New(opts ...Option) *BgpAnycast {
	bgpAny := &BgpAnycast{}
	for _, opt := range opts {
		opt(bgpAny)
	}
	return bgpAny
}

type BgpAnycast struct {
	client          apipb.GobgpApiClient
	routerId        string
	routerASN       uint32
	neighborAddress string
	neighborASN     uint32
	anyCastPrefix   *apipb.IPAddressPrefix
	bgpMultihop     bool
	ttlHops         uint32
}

type Option func(anycast *BgpAnycast)

func RouterID(addr string) Option {
	return func(b *BgpAnycast) {
		b.routerId = addr
	}
}

func RouterASN(asn uint32) Option {
	return func(b *BgpAnycast) {
		b.routerASN = asn
	}
}

func NeighborAddress(addr string) Option {
	return func(b *BgpAnycast) {
		b.neighborAddress = addr
	}
}

func NeighborASN(asn uint32) Option {
	return func(b *BgpAnycast) {
		b.neighborASN = asn
	}
}
func AnnouncePrefix(prefix *apipb.IPAddressPrefix) Option {
	return func(b *BgpAnycast) {
		b.anyCastPrefix = prefix
	}
}

func WithEbgpMulti(hops uint32) Option {
	return func(b *BgpAnycast) {
		b.bgpMultihop = true
		b.ttlHops = hops
	}
}

func (b *BgpAnycast) Init() {
	s := server.NewBgpServer(
		server.GrpcListenAddress("127.0.0.1:57777"),
		server.LoggerOption(&myLogger{logger: logger.SugarLogger}))
	go s.Serve()
	// global configuration
	if err := s.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: &api.Global{
			Asn:        b.routerASN,
			RouterId:   b.routerId,
			ListenPort: -1, // gobgp won't listen on tcp:179
		},
	}); err != nil {
		logger.SugarLogger.Fatalln(err)
	}
	// monitor the change of the peer state
	if err := s.WatchEvent(context.Background(), &api.WatchEventRequest{Peer: &api.WatchEventRequest_Peer{}}, func(r *api.WatchEventResponse) {
		if p := r.GetPeer(); p != nil && p.Type == api.WatchEventResponse_PeerEvent_STATE {
			logger.SugarLogger.Info(p)
		}
	}); err != nil {
		logger.SugarLogger.Fatalln(err)
	}

	// neighbor configuration
	n := &api.Peer{
		Conf: &api.PeerConf{
			NeighborAddress: b.neighborAddress,
			PeerAsn:         b.neighborASN,
		},
	}
	if b.bgpMultihop {
		if b.ttlHops == 0 {
			logger.SugarLogger.Infow("ttlHops not set setting to default", "default", 10)
			b.ttlHops = 10
		}
		n.EbgpMultihop = &apipb.EbgpMultihop{
			Enabled:     true,
			MultihopTtl: b.ttlHops,
		}
	}

	// add the peer to the bgp speaker
	if err := s.AddPeer(context.Background(), &api.AddPeerRequest{
		Peer: n,
	}); err != nil {
		logger.SugarLogger.Fatalln(err)
	}
	conn, err := grpc.DialContext(context.TODO(), "127.0.0.1:57777", grpc.WithInsecure())
	if err != nil {
		logger.SugarLogger.Fatalf("failed to connect to gobgp with error: %+v\n", err)
	}
	b.client = apipb.NewGobgpApiClient(conn)
}

func (b *BgpAnycast) Check() bool {
	return true
}

func (b *BgpAnycast) Success() {
	logger.SugarLogger.Infow("Sending BGP Updates", "prefix", b.anyCastPrefix.GetPrefix())
	nlri, _ := apb.New(b.anyCastPrefix)
	family := &apipb.Family{
		Afi:  apipb.Family_AFI_IP,
		Safi: apipb.Family_SAFI_UNICAST,
	}
	a1, _ := apb.New(&apipb.OriginAttribute{
		Origin: 0,
	})
	a2, _ := apb.New(&apipb.NextHopAttribute{
		NextHop: "172.31.255.199",
	})
	attrs := []*apb.Any{a1, a2}
	_, err := b.client.AddPath(context.Background(), &apipb.AddPathRequest{
		TableType: apipb.TableType_GLOBAL,
		Path: &apipb.Path{
			Family: family,
			Nlri:   nlri,
			Pattrs: attrs,
		},
	})
	if err != nil {
		logger.SugarLogger.Fatalln(err)
	}
}

func (b *BgpAnycast) Failure() {
	fmt.Println("Failure on check. We should probably withdraw later if this is more important")
}

// implement github.com/osrg/gobgp/v3/pkg/log/Logger interface
type myLogger struct {
	logger *zap.SugaredLogger
}

func (l *myLogger) Panic(msg string, fields log.Fields) {
	l.logger.Panicw(msg, fields)
}

func (l *myLogger) Fatal(msg string, fields log.Fields) {
	l.logger.Fatalw(msg, fields)
}

func (l *myLogger) Error(msg string, fields log.Fields) {
	l.logger.Errorw(msg, fields)
}

func (l *myLogger) Warn(msg string, fields log.Fields) {
	l.logger.Warnw(msg, fields)
}

func (l *myLogger) Info(msg string, fields log.Fields) {
	l.logger.Infow(msg, fields)
}

func (l *myLogger) Debug(msg string, fields log.Fields) {
	l.logger.Debugw(msg, fields)
}

func (l *myLogger) SetLevel(level log.LogLevel) {
	logr := l.logger.Level()
	switch level {
	case PanicLevel:
		err := logr.Set("panic")
		if err != nil {
			fmt.Println(err)
		}
	case FatalLevel:
		err := logr.Set("fatal")
		if err != nil {
			fmt.Println(err)
		}
	case ErrorLevel:
		err := logr.Set("error")
		if err != nil {
			fmt.Println(err)
		}
	case WarnLevel:
		err := logr.Set("warn")
		if err != nil {
			fmt.Println(err)
		}
	case InfoLevel:
		err := logr.Set("info")
		if err != nil {
			fmt.Println(err)
		}
	case DebugLevel:
		err := logr.Set("debug")
		if err != nil {
			fmt.Println(err)
		}
	case TraceLevel:
		err := logr.Set("debug")
		if err != nil {
			fmt.Println(err)
		}
	}
}

func (l *myLogger) GetLevel() log.LogLevel {
	logr := l.logger.Level()
	return log.LogLevel(logr.Get().(int8))
}

const (
	PanicLevel log.LogLevel = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	TraceLevel
)
