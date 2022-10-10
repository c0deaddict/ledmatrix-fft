package server

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	WIDTH  = 64
	HEIGHT = 8
)

type fft struct {
	conn net.Conn
	cmd  *exec.Cmd
}

func newImage() []byte {
	return make([]byte, WIDTH*HEIGHT)
}

// Convert image to a XBM bitmap.
func makeFrame(image []byte) []byte {
	frame := make([]byte, WIDTH*HEIGHT/8)
	for i := 0; i < WIDTH*HEIGHT; i++ {
		frame[i/8] |= image[i] << (i % 8)
	}
	return frame
}

func openFFT(target string) (*fft, error) {
	conn, err := net.Dial("udp4", target)
	if err != nil {
		return nil, fmt.Errorf("dial udp: %v", err)
	}

	return &fft{conn, nil}, nil
}

func (f *fft) isOn() bool {
	return f.cmd != nil
}

func (f *fft) start() error {
	if f.cmd != nil {
		return nil
	}

	cmd := exec.Command("cava", "-p", "cava.config")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %v", err)
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cava start: %v", err)
	}
	f.cmd = cmd

	go func() {
		reader := bufio.NewReader(stdout)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				log.Error().Err(err).Msg("cava read error")
				break
			}

			columns := strings.Split(string(line), ";")
			image := newImage()
			for i := 0; i < 64; i++ {
				height, _ := strconv.Atoi(columns[i])
				for j := 0; j < height; j++ {
					image[i+WIDTH*(7-j)] = 1
				}
			}
			f.conn.Write(makeFrame(image))
		}

		state, _ := cmd.Process.Wait()
		if state.ExitCode() != 0 {
			log.Error().Msgf("cava exited with %d", state.ExitCode())
		}
	}()

	return nil
}

func (f *fft) stop() error {
	if f.cmd == nil {
		return nil
	}

	err := f.cmd.Process.Kill()
	if err != nil {
		return fmt.Errorf("kill process: %v", err)
	}

	f.cmd = nil

	// Clear image.
	f.conn.Write(makeFrame(newImage()))

	return nil
}
