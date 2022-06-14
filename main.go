package main

import (
	"flag"
	"log"
	"magma-automation/amboss"
	"magma-automation/lnd"
)

func main() {
	lndAddr := flag.String("addr", "localhost:10009", "the exposed LND rpc api. <IP>:<port>")
	macaroonPath := flag.String("macaroonpath", "~/.lnd/data/chain/bitcoin/mainnet/admin.macaroon", "path/to/admin.macaroon")
	tlsPath := flag.String("tlspath", "~/.lnd/tls.cert", "path/to/tls.cert")
	apiToken := flag.String("tokenpath", "~/api.key", "path to a file where the amboss-space api token is stores in plaintext")
	apiEndpoint := flag.String("apiendpoint", "https://api.amboss.space/graphql", "url of the amboss space api")

	flag.Parse()
	conn, err := lnd.NewConn(*macaroonPath, *tlsPath, *lndAddr)
	if err != nil {
		log.Fatalf("Error. %v", err)
	}
	defer conn.Close()
	lnd := lnd.NewClient(conn)
	info, err := lnd.GetInfo()
	if err != nil {
		log.Fatalf("Error. %v", err)
	}

	log.Println("Connected to LND!")
	magma := amboss.NewClient(*apiEndpoint, *apiToken)
	alias, err := magma.GetAlias(info.IdentityPubkey)
	if err != nil {
		log.Fatalf("Error. %v", err)
	}
	if alias != info.Alias {
		log.Fatalf("Error: LND alias %s different from magma alias %s", info.Alias, alias)
	}
	log.Println("Connected to Amboss!")
}
