package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Each access point has their own name. The terminals can identify
// by that name.
type Target string // TODO: find better name for this type
const (
	TargetDownstairs = Target("gate")
	TargetUpstairs   = Target("upstairs")
	TargetElevator   = Target("elevator")
	TargetControlUI  = Target("control") // UI to add new users.
)

const (
	maxLCDRows                  = 2
	maxLCDCols                  = 24
	defaultBaudrate             = 9600
	initialReconnectOnErrorTime = 2 * time.Second
	maxReconnectOnErrorTime     = 60 * time.Second
	idleTickTime                = 500 * time.Millisecond
)

func parseArg(arg string) (devicepath string, baudrate int) {
	split := strings.Split(arg, ":")
	devicepath = split[0]
	baudrate = defaultBaudrate
	if len(split) > 1 {
		var err error
		if baudrate, err = strconv.Atoi(split[1]); err != nil {
			panic(err)
		}
	}
	return
}

type Backends struct {
	authenticator Authenticator
	appEventBus   *ApplicationBus
}

func handleSerialDevice(devicepath string, baud int, backends *Backends) {
	var t *SerialTerminal
	connect_successful := true
	retry_time := initialReconnectOnErrorTime
	for {
		if !connect_successful {
			time.Sleep(retry_time)
			retry_time *= 2 // exponential backoff.
			if retry_time > maxReconnectOnErrorTime {
				retry_time = maxReconnectOnErrorTime
			}
		}

		connect_successful = false

		t, _ = NewSerialTerminal(devicepath, baud)
		if t == nil {
			continue
		}

		// Terminals are dispatched by name. There are different handlers
		// for the name e.g. handlers that deal with reading codes
		// and opening doors, but also the UI handler dealing with
		// adding new users.
		var handler TerminalEventHandler
		switch Target(t.GetTerminalName()) {
		case TargetDownstairs, TargetUpstairs, TargetElevator:
			handler = NewAccessHandler(backends)

		case TargetControlUI:
			handler = NewControlHandler(backends)

		default:
			log.Printf("%s:%d: Terminal with unrecognized name '%s'",
				devicepath, baud, t.GetTerminalName())
		}

		if handler != nil {
			connect_successful = true
			retry_time = initialReconnectOnErrorTime
			log.Printf("%s:%d: connected to '%s'",
				devicepath, baud, t.GetTerminalName())
			backends.appEventBus.Post(&AppEvent{
				Ev:     AppTerminalConnect,
				Target: Target(t.GetTerminalName()),
				Msg:    fmt.Sprintf("%s:%d", devicepath, baud),
				Source: "serialdevice",
			})
			t.RunEventLoop(handler, backends.appEventBus)
			backends.appEventBus.Post(&AppEvent{
				Ev:     AppTerminalDisconnect,
				Target: Target(t.GetTerminalName()),
				Msg:    fmt.Sprintf("%s:%d", devicepath, baud),
				Source: "serialdevice",
			})
		}
		t.shutdown()
		t = nil
	}
}

func main() {
	userFileName := flag.String("users", "/var/access/users.csv", "User Authentication file.")
	logFileName := flag.String("logfile", "", "The log file, default = stdout")
	doorbellDir := flag.String("belldir", "", "Directory that contains upstairs.wav, gate.wav etc. Wav needs to be named like")
	httpPort := flag.Int("httpport", -1, "Port to listen HTTP requests on")

	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Fprintf(os.Stderr,
			"usage: %s [options] <serial-device>[:baudrate] [<serial-device>[:baudrate]...]\nOptions\n",
			os.Args[0])
		flag.PrintDefaults()
		return
	}

	if *logFileName != "" {
		logfile, err := os.OpenFile(*logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal("Error opening log file", err)
		}
		defer logfile.Close()
		log.SetOutput(logfile)
	}

	log.Println("Starting...")

	appEventBus := NewApplicationBus()
	actions := NewGPIOActions(*doorbellDir)
	go actions.EventLoop(appEventBus)

	authenticator := NewFileBasedAuthenticator(*userFileName,
		appEventBus)
	backends := &Backends{
		authenticator: authenticator,
		appEventBus:   appEventBus,
	}

	if authenticator == nil {
		log.Fatal("Can't continue without authenticator.")
	}

	// For each serial interface, we run an indepenent loop
	// making sure we are constantly connected.
	for _, arg := range flag.Args() {
		devicepath, baudrate := parseArg(arg)
		go handleSerialDevice(devicepath, baudrate, backends)
	}

	if *httpPort > 0 && *httpPort <= 65535 {
		apiServer := NewApiServer(appEventBus, *httpPort)
		go apiServer.Run()
	}

	var block_forever chan bool
	<-block_forever
}
