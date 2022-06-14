package lnd

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	lndpb "github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	ALREADY_CONNECTED = "already connected"
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

func (c *Client) Connect(nodeid, addr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	_, err := c.lndclient.ConnectPeer(ctx, &lndpb.ConnectPeerRequest{
		Addr: &lndpb.LightningAddress{
			Pubkey: nodeid,
			Host:   addr,
		},
	})
	if err != nil && !strings.Contains(err.Error(), ALREADY_CONNECTED) {
		return err
	}
	return nil
}

func (c *Client) AvailableFunds() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	balance, err := c.lndclient.WalletBalance(ctx, &lndpb.WalletBalanceRequest{})

	if err != nil && !strings.Contains(err.Error(), ALREADY_CONNECTED) {
		return 0, err
	}
	return int(balance.GetConfirmedBalance()), nil
}

func (c *Client) GetInvoice(amt, expiry int, memo string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	res, err := c.lndclient.AddInvoice(ctx, &lndpb.Invoice{Memo: memo, Value: int64(amt), Expiry: int64(expiry)})

	if err != nil {
		return "", err
	}
	return res.GetPaymentRequest(), nil
}

func (c *Client) OpenChannel(amt, fees int, peerId string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	var nodePubHex []byte
	var err error
	if nodePubHex, err = hex.DecodeString(peerId); err != nil {
		return "", err
	}
	stream, err := c.lndclient.OpenChannel(ctx, &lndpb.OpenChannelRequest{
		SatPerVbyte:        uint64(fees),
		NodePubkey:         nodePubHex,
		LocalFundingAmount: int64(amt),
	})

	if err != nil {
		return "", err
	}
	channelPoint := ""
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return "", fmt.Errorf("Get EOF while waiting for channel oppening")
		} else if err != nil {
			return "", err
		}

		switch update := resp.Update.(type) {
		case *lndpb.OpenStatusUpdate_ChanPending:
			channelPoint = channelIdToString(update.ChanPending.Txid) + ":" + fmt.Sprint(update.ChanPending.OutputIndex)

		case *lndpb.OpenStatusUpdate_ChanOpen:
			channelPoint = update.ChanOpen.ChannelPoint.GetFundingTxidStr() + ":" + fmt.Sprint(update.ChanOpen.ChannelPoint.GetOutputIndex())
		}
		break
	}
	return channelPoint, nil
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

// This function takes a byte array, swaps it and encodes the hex representation in a string
func channelIdToString(hash []byte) string {
	HashSize := len(hash)
	for i := 0; i < HashSize/2; i++ {
		hash[i], hash[HashSize-1-i] = hash[HashSize-1-i], hash[i]
	}
	return hex.EncodeToString(hash[:])
}
