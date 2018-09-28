// Software remote for the Sportdog SD400.
//
// Uses rpitx to transmit to the collar. To use, plug a wire on GPIO 4, i.e.
// Pin 7 of the GPIO header (header P1). Note that rpitx requires root.

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	wav "github.com/youpy/go-wav"
)

const (
	// Command start/end sequences. Specified in "binary".
	// '0' represents 4ms, unmodulated
	// '1' represents 4ms, 5kHz modulation
	// '2' represents 2ms, 5kHz modulation
	//
	// Momentary command format:
	//   [CMD_START] [PREAMBLE] [REMOTE ID] [CMD TYPE] [CMD DATA] [MOMENTARY_CMD_END]
	//
	// Continuous command format:
	//   [CMD_START]
	//   Repeated N times: [PREAMBLE] [REMOTE ID] [CMD TYPE] [CMD DATA] [CONTINUOUS_CMD_BREAK]
	//   [CONTINUOUS_CMD_END]
	CMD_START            = "11111"
	PREAMBLE             = "0001"
	MOMENTARY_CMD_END    = "10"
	CONTINUOUS_CMD_BREAK = "102"
	CONTINUOUS_CMD_END   = "1111111111"

	// Command data sequences.
	// These are all specified in hex.
	REMOTE1 = "6695"
	REMOTE2 = "999a"

	CMD_BEEP       = "59"
	CMD_NICK       = "6a"
	CMD_CONTINUOUS = "66"

	ARG_BEEP   = "a9"
	ARG_LEVEL1 = "a9"
	ARG_LEVEL2 = "a6"
	ARG_LEVEL3 = "a5"
	ARG_LEVEL4 = "9a"
	ARG_LEVEL5 = "99"
	ARG_LEVEL6 = "95"
	ARG_LEVEL7 = "6a"
	ARG_LEVEL8 = "55"
)

func hex2bin(hexStr string) string {
	var out bytes.Buffer
	for _, c := range hexStr {
		nibble, err := strconv.ParseUint(string(c), 16, 4)
		if err != nil {
			log.Fatal(err)
		}
		_, err = fmt.Fprintf(&out, "%04b", nibble)
		if err != nil {
			log.Fatal(err)
		}
	}
	return out.String()
}

func momentaryCmd(remoteId string, cmdType string, cmdArg string) string {
	return fmt.Sprint(CMD_START, PREAMBLE, hex2bin(remoteId), hex2bin(cmdType), hex2bin(cmdArg), MOMENTARY_CMD_END)
}

func continousCmd(remoteId string, cmdType string, cmdArg string, duration time.Duration) string {
	cmd := fmt.Sprint(PREAMBLE, hex2bin(remoteId), hex2bin(cmdType), hex2bin(cmdArg), CONTINUOUS_CMD_BREAK)
	sampleTime := 4 * time.Millisecond
	cmdTime := time.Duration(len(cmd)) * sampleTime
	repeats := int(duration / cmdTime)
	if repeats < 1 {
		repeats = 1
	}

	return fmt.Sprint(CMD_START, strings.Repeat(cmd, repeats), CONTINUOUS_CMD_END)
}

func generateSymbols() map[rune][]wav.Sample {
	symbols := make(map[rune][]wav.Sample)

	const sampleRate = 44000 // Hz
	const symbolTime = 0.004 // s
	const f1 = 5000

	samplesPerSymbol := symbolTime * sampleRate

	symbols['0'] = make([]wav.Sample, int(samplesPerSymbol))
	symbols['1'] = make([]wav.Sample, int(samplesPerSymbol))

	for i := 0; i < int(samplesPerSymbol); i++ {
		symbols['0'][i].Values[0] = math.MaxInt16
		symbols['1'][i].Values[0] = int(math.MaxInt16 * math.Cos(2.0*math.Pi*f1*float64(i)*(1.0/sampleRate)))
	}
	// The '2' symbol is the same as '1' but half the length.
	symbols['2'] = symbols['1'][:len(symbols['1'])/2]
	return symbols
}

func bin2waveform(binStr string, symbols map[rune][]wav.Sample) []wav.Sample {
	log.Println("binstring", binStr)
	var out []wav.Sample
	for _, c := range binStr {
		s, ok := symbols[c]
		if !ok {
			log.Fatalln("bin2waveform: invalid symbol", s)
		}
		out = append(out, s...)
	}
	return out
}

func sendCmd(binStr string) {
	const sampleRate = 44000 // Hz

	symbols := generateSymbols()
	samples := bin2waveform(binStr, symbols)

	fileName := "result.wav"
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	fileBuf := bufio.NewWriterSize(file, 10000000)
	wavWriter := wav.NewWriter(fileBuf, uint32(len(samples)), 2, sampleRate, 16)
	if err = wavWriter.WriteSamples(samples); err != nil {
		log.Fatal(err)
	}
	err = fileBuf.Flush()
	if err != nil {
		log.Fatal(err)
	}
	file.Close()

	rpitx := exec.Command(*rpitx, "-m", "IQ", "-i", fileName, "-f", "27255", "-s", strconv.Itoa(sampleRate), "-c", "1")
	out, err := rpitx.Output()
	log.Println(string(out), err)
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var rpitx = flag.String("rpitx", os.Getenv("HOME")+"/src/rpitx/rpitx", "path to rpitx")

func main() {
	flag.Parse()
	if *rpitx != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	cmd := continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
		continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL1, 1*time.Second) +
		continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
		continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL2, 2*time.Second) +
		continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
		continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL3, 3*time.Second) +
		continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
		continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL4, 4*time.Second) +
		continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
		continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL5, 5*time.Second) +
		continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
		continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL6, 6*time.Second) +
		continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
		continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL7, 7*time.Second) +
		continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second) +
		continousCmd(REMOTE1, CMD_CONTINUOUS, ARG_LEVEL8, 15*time.Second) +
		momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL1) +
		momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL2) +
		momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL3) +
		momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL4) +
		momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL5) +
		momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL6) +
		momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL7) +
		momentaryCmd(REMOTE1, CMD_NICK, ARG_LEVEL8) +
		continousCmd(REMOTE1, CMD_BEEP, ARG_BEEP, 1*time.Second)
	sendCmd(cmd)
}
