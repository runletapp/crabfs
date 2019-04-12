package crabfs

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/runletapp/crabfs/cryptostream"

	"github.com/golang/protobuf/proto"
	billy "gopkg.in/src-d/go-billy.v4"

	cid "github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	libp2pCircuit "github.com/libp2p/go-libp2p-circuit"
	libp2pCrypto "github.com/libp2p/go-libp2p-crypto"
	discovery "github.com/libp2p/go-libp2p-discovery"
	libp2pHost "github.com/libp2p/go-libp2p-host"
	libp2pdht "github.com/libp2p/go-libp2p-kad-dht"
	libp2pdhtOptions "github.com/libp2p/go-libp2p-kad-dht/opts"
	libp2pNet "github.com/libp2p/go-libp2p-net"
	libp2pPeerstore "github.com/libp2p/go-libp2p-peerstore"
	libp2pProtocol "github.com/libp2p/go-libp2p-protocol"
	libp2pRoutedHost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multiaddr"
	multihash "github.com/multiformats/go-multihash"
)

var (
	// ProtocolV1 crabfs version 1
	ProtocolV1 = libp2pProtocol.ID("/crabfs/v1")
)

// Host controls the p2p host instance
type Host struct {
	ctx context.Context

	port int

	p2pHost libp2pHost.Host
	dht     *libp2pdht.IpfsDHT

	privateKey *libp2pCrypto.RsaPrivateKey

	publicKeyData []byte
	publicKeyHash string

	mountFS billy.Filesystem
}

// HostNew creates a new host instance
func HostNew(ctx context.Context, port int, privateKey *libp2pCrypto.RsaPrivateKey, mountFS billy.Filesystem) (*Host, error) {
	if privateKey == nil {
		return nil, ErrInvalidPrivateKey
	}

	publicKeyData, err := libp2pCrypto.MarshalRsaPublicKey(privateKey.GetPublic().(*libp2pCrypto.RsaPublicKey))
	if err != nil {
		return nil, err
	}
	publicKeyHash, err := multihash.Sum(publicKeyData, multihash.SHA3_256, -1)
	if err != nil {
		return nil, err
	}

	host := &Host{
		ctx:  ctx,
		port: port,

		privateKey: privateKey,

		publicKeyHash: publicKeyHash.String(),
		publicKeyData: publicKeyData,

		mountFS: mountFS,
	}

	sourceMultiAddrIP4, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	if err != nil {
		return nil, err
	}

	sourceMultiAddrIP6, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip6/::/tcp/%d", port))
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrs(sourceMultiAddrIP4, sourceMultiAddrIP6),
		libp2p.EnableRelay(libp2pCircuit.OptDiscovery),
	}

	p2pHost, err := libp2p.New(
		ctx,
		opts...,
	)
	if err != nil {
		return nil, err
	}

	// Configure peer discovery and key validator
	dhtValidator := libp2pdhtOptions.NamespacedValidator(
		"crabfs",
		DHTNamespaceValidatorNew(ctx, host.getSwarmPublicKey),
	)
	dhtPKValidator := libp2pdhtOptions.NamespacedValidator(
		"crabfs_pk",
		DHTNamespacePKValidatorNew(),
	)
	dht, err := libp2pdht.New(ctx, p2pHost, dhtValidator, dhtPKValidator)
	if err != nil {
		return nil, err
	}

	host.dht = dht

	if err = dht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	host.p2pHost = libp2pRoutedHost.Wrap(p2pHost, dht)
	host.p2pHost.SetStreamHandler(ProtocolV1, host.handleStreamV1)

	return host, nil
}

func (host *Host) handleStreamV1(rawStream libp2pNet.Stream) {
	var stream io.ReadWriteCloser
	if host.privateKey == nil {
		stream = rawStream
	} else {
		var err error
		stream, err = cryptostream.New(host.privateKey, rawStream)
		if err != nil {
			rawStream.Close()
			return
		}
	}

	defer stream.Close()

	data, err := ioutil.ReadAll(stream)
	if err != nil {
		return
	}

	var request ProtocolRequest
	if err := proto.Unmarshal(data, &request); err != nil {
		return
	}

	file, err := host.mountFS.Open(request.PathName)
	if err != nil {
		return
	}

	size, err := file.Seek(0, os.SEEK_END)
	if err != nil {
		return
	}

	var offset int64
	if request.Offset > size {
		offset = size
	} else if request.Offset > 0 {
		offset = request.Offset
	} else {
		offset = 0
	}
	_, err = file.Seek(offset, os.SEEK_SET)
	if err != nil {
		return
	}

	var length int64

	if request.Length <= 0 || request.Length >= size-offset {
		length = size - offset
	} else {
		length = request.Length
	}

	// Send back the actual length
	lengthData := make([]byte, 8)
	binary.PutVarint(lengthData, length)
	if _, err := stream.Write(lengthData); err != nil {
		return
	}

	var totalSent int64

	bufferSize := 500 * 1024
	buffer := make([]byte, bufferSize)

	for totalSent < length {
		n, err := file.Read(buffer)
		if err != nil {
			// Unexpected error
			return
		}

		n, err = stream.Write(buffer[:n])
		if err != nil {
			// Unexpected error
			return
		}
		totalSent += int64(n)
	}
}

func (host *Host) sendMessageToStream(stream io.Writer, message proto.Message) (int, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		return 0, err
	}

	return stream.Write(data)
}

// GetID returns the id of this p2p host
func (host *Host) GetID() string {
	return host.p2pHost.ID().Pretty()
}

