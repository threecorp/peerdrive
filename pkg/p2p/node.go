package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/threecorp/peerdrive/pkg/dev"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/ipfs/go-datastore"

	badger "github.com/ipfs/go-ds-badger"
	crdt "github.com/ipfs/go-ds-crdt"

	ipfslite "github.com/hsanjuan/ipfs-lite"

	"github.com/multiformats/go-multiaddr"
	"github.com/samber/lo"
	"go.uber.org/multierr"
)

// Peers

var (
	defaultBootstrapPeers     = dht.DefaultBootstrapPeers
	defaultBootstrapPeersInfo []peer.AddrInfo
)

func init() {
	maddr, err := multiaddr.NewMultiaddr(
		"/ip4/104.131.131.82/udp/4001/quic/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	)
	if err != nil {
		panic(err)
	}
	defaultBootstrapPeers = append(defaultBootstrapPeers, maddr)

	infos, err := peer.AddrInfosFromP2pAddrs(defaultBootstrapPeers...)
	if err != nil {
		panic(err)
	}
	defaultBootstrapPeersInfo = infos
}

type PeerList []peer.ID

func (pl *PeerList) AppendUnique(ids ...peer.ID) bool {
	prevs := len(*pl)
	*pl = lo.Uniq(append(*pl, ids...))
	return prevs != len(*pl)
}

var Peers = PeerList{}

// Datastore arranges to other folder

const (
	DSName = dev.DatastoreName
)

var (
	DSKey = datastore.NewKey(DSName)
)

type Node struct {
	Host       host.Host
	Lite       *ipfslite.Peer
	DHT        *dual.DHT       // routing.Routing
	DS         *crdt.Datastore // datastore.Batching
	DSPutCh    chan lo.Tuple2[datastore.Key, []byte]
	DSDelCh    chan datastore.Key
	Rendezvous string
}

func (n *Node) Close() error {
	return multierr.Combine(
		n.Host.Close(),
		n.DHT.Close(),
		n.DS.Close(),
	)
}

// Default Behavior: https://pkg.go.dev/github.com/libp2p/go-libp2p#New
func NewNode(ctx context.Context, port int, rendezvous string) (*Node, error) {
	pkey, err := privKey()
	if err != nil {
		return nil, err
	}

	maddrs := []multiaddr.Multiaddr{}
	for _, s := range []string{
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port),
		fmt.Sprintf("/ip6/::/tcp/%d", port),
		fmt.Sprintf("/ip6/::/udp/%d/quic", port),
	} {
		ma, err := multiaddr.NewMultiaddr(s)
		if err != nil {
			return nil, err
		}
		maddrs = append(maddrs, ma)
	}

	p2pOpts := append([]libp2p.Option{}, ipfslite.Libp2pOptionsExtra...)
	p2pOpts = append(p2pOpts, []libp2p.Option{
		// libp2p.EnableAutoRelay(),
		libp2p.DefaultSecurity,
		libp2p.DefaultMuxers,
		libp2p.FallbackDefaults,
	}...)
	h, dht, err := ipfslite.SetupLibp2p(ctx, pkey, nil, maddrs, nil, p2pOpts...)
	if err != nil {
		return nil, err
	}

	badgerDS, err := badger.NewDatastore(fmt.Sprintf("./%s", DSName), &badger.DefaultOptions)
	if err != nil {
		return nil, err
	}
	lite, err := ipfslite.New(ctx, badgerDS, nil, h, dht, nil)
	if err != nil {
		return nil, err
	}
	lite.Bootstrap(defaultBootstrapPeersInfo)

	psub, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}
	bcast, err := crdt.NewPubSubBroadcaster(ctx, psub, rendezvous)
	if err != nil {
		return nil, err
	}
	// more setups
	// DAGService
	// dags := ...

	n := &Node{
		Host:       h,
		DHT:        dht,
		Lite:       lite,
		DSPutCh:    make(chan lo.Tuple2[datastore.Key, []byte]),
		DSDelCh:    make(chan datastore.Key),
		Rendezvous: rendezvous,
	}

	crdtOpts := crdt.DefaultOptions()
	crdtOpts.RebroadcastInterval = 5 * time.Second
	crdtOpts.PutHook = func(k datastore.Key, v []byte) { n.dsPutNotify(k, v) }
	crdtOpts.DeleteHook = func(k datastore.Key) { n.dsDeletedNotify(k) }
	crdtDS, err := crdt.New(badgerDS, DSKey, lite, bcast, crdtOpts)
	if err != nil {
		return nil, err
	}
	n.DS = crdtDS

	go n.run(psub)
	return n, nil
}

