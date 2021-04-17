package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	WIDTH  = 64
	HEIGHT = 8
)

type PlaybackEvent struct {
	status string
	text   string
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

func fft(conn net.Conn) {
	cmd := exec.Command("cava", "-p", "cava.config")
	stdout, err := cmd.StdoutPipe()

	if err != nil {
		log.Fatal(err)
	}

	cmd.Start()

	buf := bufio.NewReader(stdout)

	for {
		line, _, _ := buf.ReadLine()
		columns := strings.Split(string(line), ";")
		image := newImage()
		for i := 0; i < 64; i++ {
			height, _ := strconv.Atoi(columns[i])
			for j := 0; j < height; j++ {
				image[i+WIDTH*(7-j)] = 1
			}
		}
		conn.Write(makeFrame(image))
	}
}

func startFft() {
	conn, err := net.Dial("udp4", "ledmatrix:1337")
	if err != nil {
		log.Fatalln("Udp dial:", err)
	}

	go fft(conn)
}

func processSpotifyEvents(conn *dbus.Conn, out chan PlaybackEvent) {
	if err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/mpris/MediaPlayer2"),
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchSender("org.mpris.MediaPlayer2.spotify"),
	); err != nil {
		panic(err)
	}

	ch := make(chan *dbus.Signal, 10)
	conn.Signal(ch)

	var last *PlaybackEvent
	for msg := range ch {
		if msg.Body[0].(string) != "org.mpris.MediaPlayer2.Player" {
			continue
		}
		body := msg.Body[1].(map[string]dbus.Variant)
		status := body["PlaybackStatus"].Value().(string)
		metadata := body["Metadata"].Value().(map[string]dbus.Variant)
		artists := metadata["xesam:artist"].Value().([]string)
		title := metadata["xesam:title"].Value().(string)
		text := strings.Join(artists, " & ") + " - " + title
		event := PlaybackEvent{status, text}
		if last == nil || event != *last {
			last = &event
			out <- event
		}
	}
}

func postMessage(text string) {
	resp, err := http.PostForm("http://ledmatrix/message",
		url.Values{"text": {text}, "showTime": {"7500"}})
	if err != nil {
		log.Println("Error posting message:", err)
	} else {
		_, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Println("Error reading post message response:", err)
		}
	}
}

func main() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatalln("Failed to connect to session bus:", err)
	}
	defer conn.Close()

	startFft()

	out := make(chan PlaybackEvent)
	go processSpotifyEvents(conn, out)
	for e := range out {
		if e.status == "Playing" {
			postMessage(e.text)
		}
	}
}
