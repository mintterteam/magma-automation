package amboss

import (
	"context"

	graphql "github.com/hasura/go-graphql-client"
	"golang.org/x/oauth2"
)

type Client struct {
	magmaclient *graphql.Client
}

func NewClient(apiendpoint, token string) *Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

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
		} `graphql:"getnode(pubkey: $pubkey)"`
	}
	variables := map[string]interface{}{
		"pubkey": graphql.ID(nodeID),
	}
	err := c.magmaclient.Query(context.Background(), &query, variables)
	if err != nil {
		return "", err
	}
	return string(query.GetNode.Graph_info.Node.Alias), nil
	/*
		query ExampleQuery($pubkey: String!) {
			getNode(pubkey: $pubkey) {
			  graph_info {
				node {
				  alias
				}
			  }
			}
		  }
	*/
}

/*
func (c *Client) GetOffers() {
	query ExampleQuery {
		getOfferOrders {
		  list {
			account
			id
			size
			status
		  }
		}
		getMempoolFees {
		  halfHourFee
		  hourFee
		}
	  }
}*/
