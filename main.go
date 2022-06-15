package main

import (
	"flag"
	"io/ioutil"
	"log"
	"magma-automation/amboss"
	"magma-automation/lnd"
	"strings"
	"time"
)

func main() {
	lndAddr := flag.String("addr", "localhost:10009", "the exposed LND rpc api. <IP>:<port>")
	macaroonPath := flag.String("macaroonpath", "~/.lnd/data/chain/bitcoin/mainnet/admin.macaroon", "path/to/admin.macaroon")
	tlsPath := flag.String("tlspath", "~/.lnd/tls.cert", "path/to/tls.cert")
	apiTokenPath := flag.String("tokenpath", "~/api.key", "path to a file where the amboss-space api token is stored in plaintext")
	apiEndpoint := flag.String("apiendpoint", "https://api.amboss.space/graphql", "url of the amboss space api")
	minFee := flag.Int("minfee", 2, "minimum mining fee (in sats per vByte) to pay for oppening channels")
	maxFee := flag.Int("maxfee", 10, "maximum mining fee (in sats per vByte) to pay for oppening channels")
	period := flag.Int("period", 0, "time to wait in seconds bewteen rounds. If <=0 then we will do one round and exit. Infinite loop otherwise")
	rejectOnFailure := flag.Bool("rejectonfailure", false, "Flag to indicate rejecting the offer if there exists any failure. Do nothing otherwise")

	flag.Parse()
	conn, err := lnd.NewConn(*macaroonPath, *tlsPath, *lndAddr)
	if err != nil {
		log.Fatalf("[ERROR]: Could not connect to lnd. %v", err)
	}
	defer conn.Close()
	lnd := lnd.NewClient(conn)
	info, err := lnd.GetInfo()
	if err != nil {
		log.Fatalf("[ERROR]: Initial lnd healthcheck failed. %v", err)
	}
	log.Println("[INFO]: Connected to LND!")

	token, err := ioutil.ReadFile(*apiTokenPath)

	if err != nil {
		log.Fatalf("[ERROR]: Could not read provided token file %v", err)
	}

	magma := amboss.NewClient(*apiEndpoint, strings.TrimSuffix(string(token), "\n"), *minFee, *maxFee)
	alias, err := magma.GetAlias(info.IdentityPubkey)
	if err != nil {
		log.Fatalf("[ERROR]: Initial amboss healthcheck failed. %v", err)
	}
	if alias != info.Alias {
		log.Fatalf("[ERROR]: LND alias %s different from magma alias %s", info.Alias, alias)
	}
	_, err = magma.GetWaitingOrder()
	if err != nil {
		log.Fatalf("[ERROR]: Initial auth check failed. %v", err)
	}
	log.Println("[INFO]: Connected to Amboss!")
	for {
		if *period > 0 {
			time.Sleep(time.Duration(*period) * time.Second)
		}

		order, err := magma.GetWaitingOrder()
		if order != nil {
			if err != nil {
				log.Printf("[WARNING]: Could not get Magma Orders %v", err)
				continue
			}
			if order.FeesvByte > *maxFee {
				log.Printf("[WARNING]: Current mining fees (%d) are higher than maximum fees alowed (%d)", order.FeesvByte, *maxFee)
				continue
			}
			addr, err := magma.GetNodeAddress(order.Peer)
			if err != nil {
				log.Printf("[WARNING]: Could not get node address %v", err)
				reject(*rejectOnFailure, magma, order.Id)
				continue
			}
			if err := lnd.Connect(order.Peer, addr); err != nil {
				log.Printf("[WARNING]: Could not connect to %s@%s", order.Peer, addr)
				reject(*rejectOnFailure, magma, order.Id)
				continue
			}

			if funds, err := lnd.AvailableFunds(); err != nil || funds < int(order.ChanSize) {
				if err != nil {
					log.Printf("[WARNING]: Could not get funds %v", err)
				} else {
					log.Printf("[WARNING]: Insufficient funds (%d) to afford the lease (%d)", funds, int(order.ChanSize))
				}

				reject(*rejectOnFailure, magma, order.Id)
				continue
			}
			payreq, err := lnd.GetInvoice(int(order.InvoiceAmt), 300000, "magma "+order.Id)
			if err != nil {
				log.Printf("[WARNING]: Could not generate invoice for order %s. %v", order.Id, err)
				reject(*rejectOnFailure, magma, order.Id)
				continue
			}
			if err := magma.AcceptOrder(order.Id, payreq); err != nil {
				log.Printf("[WARNING]: Error trying to accept order id %s. %v", order.Id, err)
				reject(*rejectOnFailure, magma, order.Id)
				continue
			}
			log.Printf("[INFO]: Accepted order . %v", err)
		}
		order, err = magma.GetWaiting2Open()
		if err != nil {
			log.Printf("[WARNING]: Could not get Magma Orders waiting for channel opening. %v", err)
			continue
		}
		if order != nil {
			if order.FeesvByte < *minFee {
				order.FeesvByte = *minFee
			} else if order.FeesvByte > *maxFee {
				log.Printf("[WARNING]: Current mining fees (%d) are higher than maximum fees allowed (%d)", order.FeesvByte, *maxFee)
				continue
			}
			chanPoint, err := lnd.OpenChannel(int(order.ChanSize), order.FeesvByte, order.Peer)
			if err != nil {
				log.Fatalf("[ERROR]: Could not open channel for order %s. %v", order.Id, err)
			}
			chanSplit := strings.Split(chanPoint, ":")
			if len(chanSplit) != 2 {
				log.Fatalf("[ERROR]: Wrong chanpoint format %s", chanPoint)
			}
			if err := magma.NotifyChannelPoint(chanSplit[0], chanSplit[1]); err != nil {
				log.Fatalf("[ERROR]: Could not notify channel opening on order %s. %v", order.Id, err)
			}
			log.Printf("[INFO]: Sucessfully channel notification (%s). Will earn %dsats once it has over 3 confirmations.", chanPoint, int(order.InvoiceAmt))
		}

		if *period <= 0 {
			break
		}
	}

}

func reject(rejectonfailure bool, magma *amboss.Client, id string) {
	if rejectonfailure {
		if err := magma.RejectOrder(id); err != nil {
			log.Printf("[WARNING]: Could not reject order id %s", id)
		}
	}
}
