package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	conn net.Conn
}

func New(socketPath string) (*Client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("connect to server: %v", err)
	}

	return &Client{conn}, nil
}

func MakeMessageCommand(message string, showTime time.Duration) string {
	return strings.Join([]string{
		"message",
		strconv.Itoa(int(showTime.Milliseconds())),
		message,
	}, " ")
}

func (c *Client) Send(command string) error {
	_, err := c.conn.Write([]byte(command + "\n"))
	if err != nil {
		return err
	}

	reader := bufio.NewReader(c.conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		if err != io.EOF {
			fmt.Printf("client got disconnected: %v", err)
		}
	} else {
		os.Stdout.Write([]byte(line))
	}

	return nil
}

func (c *Client) Close() {
	c.conn.Close()
}
