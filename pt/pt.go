package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"go.fd.io/govpp"
	"go.fd.io/govpp/adapter/socketclient"
	"go.fd.io/govpp/api"
	interfaces "go.fd.io/govpp/binapi/interface"
	"go.fd.io/govpp/binapi/interface_types"
	"go.fd.io/govpp/binapi/sr_pt"
	"go.fd.io/govpp/binapi/vpe"
	"go.fd.io/govpp/core"
)

var (
	sockAddr = flag.String("sock", socketclient.DefaultSocketName, "Path to VPP binary API socket file")
)

func main() {
	flag.Parse()

	fmt.Println("Starting simple client example")
	fmt.Println()

	// connect to VPP asynchronously
	conn, connEv, err := govpp.AsyncConnect(*sockAddr, core.DefaultMaxReconnectAttempts, core.DefaultReconnectInterval)
	if err != nil {
		log.Fatalln("ERROR:", err)
	}
	defer conn.Disconnect()

	// wait for Connected event
	e := <-connEv
	if e.State != core.Connected {
		log.Fatalln("ERROR: connecting to VPP failed:", e.Error)
	}

	// check compatibility of used messages
	ch, err := conn.NewAPIChannel()
	if err != nil {
		log.Fatalln("ERROR: creating channel failed:", err)
	}
	defer ch.Close()
	if err := ch.CheckCompatiblity(vpe.AllMessages()...); err != nil {
		log.Fatal(err)
	}
	if err := ch.CheckCompatiblity(interfaces.AllMessages()...); err != nil {
		log.Fatal(err)
	}

	// use request/reply (channel API)
	dumpPtIface(ch)
	index := createLoopback(ch)
	addPtIface(ch, index)
	dumpPtIface(ch)
	delPtIface(ch, index)
	dumpPtIface(ch)
}

func dumpPtIface(ch api.Channel) {
	fmt.Println("Dumping pt interfaces..")

	n := 0
	reqCtx := ch.SendMultiRequest(&sr_pt.SrPtIfaceDump{})
	for {
		msg := &sr_pt.SrPtIfaceDetails{}
		stop, err := reqCtx.ReceiveReply(msg)
		if stop {
			break
		}
		if err != nil {
			log.Fatalf("dumping pt interfaces, err: %v", err)
			return
		}
		n++
		fmt.Printf(" - pt interface #%d: %+v\n", n, msg)
		marshal(msg)
	}

	fmt.Println("OK")
	fmt.Println()
}

func createLoopback(ch api.Channel) interface_types.InterfaceIndex {
	fmt.Println("Creating loopback..")

	req := &interfaces.CreateLoopback{}
	reply := &interfaces.CreateLoopbackReply{}

	if err := ch.SendRequest(req).ReceiveReply(reply); err != nil {
		log.Fatalf("ERROR: creating loopback: %v", err)
		return 0
	}

	fmt.Printf("Loopback created, index: %d\n", reply.SwIfIndex)
	fmt.Println("OK")
	fmt.Println()

	return reply.SwIfIndex
}

func addPtIface(ch api.Channel, index interface_types.InterfaceIndex) {
	fmt.Println("Adding pt interface..")
	req := &sr_pt.SrPtIfaceAdd{
		SwIfIndex:   index,
		ID:          400,
		TtsTemplate: 2,
		IngressLoad: 1,
		EgressLoad:  1,
	}
	marshal(req)
	reply := &sr_pt.SrPtIfaceAddReply{}

	if err := ch.SendRequest(req).ReceiveReply(reply); err != nil {
		log.Fatalf("ERROR: adding pt interface: %v", err)
		return
	}

	fmt.Println("OK")
	fmt.Println()
}

func delPtIface(ch api.Channel, index interface_types.InterfaceIndex) {
	fmt.Println("Deleting pt interface..")
	req := &sr_pt.SrPtIfaceDel{
		SwIfIndex: index,
	}
	marshal(req)
	reply := &sr_pt.SrPtIfaceDelReply{}

	if err := ch.SendRequest(req).ReceiveReply(reply); err != nil {
		log.Fatalf("ERROR: deleting pt interface: %v", err)
		return
	}

	fmt.Println("OK")
	fmt.Println()
}

func marshal(v interface{}) {
	fmt.Printf("GO: %#v\n", v)
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("JSON: %s\n", b)
}
