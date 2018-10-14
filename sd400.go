// Software remote for the Sportdog SD400.
//
// Uses rpitx to transmit to the collar. To use, plug a wire on GPIO 4, i.e.
// Pin 7 of the GPIO header (header P1). Note that rpitx requires root.

//go:generate protoc -I ../shockcollar/ --go_out=plugins=grpc:../shockcollar ../shockcollar/shockcollar.proto
package shockcollar

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
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

	CMD_NICK       = "6a"
	CMD_CONTINUOUS = "66"
)

type Sd400 struct {
	symbols       map[rune][]wav.Sample // Map from symbol ('0', '1', '2') to output waveform.
	remoteId      string
	rpitxPath     string
	wavOutputPath string
	sampleRate    int // Output sample rate in Hz
}

func New(remoteId string, rpitxPath string, wavOutputPath string) Sd400 {
	s := Sd400{remoteId: remoteId, rpitxPath: rpitxPath, wavOutputPath: wavOutputPath, sampleRate: 44000}
	s.generateSymbols()
	return s
}

// Sends a beep command for the specified duration.
func (s *Sd400) Beep(duration time.Duration) {
	const CMD_BEEP = "59"
	const ARG_BEEP = "a9"
	s.sendCmd(s.continuousCmd(CMD_BEEP, ARG_BEEP, duration))
}

func (s *Sd400) SendCommand(ctx context.Context, request *CollarRequest) (*CollarResponse, error) {
	log.Println("command: ", request.String())

	switch request.GetType() {
	case CollarRequest_NICK:
		s.Nick(int(request.GetIntensity()))
	case CollarRequest_SHOCK:
		s.Shock(int(request.GetIntensity()), time.Millisecond*time.Duration(request.GetDurationMs()))
	case CollarRequest_BEEP:
		s.Beep(time.Millisecond * time.Duration(request.GetDurationMs()))
	}
	return &CollarResponse{}, nil
}

var shockArgs []string = []string{
	"a9",
	"a6",
	"a5",
	"9a",
	"99",
	"95",
	"6a",
	"55",
}

func (s *Sd400) Nick(level int) {
	if level < 0 {
		level = 0
	}
	if level > len(shockArgs)-1 {
		level = len(shockArgs) - 1
	}
	s.sendCmd(s.momentaryCmd(CMD_NICK, shockArgs[level]))
}

// Sends a continuous stimulation command for the specified intensity and
// duration.
func (s *Sd400) Shock(level int, duration time.Duration) {
	if level < 0 {
		level = 0
	}
	if level > len(shockArgs)-1 {
		level = len(shockArgs) - 1
	}
	s.sendCmd(s.continuousCmd(CMD_CONTINUOUS, shockArgs[level], duration))
}

// hex2bin converts a string containing an arbitrary length of hexadecimal
// nibbles to a binary string representation with no truncation of leading
// zeros.
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

func (s *Sd400) momentaryCmd(cmdType string, cmdArg string) string {
	return fmt.Sprint(CMD_START, PREAMBLE, hex2bin(s.remoteId), hex2bin(cmdType), hex2bin(cmdArg), MOMENTARY_CMD_END)
}

func (s *Sd400) continuousCmd(cmdType string, cmdArg string, duration time.Duration) string {
	cmd := fmt.Sprint(PREAMBLE, hex2bin(s.remoteId), hex2bin(cmdType), hex2bin(cmdArg), CONTINUOUS_CMD_BREAK)
	sampleTime := 4 * time.Millisecond
	cmdTime := time.Duration(len(cmd)) * sampleTime
	repeats := int(duration / cmdTime)
	// In practice, the collar does not respond if there are fewer than 2 of
	// the repeated section.
	if repeats < 2 {
		repeats = 2
	}

	return fmt.Sprint(CMD_START, strings.Repeat(cmd, repeats), CONTINUOUS_CMD_END)
}

func (s *Sd400) generateSymbols() {
	s.symbols = make(map[rune][]wav.Sample)

	const symbolTime = 0.004 // s
	const f1 = 5000

	samplesPerSymbol := symbolTime * float64(s.sampleRate)

	s.symbols['0'] = make([]wav.Sample, int(samplesPerSymbol))
	s.symbols['1'] = make([]wav.Sample, int(samplesPerSymbol))

	for i := 0; i < int(samplesPerSymbol); i++ {
		s.symbols['0'][i].Values[0] = math.MaxInt16
		s.symbols['1'][i].Values[0] = int(math.MaxInt16 * math.Cos(2.0*math.Pi*f1*float64(i)*(1.0/float64(s.sampleRate))))
	}
	// The '2' symbol is the same as '1' but half the length.
	s.symbols['2'] = s.symbols['1'][:len(s.symbols['1'])/2]
}

func (s *Sd400) bin2waveform(binStr string) []wav.Sample {
	glog.V(2).Info("binary waveform: ", binStr)
	var out []wav.Sample
	for _, c := range binStr {
		s, ok := s.symbols[c]
		if !ok {
			log.Fatalln("bin2waveform: invalid symbol", s)
		}
		out = append(out, s...)
	}
	return out
}

func (s *Sd400) sendCmd(binStr string) {
	samples := s.bin2waveform(binStr)
	samples = append(samples, make([]wav.Sample, s.sampleRate/10)...)

	fileName := s.wavOutputPath + "result.wav"
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	fileBuf := bufio.NewWriterSize(file, 10000000)
	wavWriter := wav.NewWriter(fileBuf, uint32(len(samples)), 2, uint32(s.sampleRate), 16)
	if err = wavWriter.WriteSamples(samples); err != nil {
		log.Fatal(err)
	}
	err = fileBuf.Flush()
	if err != nil {
		log.Fatal(err)
	}
	file.Close()

	rpitx := exec.Command(s.rpitxPath, "-m", "IQ", "-i", fileName, "-f", "27255", "-s", strconv.Itoa(s.sampleRate), "-c", "1")
	out, err := rpitx.Output()
	log.Println(string(out), err)
}
