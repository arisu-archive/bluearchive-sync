package adbx

import (
	"fmt"
	"net"
	"strconv"

	"github.com/arisu-archive/bluearchive-data-sync/pkg/gadb"
)

type Client struct {
	client gadb.Client
}

func NewClient() (*Client, error) {
	client, err := gadb.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create adb client: %w", err)
	}

	return &Client{
		client: client,
	}, nil
}

func (c *Client) Connect(host string) error {
	ip, portString, err := net.SplitHostPort(host)
	if err != nil {
		return fmt.Errorf("failed to split host and port: %w", err)
	}

	port, err := strconv.Atoi(portString)
	if err != nil {
		return fmt.Errorf("failed to convert port to int: %w", err)
	}

	if err := c.client.Connect(ip, port); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	return nil
}

func (c *Client) Disconnect(host string) error {
	ip, portString, err := net.SplitHostPort(host)
	if err != nil {
		return fmt.Errorf("failed to split host and port: %w", err)
	}

	port, err := strconv.Atoi(portString)
	if err != nil {
		return fmt.Errorf("failed to convert port to int: %w", err)
	}

	if err := c.client.Disconnect(ip, port); err != nil {
		return fmt.Errorf("failed to disconnect: %w", err)
	}

	return nil
}

func (c *Client) GetDevice(serial string) (Device, error) {
	devices, err := c.client.DeviceList()
	if err != nil {
		return Device{}, fmt.Errorf("failed to get device list: %w", err)
	}

	for _, device := range devices {
		if device.Serial() == serial {
			return Device{Device: device}, nil
		}
	}

	return Device{}, fmt.Errorf("device not found: %s", serial)
}

func (c *Client) Shutdown() error {
	if err := c.client.KillServer(); err != nil {
		return fmt.Errorf("failed to kill adb server: %w", err)
	}

	return nil
}
