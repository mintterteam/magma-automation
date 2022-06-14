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
	apiToken := flag.String("tokenpath", "~/api.key", "path to a file where the amboss-space api token is stored in plaintext")
	apiEndpoint := flag.String("apiendpoint", "https://api.amboss.space/graphql", "url of the amboss space api")
	minFee := flag.Int("minfee", 2, "minimum fee (in sats per vByte) to pay for oppening channels")
	maxFee := flag.Int("maxfee", 10, "maximum fee (in sats per vByte) to pay for oppening channels")

	flag.Parse()
	conn, err := lnd.NewConn(*macaroonPath, *tlsPath, *lndAddr)
	if err != nil {
		log.Fatalf("[ERROR]: %v", err)
	}
	defer conn.Close()
	lnd := lnd.NewClient(conn)
	info, err := lnd.GetInfo()
	if err != nil {
		log.Fatalf("[ERROR]: %v", err)
	}

	log.Println("[INFO]: Connected to LND!")
	magma := amboss.NewClient(*apiEndpoint, *apiToken, *minFee, *maxFee)
	alias, err := magma.GetAlias(info.IdentityPubkey)
	if err != nil {
		log.Fatalf("[ERROR]: %v", err)
	}
	if alias != info.Alias {
		log.Fatalf("[ERROR]: LND alias %s different from magma alias %s", info.Alias, alias)
	}
	log.Println("Connected to Amboss!")
	order, err := magma.GetWaitingOrder()
	if err != nil {
		log.Printf("[WARNING]: Could not get Magma Orders %v", err)
	}
	if order.FeesvByte < *minFee {
		order.FeesvByte = *minFee
	} else if order.FeesvByte > *maxFee {
		log.Printf("[WARNING]: Current fees (%d) are higher than maximum fee alowed (%d)", order.FeesvByte, *maxFee)
	}

}