func (nd *Node) dsPutNotify(k datastore.Key, v []byte) {
	// fmt.Printf("Added: [%s] -> %d bytes\n", k, len(v))
	nd.DSPutCh <- lo.T2(k, v)
}

func (nd *Node) dsDeletedNotify(k datastore.Key) {
	// fmt.Printf("Removed: [%s]\n", k)
	// nd.DSDelCh <- k
	panic("Not implemented yet")
}

func (nd *Node) run(psub *pubsub.PubSub) {
	topic, err := psub.Join(fmt.Sprintf("%s-net", nd.Rendezvous))
	if err != nil {
		log.Fatalln(err)
	}
	netSubs, err := topic.Subscribe()
	if err != nil {
		log.Fatalln(err)
	}

	// Use a special pubsub topic to avoid disconnecting
	// from globaldb peers.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for {
			msg, err := netSubs.Next(ctx)
			if err != nil {
				log.Printf("subscribe: %+v\n", err)
				break
			}
			nd.Host.ConnManager().TagPeer(msg.ReceivedFrom, "keep", 100)
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				topic.Publish(ctx, []byte("hi!"))
				time.Sleep(20 * time.Second)
			}
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		ctx := context.Background()

		rd := routing.NewRoutingDiscovery(nd.DHT)
		util.Advertise(ctx, rd, nd.Rendezvous)

		peerCh, err := rd.FindPeers(ctx, nd.Rendezvous)
		if err != nil {
			log.Printf("DHT FindPeers failed: %+v\n", err)
			continue
		}

		for p := range peerCh {
			if p.ID == nd.Host.ID() || len(p.Addrs) == 0 {
				continue
			}
			if err := nd.Host.Connect(ctx, p); err != nil {
				// log.Println("DHT Connection failed:", p.ID, ">>", err)
				continue
			}
			if !Peers.AppendUnique(p.ID) {
				continue
			}
			log.Printf("Connected peer by DHT: %s\n", p.ID)

			if err := nd.DS.Sync(ctx, DSKey); err != nil {
				log.Fatalf("start sync first: %+v\n", err)
			}
		}
	}
}

type discoveryMDNS struct {
	PeerCh chan peer.AddrInfo
	host   host.Host
}

func (n *discoveryMDNS) HandlePeerFound(pi peer.AddrInfo) {
	n.PeerCh <- pi
}

func (n *discoveryMDNS) Run() {
	for {
		p := <-n.PeerCh
		if p.ID == n.host.ID() {
			continue
		}
		if err := n.host.Connect(context.Background(), p); err != nil {
			// log.Println("MDNS Connection failed:", p.ID, ">>", err)
			continue
		}
		if Peers.AppendUnique(p.ID) {
			log.Printf("Connect peer by MDNS: %s\n", p.ID)
		}
	}
}

func NewMDNS(h host.Host, rendezvous string) (*discoveryMDNS, error) {
	n := &discoveryMDNS{
		host:   h,
		PeerCh: make(chan peer.AddrInfo),
	}

	if err := mdns.NewMdnsService(h, rendezvous, n).Start(); err != nil {
		return nil, err
	}

	return n, nil
}

func privKey() (crypto.PrivKey, error) {
	name := dev.PrivateKeyName

	// Restore pkey
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		dat, err := ioutil.ReadFile(name)
		if err != nil {
			return nil, err
		}

		return crypto.UnmarshalPrivateKey(dat)
	}

	pkey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		return nil, err
	}
	// Store Key
	privBytes, err := crypto.MarshalPrivateKey(pkey)
	if err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(name, privBytes, 0644); err != nil {
		return nil, err
	}

	return pkey, nil
}