// ConnectToPeer connect this node to a peer
func (host *Host) ConnectToPeer(addr string) error {
	ma := multiaddr.StringCast(addr)

	peerinfo, err := libp2pPeerstore.InfoFromP2pAddr(ma)
	if err != nil {
		return err
	}

	return host.p2pHost.Connect(host.ctx, *peerinfo)
}

// GetAddrs get the addresses that this node is bound to
func (host *Host) GetAddrs() []string {
	addrs := []string{}

	hostAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", host.p2pHost.ID().Pretty()))
	if err != nil {
		return addrs
	}

	for _, addr := range host.p2pHost.Addrs() {
		addrs = append(addrs, addr.Encapsulate(hostAddr).String())
	}

	return addrs
}

// Announce announce this host in the network
func (host *Host) Announce() error {
	routingDiscovery := discovery.NewRoutingDiscovery(host.dht)
	discovery.Advertise(host.ctx, routingDiscovery, host.publicKeyHash)

	if err := host.dht.PutValue(host.ctx, fmt.Sprintf("/crabfs_pk/%s", host.publicKeyHash), host.publicKeyData); err != nil {
		return err
	}

	return nil
}

func (host *Host) pathNameToKey(name string) (string, error) {
	nameHash, err := multihash.Sum([]byte(name), multihash.SHA3_256, -1)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("/crabfs/%s/%s", host.publicKeyHash, nameHash.String()), nil
}

// AnnounceContent announce that we can provide the content id 'contentID'
func (host *Host) AnnounceContent(name string, contentID *cid.Cid, record *DHTNameRecord) error {
	var contentIDData []byte
	if contentID != nil {
		err := host.dht.Provide(host.ctx, *contentID, true)
		if err != nil {
			return err
		}
		contentIDData = contentID.Bytes()
	} else {
		contentIDData = []byte{}
	}

	key, err := host.pathNameToKey(name)
	if err != nil {
		return err
	}

	record.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	record.Data = contentIDData

	signData, err := host.privateKey.Sign(contentIDData)
	if err != nil {
		return err
	}
	record.Signature = signData

	value, err := proto.Marshal(&record.DHTNameRecordV1)
	if err != nil {
		return err
	}

	return host.dht.PutValue(host.ctx, key, value)
}

// GetContentRecord resolves the pathName to a content id
func (host *Host) GetContentRecord(ctx context.Context, name string) (*DHTNameRecord, error) {
	key, err := host.pathNameToKey(name)
	if err != nil {
		return nil, err
	}

	data, err := host.dht.GetValue(ctx, key)
	if err != nil {
		return nil, err
	}

	var rawRecord DHTNameRecordV1
	if err := proto.Unmarshal(data, &rawRecord); err != nil {
		return nil, err
	}

	if len(rawRecord.Data) == 0 {
		return nil, ErrNotFound
	}

	contentID, err := cid.Cast(rawRecord.Data)
	if err != nil {
		return nil, err
	}

	record := DHTNameRecord{
		DHTNameRecordV1: rawRecord,

		ContentID: &contentID,
	}

	return &record, nil
}

// GetProviders resolves the content id to a list of peers that provides that content
func (host *Host) GetProviders(ctx context.Context, contentID *cid.Cid) ([][]string, error) {
	peers := [][]string{}

	peerInfoList, err := host.dht.FindProviders(ctx, *contentID)
	if err != nil {
		return nil, err
	}

	for _, peerInfo := range peerInfoList {
		addrs := peerInfo.Addrs

		peerAddrID, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", peerInfo.ID.Pretty()))
		if err != nil {
			return nil, err
		}

		addrsStr := []string{}
		for _, addr := range addrs {
			addrsStr = append(addrsStr, addr.Encapsulate(peerAddrID).String())
		}

		peers = append(peers, addrsStr)
	}

	return peers, nil
}

// CreateStream create a stream from remote peers
func (host *Host) CreateStream(ctx context.Context, contentID *cid.Cid, pathName string) (chan *StreamSlice, error) {
	peerInfoList, err := host.dht.FindProviders(ctx, *contentID)
	if err != nil {
		return nil, err
	}

	streamChan := make(chan *StreamSlice)

	go func() {
		defer close(streamChan)
		for _, peer := range peerInfoList {
			rawStream, err := host.p2pHost.NewStream(ctx, peer.ID, ProtocolV1)
			if err != nil {
				continue
			}

			var stream io.ReadWriteCloser

			if host.privateKey == nil {
				stream = rawStream
			} else {
				var err error
				stream, err = cryptostream.New(host.privateKey, rawStream)
				if err != nil {
					continue
				}
			}

			request := ProtocolRequest{
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Cid:       contentID.Bytes(),
				Offset:    0,
				Length:    0,
				PathName:  pathName,
			}
			if _, err := host.sendMessageToStream(stream, &request); err != nil {
				return
			}
			stream.Close()

			lengthData := make([]byte, 8)
			if _, err := stream.Read(lengthData); err != nil {
				return
			}

			length, err := binary.ReadVarint(bytes.NewReader(lengthData))
			// TODO use length < 0 to return errors
			if length == 0 {
				return
			}

			slice := StreamSlice{
				Reader: stream,
				Offset: 0,
				Length: length,
			}

			streamChan <- &slice
		}
	}()

	return streamChan, nil
}

func (host *Host) getSwarmPublicKey(ctx context.Context, hash string) (*libp2pCrypto.RsaPublicKey, error) {
	data, err := host.dht.GetValue(ctx, fmt.Sprintf("/crabfs_pk/%s", hash))
	if err != nil {
		return nil, err
	}

	pk, err := libp2pCrypto.UnmarshalRsaPublicKey(data)
	if err != nil {
		return nil, err
	}

	return pk.(*libp2pCrypto.RsaPublicKey), nil
}
