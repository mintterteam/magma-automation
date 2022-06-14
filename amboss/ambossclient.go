package amboss

import (
	"context"
	"net/http"
	"regexp"
	"strconv"

	graphql "github.com/hasura/go-graphql-client"
	"golang.org/x/oauth2"
)

const (
	STATUS_WAITING_APPROVAL = "WAITING_FOR_SELLER_APPROVAL"
)

var (
	ip_regexp    = regexp.MustCompile("^(?:[0-9]{1,3}\\.){3}[0-9]{1,3}:[0-9]+$")
	onion_regexp = regexp.MustCompile("^[a-z0-9]+\\.onion:[0-9]+$")
)

type Client struct {
	magmaclient *graphql.Client
}

type Order struct {
	Id        string
	Sats      int64
	Peer      string
	FeesvByte int
}

func NewClient(apiendpoint, token string, minfee, maxfee int) *Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	var httpClient *http.Client
	if token != "" {
		httpClient = oauth2.NewClient(context.Background(), src)
	}

	client := graphql.NewClient(apiendpoint, httpClient)
	return &Client{
		magmaclient: client,
	}
}

func (c *Client) Helloworld() (string, error) {
	var query struct {
		GetHello graphql.String
	}
	err := c.magmaclient.Query(context.Background(), &query, nil)
	if err != nil {
		return "", err
	}
	return string(query.GetHello), nil
}

func (c *Client) GetAlias(nodeID string) (string, error) {
	var query struct {
		GetNode struct {
			Graph_info struct {
				Node struct {
					Alias graphql.String
				}
			}
		} `graphql:"getNode(pubkey: $pubkey)"`
	}
	variables := map[string]interface{}{
		"pubkey": graphql.String(nodeID),
	}
	err := c.magmaclient.Query(context.Background(), &query, variables)
	if err != nil {
		return "", err
	}
	return string(query.GetNode.Graph_info.Node.Alias), nil
}

func (c *Client) GetNodeAddress(nodeID string) (string, error) {
	var query struct {
		GetNode struct {
			Graph_info struct {
				Node struct {
					Addresses []struct {
						Addr string
					}
				}
			}
		} `graphql:"getNode(pubkey: $pubkey)"`
	}
	variables := map[string]interface{}{
		"pubkey": graphql.String(nodeID),
	}
	err := c.magmaclient.Query(context.Background(), &query, variables)
	if err != nil {
		return "", err
	}
	var ret string = ""
	for _, addr := range query.GetNode.Graph_info.Node.Addresses {
		if ip_regexp.Match([]byte(addr.Addr)) {
			return addr.Addr, nil
		} else if onion_regexp.Match([]byte(addr.Addr)) {
			ret = addr.Addr
		}
	}
	return ret, nil
}

func (c *Client) AcceptOrder(id, payreq string) error {
	var m struct {
		SellerAcceptOrder string `graphql:"sellerAcceptOrder(id: $sellerAcceptOrderId, request: $request)"`
	}

	variables := map[string]interface{}{
		"sellerAcceptOrderId": graphql.String(id),
		"request":             graphql.String(payreq),
	}
	err := c.magmaclient.Mutate(context.Background(), &m, variables)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) GetWaitingOrder() (*Order, error) {
	var query struct {
		GetOfferOrders struct {
			List []struct {
				Account string
				Id      string
				Size    string
				Status  string
			}
		}
		GetMempoolFees struct {
			HalfHourFee graphql.Float
			HourFee     graphql.Float
		}
	}

	err := c.magmaclient.Query(context.Background(), &query, nil)
	if err != nil {
		return nil, err
	}
	for _, order := range query.GetOfferOrders.List {
		if order.Status == STATUS_WAITING_APPROVAL {
			amt, err := strconv.ParseInt(order.Size, 10, 64)
			if err != nil {
				return nil, err
			}
			return &Order{
				Id:        order.Id,
				Sats:      amt,
				Peer:      order.Account,
				FeesvByte: int(query.GetMempoolFees.HourFee),
			}, nil
		}
	}
	return nil, nil
}
