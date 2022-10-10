package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Server struct {
	alwaysOn bool
	hostname string
	udpPort  int

	mu       sync.Mutex
	listener net.Listener
	clients  []net.Conn

	enabled   bool
	isPlaying bool
	fft       *fft
	spotify   *spotifyWatcher
}

func New(socketPath string, hostname string, udpPort int, alwaysOn bool) (*Server, error) {
	s := Server{
		alwaysOn:  alwaysOn,
		hostname:  hostname,
		udpPort:   udpPort,
		enabled:   true,
		isPlaying: false,
	}

	err := os.MkdirAll(path.Dir(socketPath), os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("ensure socket path parent dirs: %v", err)
	}

	err = os.RemoveAll(socketPath)
	if err != nil {
		return nil, fmt.Errorf("unlink socket: %v", err)
	}

	s.listener, err = net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen error: %v", err)
	}

	return &s, nil
}

func (s *Server) Run() error {
	defer s.listener.Close()

	var err error
	s.fft, err = openFFT(fmt.Sprintf("%s:%d", s.hostname, s.udpPort))
	if err != nil {
		return fmt.Errorf("open fft: %v", err)
	}
	defer s.stop()
	s.updateState()

	log.Info().Msg("watching spotify events")
	s.spotify, err = newSpotifyWatcher()
	if err != nil {
		return fmt.Errorf("watch spotify: %v", err)
	}
	go s.processSpotifyEvents()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return fmt.Errorf("accept error: %v", err)
		}

		go s.clientLoop(conn)
	}
}

func (s *Server) removeClient(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.clients {
		if c == conn {
			log.Info().Msgf("disconnecting client %v", conn)
			s.clients = append(s.clients[:i], s.clients[i+1:]...)
			return
		}
	}
}

func (s *Server) clientLoop(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Error().Err(err).Msg("client read error")
			}
			s.removeClient(conn)
			conn.Close()
			return
		}

		command := strings.TrimSpace(line)
		err = s.executeCommand(command)
		response := "ok"
		if err != nil {
			log.Error().Err(err).Msgf("command: '%s'", command)
			response = err.Error()
		}
		response += "\n"
		conn.Write([]byte(response))
	}
}

func (s *Server) executeCommand(command string) error {
	if command == "enable" {
		s.enable()
		return nil
	} else if command == "disable" {
		s.disable()
		return nil
	} else if strings.HasPrefix(command, "message ") {
		parts := strings.SplitN(command, " ", 3)
		if len(parts) != 3 {
			return errors.New("expected 2 parameters")
		}

		showTime, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("failed to parse showTime: %v", err)
		}

		message := parts[2]
		return s.postMessage(message, time.Duration(showTime)*time.Millisecond)
	} else {
		return errors.New("unknown command")
	}
}

func (s *Server) processSpotifyEvents() {
	for e := range s.spotify.out {
		s.mu.Lock()
		if e.status != nil {
			s.isPlaying = *e.status == "Playing"
			s.updateState()
		}
		if e.text != nil && s.isPlaying {
			s.postMessage(*e.text, 7500*time.Millisecond)
		}
		s.mu.Unlock()
	}
}

func (s *Server) enable() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.enabled = true
	s.updateState()
}

func (s *Server) disable() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.enabled = false
	s.updateState()
}

func (s *Server) updateState() {
	if s.enabled && (s.alwaysOn || s.isPlaying) {
		s.start()
	} else {
		s.stop()
	}
}

func (s *Server) start() {
	if !s.fft.isOn() {
		log.Info().Msg("starting fft")
		err := s.fft.start()
		if err != nil {
			log.Error().Err(err).Msg("start fft")
		}
	}
}

func (s *Server) stop() {
	if s.fft.isOn() {
		log.Info().Msg("stopping fft")
		err := s.fft.stop()
		if err != nil {
			log.Error().Err(err).Msg("stop fft")
		}
	}
}

func (s *Server) postMessage(text string, showTime time.Duration) error {
	resp, err := http.PostForm(fmt.Sprintf("http://%s/message", s.hostname),
		url.Values{"text": {text}, "showTime": {strconv.Itoa(int(showTime.Milliseconds()))}})
	if err != nil {
		return fmt.Errorf("post message http request: %v", err)
	}

	_, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("post message response: %v", err)
	}
	return nil
}
