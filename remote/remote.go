package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/abiosoft/ishell"
	"github.com/biribiribiri/sd400"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var rpitx = flag.String("rpitx", os.Getenv("HOME")+"/src/rpitx/rpitx", "path to rpitx")
var wavOutputPath = flag.String("wavpath", "", "folder to store wav files to send to rpitx")

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

	remote := sd400.New(sd400.REMOTE1, *rpitx, *wavOutputPath)

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
			remote.Beep(d)
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
			remote.Nick(level)
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
			remote.Shock(level, d)
		},
	})

	shell.Println("SD400 remote by biribiribiri. Type \"help\" to get a list of commands.")
	shell.Run()

	// cmd := continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
	// 	continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL1, 1*time.Second) +
	// 	continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
	// 	continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL2, 2*time.Second) +
	// 	continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
	// 	continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL3, 3*time.Second) +
	// 	continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
	// 	continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL4, 4*time.Second) +
	// 	continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
	// 	continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL5, 5*time.Second) +
	// 	continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
	// 	continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL6, 6*time.Second) +
	// 	continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
	// 	continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL7, 7*time.Second) +
	// 	continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
	// 	continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL8, 15*time.Second) +
	// 	momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL1) +
	// 	momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL2) +
	// 	momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL3) +
	// 	momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL4) +
	// 	momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL5) +
	// 	momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL6) +
	// 	momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL7) +
	// 	momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL8) +
	// 	continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second)
	// sendCmd(cmd)
}
