package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/crgimenes/go-osc"
)

const FPS = 60

const QueryDelay = 250 * time.Millisecond

var channelMixFader [32]float32
var auxMixFader [6]float32
var busMixFader [16]float32

var conn *osc.ServerAndClient

func main() {
	ipAddrStr := os.Args[1]
	ipAddr := net.ParseIP(ipAddrStr)
	if ipAddr == nil {
		log.Fatalf("Invalid console IP address")
	}

	addr1, err := net.ResolveUDPAddr("udp", "0.0.0.0:10022")
	if err != nil {
		log.Fatalf("failed resolving local port")
	}

	addr2, err := net.ResolveUDPAddr("udp", ipAddr.String()+":10023")
	if err != nil {
		log.Fatalf("failed resolving desk port")
	}

	respDispatcher := osc.NewStandardDispatcher()
	err = respDispatcher.AddMsgHandler("*", responseDispatch)
	if err != nil {
		log.Fatalf("Failed adding response message handler: %s", err)
	}

	conn = osc.NewServerAndClient(respDispatcher)
	err = conn.NewConn(addr1, addr2)
	if err != nil {
		log.Fatalf("Failed setting up connection to desk: %s", err)
	}

	go conn.ListenAndServe()

	commandDispatcher := osc.NewStandardDispatcher()
	err = commandDispatcher.AddMsgHandler("/fade/channel", channelFade)
	if err != nil {
		log.Fatalf("Failed adding message handler: %s", err)
	}
	err = commandDispatcher.AddMsgHandler("/fade/aux", auxFade)
	if err != nil {
		log.Fatalf("Failed adding message handler: %s", err)
	}
	err = commandDispatcher.AddMsgHandler("/fade/bus", busFade)
	if err != nil {
		log.Fatalf("Failed adding message handler: %s", err)
	}

	server := &osc.Server{
		Addr:       "0.0.0.0:10021",
		Dispatcher: commandDispatcher,
	}
	err = server.ListenAndServe()
	if err != nil {
		log.Fatalf("ListenAndServe failed: %s", err)
	}
}

func channelFade(msg *osc.Message) {
	id, err := msg.Arguments.Int32(0)
	if err != nil {
		log.Printf("failed retrieving ID: %s (%s)", err, msg.String())
		return
	}
	var target float32
	var duration float32

	switch msg.Arguments[1].(type) {
	case float32:
		target = msg.Arguments[1].(float32)
	case int32:
		target = float32(msg.Arguments[1].(int32))
	default:
		log.Printf("invalid argument format for target")
		return
	}
	switch msg.Arguments[2].(type) {
	case float32:
		duration = msg.Arguments[2].(float32)
	case int32:
		duration = float32(msg.Arguments[2].(int32))
	default:
		log.Printf("invalid argument format for duration")
		return
	}

	if id < 1 || id > 32 {
		log.Printf("ID out of range, ignoring: %d", id)
		return
	}
	if target < 0 || target > 1 {
		log.Printf("Target out of range, ignoring: %f", target)
		return
	}
	if duration < 0 || duration > 60 {
		log.Printf("Duration out of range, ignoring: %f", duration)
		return
	}

	address := fmt.Sprintf("/ch/%d/mix/fader", id)

	err = conn.Send(osc.NewMessage(address))
	if err != nil {
		log.Printf("failed sending fader GET command: %s (%s)", err, msg.String())
		return
	}

	time.Sleep(QueryDelay)

	go Fade(conn, address, channelMixFader[int(id-1)], target, time.Duration(duration*float32(time.Second)))
	channelMixFader[int(id-1)] = target
}

func auxFade(msg *osc.Message) {
	id, err := msg.Arguments.Int32(0)
	if err != nil {
		log.Printf("failed retrieving ID: %s (%s)", err, msg.String())
		return
	}
	target, err := msg.Arguments.Float32(0)
	if err != nil {
		log.Printf("failed retrieving target: %s (%s)", err, msg.String())
		return
	}
	duration, err := msg.Arguments.Float32(0)
	if err != nil {
		log.Printf("failed retrieving duration: %s (%s)", err, msg.String())
		return
	}

	if id < 1 || id > 6 {
		log.Printf("ID out of range, ignoring: %d", id)
		return
	}
	if target < 0 || target > 1 {
		log.Printf("Target out of range, ignoring: %f", target)
		return
	}
	if duration < 0 || duration > 60 {
		log.Printf("Duration out of range, ignoring: %f", duration)
		return
	}

	address := fmt.Sprintf("/auxin/%d/mix/fader", id)

	err = conn.Send(osc.NewMessage(address))
	if err != nil {
		log.Printf("failed sending fader GET command: %s (%s)", err, msg.String())
		return
	}

	time.Sleep(QueryDelay)

	go Fade(conn, address, auxMixFader[int(id-1)], target, time.Duration(duration*float32(time.Second)))
	auxMixFader[int(id-1)] = target
}

