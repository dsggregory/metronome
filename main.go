package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/sys/unix"

	"github.com/faiface/beep/effects"

	"github.com/faiface/beep/wav"

	"github.com/faiface/beep"

	"github.com/faiface/beep/speaker"
)

const (
	ClickBeat1 = "clicks/Perc_Clap_hi.wav"
	ClickBeat2 = "clicks/Perc_Clap_lo.wav"
)

const (
	RateIncrease = 10
	RateAdjMin   = 20
	RateAdjMax   = 200
)

type config struct {
	// rate beats per minute. The rate of a measure.
	rate            int
	beatsPerMeasure int // 4/(4)
	// mrate increase rate by some factor every specified measures
	mrate int
	// rateKeys increase or decrease rate using keyboard keys. Example: "12" increases rate when 1 is pressed and decreases rate when 2 is pressed.
	rateKeys       string
	volumeIncrease float64
}

type Beats struct {
	src    string
	buffer *beep.Buffer
	format beep.Format
}

func main() {
	cfg := config{
		rate:            120,
		beatsPerMeasure: 4,
	}

	flag.IntVar(&cfg.rate, "rate", 60, "specify the rate in beats per minute")
	flag.IntVar(&cfg.beatsPerMeasure, "tsig", 4, "specify the beats per measure")
	flag.IntVar(&cfg.mrate, "mrate", 0, "increase rate every 'mrate' measures up to a rate of 200 max")
	flag.StringVar(&cfg.rateKeys, "rate-keys", ", ", "increase or decrease rate using keyboard keys. Example: \"12\" increases rate when 1 is pressed and decreases rate when 2 is pressed")
	flag.Float64Var(&cfg.volumeIncrease, "vol", 0, "increase/decrease volume logarithmically, pos or neg")
	flag.Parse()

	beats := []Beats{
		{src: ClickBeat2},
		{src: ClickBeat1},
		{src: ClickBeat1},
		{src: ClickBeat1},
	}
	for i := range beats {
		f, err := os.Open(beats[i].src)
		if err != nil {
			log.Fatal(err)
		}
		ostr, format, err := wav.Decode(f)
		if err != nil {
			log.Fatal(err)
		}
		volume := &effects.Volume{
			Streamer: ostr,
			Base:     2,
			Volume:   cfg.volumeIncrease,
			Silent:   false,
		}
		streamer := beep.ResampleRatio(4, 1, volume)
		buffer := beep.NewBuffer(format)
		buffer.Append(streamer)
		_ = ostr.Close()

		beats[i].buffer = buffer
		beats[i].format = format
	}

	// shortening sampleRate.N at higher BPMs makes for a smoother stream of ticks
	if err := speaker.Init(beats[0].format.SampleRate, beats[0].format.SampleRate.N(time.Second/time.Duration(cfg.rate))); err != nil {
		log.Println(err.Error())
		return
	}

	// keypress rate change
	ostate, kbRate, err := handleKeypressRate(cfg.rateKeys)
	if err != nil {
		log.Println("Error: unable to make raw term.", err.Error())
	} else {
		defer reset(int(os.Stdin.Fd()), ostate)
	}

	// interval per beat based on BPM
	tick := newTicker(cfg.rate)
	defer tick.Stop()

	nBeat := 0
	measure := 1
	for {
		select {
		case t := <-tick.C:
			if nBeat == cfg.beatsPerMeasure {
				nBeat = 0
				measure += 1
				if cfg.mrate > 0 && ((measure+1)%cfg.mrate) == 0 && cfg.rate < RateAdjMax {
					tick.Stop()
					cfg.rate += RateIncrease
					tick = newTicker(cfg.rate)
				}
			}
			fmt.Printf("%s Beat %d.%d\n", t.Format("15:04:05.000000"), measure, nBeat)
			tik := beats[nBeat].buffer.Streamer(0, beats[nBeat].buffer.Len())
			speaker.Play(tik)
			nBeat += 1
		case rateAdj := <-kbRate:
			var newRate int
			if rateAdj == KeypressDecrRate {
				newRate = cfg.rate - RateIncrease
			} else {
				newRate = cfg.rate + RateIncrease
			}
			if newRate >= RateAdjMin && newRate <= RateAdjMax {
				tick.Stop()
				cfg.rate = newRate
				tick = newTicker(cfg.rate)
			}
		}
	}
}

func newTicker(rate int) *time.Ticker {
	f := float32(rate) / float32(60)
	d := float32(time.Second) / f
	println("BPM: rate", rate, "dur", time.Duration(d))
	return time.NewTicker(time.Duration(d))
}

const (
	KeypressIncrRate = iota
	KeypressDecrRate
)

func handleKeypressRate(keys string) (*unix.Termios, chan int, error) {
	rateKeys := make([]byte, 2)
	rateKeys[0] = keys[0]
	rateKeys[1] = keys[1]

	ostate, err := makeRaw(int(os.Stdin.Fd()))

	kbRate := make(chan int)
	go func() {
		var b = make([]byte, 1)
		for {
			_, err := os.Stdin.Read(b) // read one byte from raw terminal
			if err != nil {
				log.Fatal(err.Error())
				return
			}
			log.Println("read keypress ", b[0])
			switch b[0] {
			case rateKeys[0]:
				kbRate <- KeypressDecrRate
			case rateKeys[1]:
				kbRate <- KeypressIncrRate
			default:
				log.Println("unhandled keypress", b[0])
			}
		}
	}()
	return ostate, kbRate, err
}

// our own b/c we don't want to muck with output or signal reception
func makeRaw(fd int) (*unix.Termios, error) {
	termios, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		return nil, err
	}

	oldState := *termios

	// This attempts to replicate the behaviour documented for cfmakeraw in
	// the termios(3) manpage.
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	//termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | /*unix.ISIG |*/ unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, unix.TIOCSETA, termios); err != nil {
		return nil, err
	}

	return &oldState, nil
}

func reset(fd int, termios *unix.Termios) error {
	return unix.IoctlSetTermios(fd, unix.TIOCSETA, termios)
}
