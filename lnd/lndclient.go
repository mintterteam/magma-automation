package lnd

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"time"

	lndpb "github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type macaroon struct {
	macaroon string
}

type Client struct {
	lndclient lndpb.LightningClient
}

func NewClient(conn *grpc.ClientConn) *Client {
	client := lndpb.NewLightningClient(conn)
	return &Client{
		lndclient: client,
	}
}

func NewConn(macaroonPath, tlsPath, lndAddr string) (*grpc.ClientConn, error) {
	// Set up the credentials for the connection.
	macaroon, err := getMacaroon(macaroonPath)

	if err != nil {
		return nil, fmt.Errorf("failed to load macaroon: %v", err)
	}

	creds, err := credentials.NewClientTLSFromFile(tlsPath, "")
	if err != nil {
		return nil, fmt.Errorf("failed to load cert: %v", err)
	}

	opts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(macaroon),
		grpc.WithTransportCredentials(creds),
		grpc.WithTimeout(time.Second * 10),
	}

	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(lndAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("Couldn't connect to %s: %v", lndAddr, err)
	}
	return conn, nil

}

func (c *Client) GetInfo() (*lndpb.GetInfoResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	resp, err := c.lndclient.GetInfo(ctx, &lndpb.GetInfoRequest{})
	if err != nil {
		return nil, err
	}
	return resp, nil
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
