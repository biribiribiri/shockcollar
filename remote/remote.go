package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/abiosoft/ishell"
	"github.com/biribiribiri/sd400"
	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var rpitx = flag.String("rpitx", os.Getenv("HOME")+"/src/rpitx/rpitx", "path to rpitx")
var wavOutputPath = flag.String("wavpath", "", "folder to store wav files to send to rpitx")
var grpcPort = flag.String("port", ":50051", "Port for GRPC server to listen on")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	collar := sd400.New(sd400.REMOTE1, *rpitx, *wavOutputPath)

	// Start gRPC server.
	lis, err := net.Listen("tcp", *grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	sd400.RegisterCollarServer(s, &collar)
	reflection.Register(s)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	router := mux.NewRouter()

	cmdHandler := func(w http.ResponseWriter, r *http.Request) {
		var req sd400.CollarRequest
		err = jsonpb.Unmarshal(r.Body, &req)
		log.Println(err)
		collar.SendCommand(context.Background(), &req)
	}
	fs := http.FileServer(http.Dir("static"))
	router.Handle("/", fs)
	router.HandleFunc("/cmd", cmdHandler).Methods("POST")

	go func() {
		log.Fatal(http.ListenAndServe(":8000", router))
	}()

	shell := ishell.New()

	shell.AddCmd(&ishell.Cmd{
		Name: "beep",
		Help: "Send a beep command. Ex: beep 1s",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("expected 1 argument")
				return
			}
			d, err := time.ParseDuration(c.Args[0])
			if err != nil {
				c.Println(err)
				return
			}
			collar.Beep(d)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "nick",
		Help: "Send a nick command. Ex: nick 3",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("expected 1 argument")
				return
			}
			level, err := strconv.Atoi(c.Args[0])
			if err != nil {
				c.Println(err)
				return
			}
			collar.Nick(level)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "shock",
		Help: "Send a continuous shock command. Ex: shock 3 5s",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 2 {
				c.Println("expected 2 argument")
				return
			}
			level, err := strconv.Atoi(c.Args[0])
			if err != nil {
				c.Println(err)
				return
			}
			d, err := time.ParseDuration(c.Args[1])
			if err != nil {
				c.Println(err)
				return
			}
			collar.Shock(level, d)
		},
	})

	shell.Println("SD400 remote by biribiribiri. Type \"help\" to get a list of commands.")
	shell.Run()

}