func busFade(msg *osc.Message) {
	id, err := msg.Arguments.Int32(0)
	if err != nil {
		log.Printf("failed retrieving ID: %s (%s)", err, msg.String())
		return
	}
	target, err := msg.Arguments.Float32(0)
	if err != nil {
		log.Printf("failed retrieving target: %s (%s)", err, msg.String())
		return
	}
	duration, err := msg.Arguments.Float32(0)
	if err != nil {
		log.Printf("failed retrieving duration: %s (%s)", err, msg.String())
		return
	}

	if id < 1 || id > 16 {
		log.Printf("ID out of range, ignoring: %d", id)
		return
	}
	if target < 0 || target > 1 {
		log.Printf("Target out of range, ignoring: %f", target)
		return
	}
	if duration < 0 || duration > 60 {
		log.Printf("Duration out of range, ignoring: %f", duration)
		return
	}

	address := fmt.Sprintf("/bus/%d/mix/fader", id)

	err = conn.Send(osc.NewMessage(address))
	if err != nil {
		log.Printf("failed sending fader GET command: %s (%s)", err, msg.String())
		return
	}

	time.Sleep(QueryDelay)

	go Fade(conn, address, busMixFader[int(id-1)], target, time.Duration(duration*float32(time.Second)))
	busMixFader[int(id-1)] = target
}

func responseDispatch(msg *osc.Message) {
	data := strings.Split(msg.Address[1:], "/")
	switch data[0] {
	case "ch":
		id, err := strconv.Atoi(data[1])
		if err != nil {
			log.Printf("invalid ID: %s (%s)", data[1], msg.String())
			return
		}
		id-- // Convert to zero index

		if data[2] == "mix" && data[3] == "fader" {
			value, err := msg.Arguments.Float32(0)
			if err != nil {
				log.Printf("failed parsing value: %s (%s)", err, msg.String())
				return
			}

			channelMixFader[id] = value
		}

	case "auxin":
		id, err := strconv.Atoi(data[1])
		if err != nil {
			log.Printf("invalid ID: %s (%s)", data[1], msg.String())
			return
		}
		id-- // Convert to zero index

		if data[2] == "mix" && data[3] == "fader" {
			value, err := msg.Arguments.Float32(0)
			if err != nil {
				log.Printf("failed parsing value: %s (%s)", err, msg.String())
				return
			}

			auxMixFader[id] = value
		}

	case "bus":
		id, err := strconv.Atoi(data[1])
		if err != nil {
			log.Printf("invalid ID: %s (%s)", data[1], msg.String())
			return
		}
		id-- // Convert to zero index

		if data[2] == "mix" && data[3] == "fader" {
			value, err := msg.Arguments.Float32(0)
			if err != nil {
				log.Printf("failed parsing value: %s (%s)", err, msg.String())
				return
			}

			busMixFader[id] = value
		}
	}
}

func Fade(client *osc.ServerAndClient, address string, start float32, target float32, duration time.Duration) {
	stop := time.Now().Add(duration)
	interval := 1 * time.Second / FPS
	ticks := float32(math.Floor(float64(duration / interval)))
	ticker := time.NewTicker(interval)

	increment := (target - start) / ticks

	log.Printf("Fading %s from %3f to %3f over %1f seconds", address, start, target, float64(duration)/float64(time.Second))

	value := start

	for range ticker.C {
		if time.Now().After(stop) {
			break
		}

		value += increment
		if target > start && value >= target {
			break
		} else if target < start && value <= target {
			break
		}

		msg := osc.NewMessage(address, value)
		err := client.Send(msg)
		if err != nil {
			log.Printf("failed sending OSC message: %s", err)
		}
	}
	ticker.Stop()

	msg := osc.NewMessage(address, target)
	err := client.Send(msg)
	if err != nil {
		log.Printf("failed sending OSC message: %s", err)
	}
}
