// Package mqtt connects to the MQTT broker, subscribes to the lock's
// Zigbee2MQTT topics, feeds incoming state into the lock store and implements
// the lock.Publisher interface for outgoing commands.
package mqtt

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/Donkie/nimly-panel/backend/internal/lock"
)

// Options configures the MQTT client.
type Options struct {
	BrokerURL string
	Username  string
	Password  string
	ClientID  string
	LockTopic string // Z2M base topic, e.g. "zigbee2mqtt/Front Door Lock"
}

// Client wraps a paho MQTT client bound to the lock store.
type Client struct {
	c        mqtt.Client
	opts     Options
	store    *lock.Store
	log      *slog.Logger
	svc      *lock.Service
	firstMsg sync.Once
}

// New creates (but does not connect) an MQTT client. The lock.Service is wired
// later via Bind so that RefreshPins can run on connect.
func New(opts Options, store *lock.Store, log *slog.Logger) *Client {
	cl := &Client{opts: opts, store: store, log: log}

	// Surface paho's internal connection/auth diagnostics via slog.
	enablePahoLogging(log)

	mo := mqtt.NewClientOptions().
		AddBroker(opts.BrokerURL).
		SetClientID(opts.ClientID).
		SetUsername(opts.Username).
		SetPassword(opts.Password).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetKeepAlive(30 * time.Second).
		SetOnConnectHandler(cl.onConnect).
		SetConnectionLostHandler(cl.onConnectionLost).
		SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
			cl.log.Info("mqtt reconnecting", "broker", opts.BrokerURL)
		}).
		SetConnectionAttemptHandler(func(broker *url.URL, tlsCfg *tls.Config) *tls.Config {
			cl.log.Info("mqtt connection attempt", "broker", broker.String(), "username", opts.Username, "client_id", opts.ClientID)
			return tlsCfg
		})

	cl.c = mqtt.NewClient(mo)
	return cl
}

// Bind attaches the high-level service so the client can trigger an initial
// read of PIN slots / constraints once connected.
func (c *Client) Bind(svc *lock.Service) { c.svc = svc }

// Connect establishes the broker connection (non-blocking thanks to
// SetConnectRetry).
func (c *Client) Connect() error {
	t := c.c.Connect()
	t.Wait()
	return t.Error()
}

// Disconnect cleanly closes the connection.
func (c *Client) Disconnect() {
	c.c.Disconnect(250)
}

// Publish sends a payload to a topic at QoS 1. Implements lock.Publisher.
func (c *Client) Publish(topic string, payload []byte) error {
	if !c.c.IsConnected() {
		return fmt.Errorf("mqtt not connected")
	}
	t := c.c.Publish(topic, 1, false, payload)
	t.Wait()
	return t.Error()
}

func (c *Client) onConnect(_ mqtt.Client) {
	c.log.Info("mqtt connected", "broker", c.opts.BrokerURL)
	c.store.ApplyState(func(s *lock.State) { s.BrokerConnected = true })

	stateTopic := c.opts.LockTopic
	availTopic := c.opts.LockTopic + "/availability"

	if t := c.c.Subscribe(stateTopic, 1, c.onState); t.Wait() && t.Error() != nil {
		c.log.Error("subscribe state failed", "topic", stateTopic, "err", t.Error())
	} else {
		c.log.Info("subscribed to lock state", "topic", stateTopic)
	}
	if t := c.c.Subscribe(availTopic, 1, c.onAvailability); t.Wait() && t.Error() != nil {
		c.log.Error("subscribe availability failed", "topic", availTopic, "err", t.Error())
	} else {
		c.log.Info("subscribed to lock availability", "topic", availTopic)
	}

	// Prime the cache with constraints and the PIN table.
	if c.svc != nil {
		go func() {
			time.Sleep(500 * time.Millisecond)
			if err := c.svc.RefreshConstraints(); err != nil {
				c.log.Warn("refresh constraints failed", "err", err)
			} else {
				c.log.Info("requested lock state/constraints", "topic", c.opts.LockTopic+"/get")
			}
			if err := c.svc.RefreshPins(); err != nil {
				c.log.Warn("refresh pins failed", "err", err)
			}
		}()
	}
}

func (c *Client) onConnectionLost(_ mqtt.Client, err error) {
	c.log.Warn("mqtt connection lost", "err", err)
	c.store.ApplyState(func(s *lock.State) {
		s.BrokerConnected = false
		s.Available = false
	})
}

func (c *Client) onState(_ mqtt.Client, m mqtt.Message) {
	c.firstMsg.Do(func() {
		c.log.Info("received first lock message", "topic", m.Topic(), "bytes", len(m.Payload()))
	})
	c.log.Debug("lock state message", "topic", m.Topic(), "payload", string(m.Payload()))
	parsed, err := lock.Parse(m.Payload())
	if err != nil {
		c.log.Warn("parse lock state failed", "topic", m.Topic(), "err", err, "payload", string(m.Payload()))
		return
	}
	c.store.Ingest(parsed)
}

func (c *Client) onAvailability(_ mqtt.Client, m mqtt.Message) {
	payload := strings.ToLower(strings.TrimSpace(string(m.Payload())))
	// Z2M publishes either "online"/"offline" or {"state":"online"}.
	available := strings.Contains(payload, "online") && !strings.Contains(payload, "offline")
	c.log.Info("lock availability", "topic", m.Topic(), "payload", payload, "available", available)
	c.store.ApplyState(func(s *lock.State) { s.Available = available })
}
