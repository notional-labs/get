package get

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	gopath "path"
	"path/filepath"
	"sync"

	"github.com/cheggaaa/pb"
	files "github.com/ipfs/go-ipfs-files"
	ipfshttp "github.com/ipfs/go-ipfs-http-client"
	iface "github.com/ipfs/interface-go-ipfs-core"
	ipath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	cleanup      []func() error
	cleanupMutex sync.Mutex
)

// Connect Gets us connected to the IPFS network
func Connect(ctx context.Context, ipfs iface.CoreAPI, peers []string) {
	var wg sync.WaitGroup
	pinfos := make(map[peer.ID]*peer.AddrInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			fmt.Println("multiaddress issue!")

		}
		pii, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			fmt.Println("cannot connect!")
		}
		pi, ok := pinfos[pii.ID]
		if !ok {
			pi = &peer.AddrInfo{ID: pii.ID}
			pinfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	wg.Add(len(pinfos))
	for _, pi := range pinfos {
		go func(pi *peer.AddrInfo) {
			defer wg.Done()
			log.Printf("attempting to connect to peer: %q\n", pi)
			err := ipfs.Swarm().Connect(ctx, *pi)
			if err != nil {
				log.Printf("failed to connect to %s: %s", pi.ID, err)
			}
			log.Printf("successfully connected to %s\n", pi.ID)
		}(pi)
	}
	wg.Wait()
}

// WriteTo writes the given node to the local filesystem at fpath.
func WriteTo(nd files.Node, fpath string, progress bool) error {
	s, err := nd.Size()
	if err != nil {
		return err
	}

	var bar *pb.ProgressBar
	if progress {
		bar = pb.New64(s).Start()
	}

	return writeToRec(nd, fpath, bar)
}

func writeToRec(nd files.Node, fpath string, bar *pb.ProgressBar) error {
	switch nd := nd.(type) {
	case *files.Symlink:
		return os.Symlink(nd.Target, fpath)
	case files.File:
		f, err := os.Create(fpath)
		defer f.Close()
		if err != nil {
			return err
		}

		var r io.Reader = nd
		if bar != nil {
			r = bar.NewProxyReader(r)
		}
		_, err = io.Copy(f, r)
		if err != nil {
			return err
		}
		return nil
	case files.Directory:
		err := os.Mkdir(fpath, 0777)
		if err != nil {
			return err
		}

		entries := nd.Entries()
		for entries.Next() {
			child := filepath.Join(fpath, entries.Name())
			if err := writeToRec(entries.Node(), child, bar); err != nil {
				return err
			}
		}
		return entries.Err()
	default:
		return fmt.Errorf("file type %T at %q is not supported", nd, fpath)
	}
}

// takes an ipfs path and validates it
func ParsePath(path string) (ipath.Path, error) {
	ipfsPath := ipath.New(path)
	if ipfsPath.IsValid() == nil {
		return ipfsPath, nil
	}

	u, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("%q could not be parsed: %s", path, err)
	}

	switch proto := u.Scheme; proto {
	case "ipfs", "ipld", "ipns":
		ipfsPath = ipath.New(gopath.Join("/", proto, u.Host, u.Path))
	case "http", "https":
		ipfsPath = ipath.New(u.Path)
	default:
		return nil, fmt.Errorf("%q is not recognized as an IPFS path", path)
	}
	return ipfsPath, ipfsPath.IsValid()
}

func AddCleanup(f func() error) {
	cleanupMutex.Lock()
	defer cleanupMutex.Unlock()
	cleanup = append(cleanup, f)
}

func DoCleanup() {
	cleanupMutex.Lock()
	defer cleanupMutex.Unlock()

	for _, f := range cleanup {
		if err := f(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func http(ctx context.Context) (iface.CoreAPI, error) {
	httpApi, err := ipfshttp.NewLocalApi()
	if err != nil {
		return nil, err
	}

	err = httpApi.Request("version").Exec(ctx, nil)
	if err != nil {
		return nil, err
	}
	return httpApi, err
}

// Get takes fspath and cid, and then downloads a file from ipfs
func Get(fspath string, cid string) {
	//cleanup when done
	defer DoCleanup()

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	ipfs, err := http(ctx)
	if err != nil {
		fmt.Println(err)
		fmt.Println("flksdfj")
	}

	var nilslice []string = nil
	go Connect(ctx, ipfs, nilslice)

	iPath, err := ParsePath(cid)
	if err != nil {
		fmt.Println("Couldn't parse the cid!")
	}

	out, err := ipfs.Unixfs().Get(ctx, iPath)
	if err != nil {
		fmt.Println("err on the old unixfs")
	}

	progress := true

	err = WriteTo(out, fspath, progress)
	if err != nil {
		fmt.Println("Couldn't download the cid, sorry.")
	}
}
