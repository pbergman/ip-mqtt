package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"unsafe"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/pbergman/logger"
	"gopkg.in/ini.v1"
)

type Config struct {
	Server    *url.URL
	Interface []string
	Debug     bool
	Workers   int
}

type App struct {
	config     *Config
	configFile string
	topic      *Topic
	logger     *logger.Logger
	client     *autopaho.ConnectionManager
	endianness binary.ByteOrder

	stop context.CancelFunc
	ctx  context.Context
}

func (a *App) GetLogger() *logger.Logger {

	if nil == a.logger {

		if nil == a.config {
			return getLogger(false)
		}

		a.logger = getLogger(a.config.Debug)
	}

	return a.logger
}

func bootstrap(file string) (*App, error) {

	var app = &App{
		configFile: file,
		config: &Config{
			Interface: make([]string, 0),
			Workers:   2,
		},
		endianness: getEndianness(),
	}

	if nil == app.endianness {
		return app, errors.New("could not determine native endianness")
	}

	cfg, err := ini.LoadSources(ini.LoadOptions{AllowShadows: true, Insensitive: true}, app.configFile)

	if err != nil {
		return app, err
	}

	cfg.BlockMode = false

	if err := cfg.MapTo(&app.config); err != nil {
		return nil, err
	}

	section := cfg.Section(ini.DefaultSection)

	if section == nil {
		return app, errors.New("invalid config file")
	}

	if err := bootstrapServer(section, app); err != nil {
		return app, err
	}

	if err := bootstrapTopic(section, app); err != nil {
		return app, err
	}

	if err := validateInterfaces(app); err != nil {
		return app, err
	}

	app.ctx, app.stop = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	client, err := NewClient(app)

	if err != nil {
		return app, err
	}

	app.client = client

	if err := registerInterfaces(app); err != nil {
		return app, err
	}

	return app, nil
}

func getDefaultClientId(ctx *TopicDefaultCtx) string {
	var b = make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s_%x", ctx.Hostname, b)
}

func bootstrapTopic(section *ini.Section, app *App) error {
	defaults, err := getTopicsDefaults()

	if err != nil {
		return fmt.Errorf("could not create topic defaults")
	}

	var query = app.config.Server.Query()

	if false == query.Has("client_id") {
		query.Set("client_id", getDefaultClientId(defaults))
		app.config.Server.RawQuery = query.Encode()
	}

	var topicPath string

	if section.HasKey("mqtt_topic") {
		topicPath = section.Key("mqtt_topic").String()
	}

	if topicPath == "" {
		topicPath = DefaultTopic
	}

	app.topic = NewTopic(topicPath, defaults)

	return nil
}

func bootstrapServer(section *ini.Section, app *App) error {
	if x := section.Key("server").String(); x != "" {
		uri, err := url.Parse(x)

		if err != nil {
			return fmt.Errorf("could not parse mqtt uri from config file: %s", err)
		}

		if uri.Port() == "" {
			uri.Host = uri.Host + ":1883"
		}

		app.config.Server = uri
	}

	if app.config.Server == nil {
		return errors.New("no valid mqtt server found in config")
	}

	return nil
}

func getEndianness() binary.ByteOrder {
	var buf [2]byte

	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)

	switch buf {
	case [2]byte{0xCD, 0xAB}:
		return binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		return binary.BigEndian
	default:
		return nil
	}
}

func getLogger(debug bool) *logger.Logger {

	var handler logger.HandlerInterface

	if debug {
		handler = logger.NewWriterHandler(os.Stdout, logger.LogLevelDebug(), true)
	} else {
		handler = logger.NewThresholdHandler(logger.NewWriterHandler(os.Stderr, logger.LogLevelDebug(), true), 20, logger.LogLevelError(), true)
	}

	return logger.NewLogger("app", handler)
}
