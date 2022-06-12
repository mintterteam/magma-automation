package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	lndpb "github.com/lightningnetwork/lnd/lnrpc"
)

var lndAddr = flag.String("addr", "localhost:10009", "the exposed LND rpc api. <IP>:<port>")
var macaroonPath = flag.String("macaroonpath", "~/.lnd/data/chain/bitcoin/mainnet/admin.macaroon", "path/to/admin.macaroon")
var tlsPath = flag.String("tlspath", "~/.lnd/tls.cert", "path/to/tls.cert")
var apiToken = flag.String("tokenpath", "~/api.key", "path to a file where the amboss-space api token is stores in plaintext")

type macaroon struct {
	macaroon string
}

func callGetInfo(client lndpb.LightningClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	resp, err := client.GetInfo(ctx, &lndpb.GetInfoRequest{})
	if err != nil {
		log.Fatalf("callGetInfo = _, %v: ", err)
	}
	resp.ProtoMessage()
	fmt.Printf("%+v\n", resp)
}

func main() {
	flag.Parse()
	conn, err := connectLND()
	if err != nil {
		log.Fatalf("Error. %v", err)
	}
	defer conn.Close()
	log.Println("Connected to LND!")
	rgc := lndpb.NewLightningClient(conn)
	callGetInfo(rgc)
}

func connectLND() (*grpc.ClientConn, error) {
	// Set up the credentials for the connection.
	macaroon, err := getMacaroon(*macaroonPath)

	if err != nil {
		return nil, fmt.Errorf("failed to load macaroon: %v", err)
	}

	creds, err := credentials.NewClientTLSFromFile(*tlsPath, "")
	if err != nil {
		return nil, fmt.Errorf("failed to load cert: %v", err)
	}

	opts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(macaroon),
		grpc.WithTransportCredentials(creds),
		grpc.WithTimeout(time.Second * 10),
	}

	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(*lndAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("Couldn't connect to %s: %v", *lndAddr, err)
	}
	return conn, nil
}
func getMacaroon(path string) (macaroon, error) {
	mac := macaroon{""}
	hexa, err := ioutil.ReadFile(path)
	if err != nil {
		return mac, err
	}
	mac.macaroon = hex.EncodeToString(hexa)

	return mac, nil
}

func (m macaroon) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"macaroon": m.macaroon,
	}, nil
}

func (macaroon) RequireTransportSecurity() bool {
	return true
}
