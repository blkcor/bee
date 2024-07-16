package beeCache

import pb "github.com/blkcor/beeCache/proto"

type PeerPicker interface {
	PickPeer(key string) (PeerGetter, bool)
}

type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}
