package main

import (
	"fmt"
	"io"
	"net"
	"net/url"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/pbergman/logger"
)

type MQTTLogger struct {
	inner io.Writer
}

func (m *MQTTLogger) Println(v ...interface{}) {
	_, _ = fmt.Fprint(m.inner, v...)
}

func (m *MQTTLogger) Printf(format string, v ...interface{}) {
	_, _ = fmt.Fprintf(m.inner, format, v...)
}

func NewClient(app *App) (*autopaho.ConnectionManager, error) {

	var uri = app.config.Server
	var vars = uri.Query()

	var cnf = autopaho.ClientConfig{
		SessionExpiryInterval:         60, // If connection drops we want session to remain live whilst we reconnect
		CleanStartOnInitialConnection: true,
		KeepAlive:                     20,
		ServerUrls: []*url.URL{{
			Host:   uri.Host,
			Scheme: uri.Scheme,
		}},
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			app.GetLogger().Debug("publish: mqtt connection up")
		},
		OnConnectError: func(err error) {
			app.GetLogger().Error(fmt.Sprintf("publish: error whilst attempting connection: %s", err))
		},

		Debug:      &MQTTLogger{app.GetLogger().NewWriter(logger.Debug)},
		Errors:     &MQTTLogger{app.GetLogger().NewWriter(logger.Error)},
		PahoDebug:  &MQTTLogger{app.GetLogger().NewWriter(logger.Debug)},
		PahoErrors: &MQTTLogger{app.GetLogger().NewWriter(logger.Error)},

		ClientConfig: paho.ClientConfig{
			OnClientError: func(err error) {
				app.GetLogger().Error(fmt.Sprintf("publish: client error: %s", err))
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					app.GetLogger().Notice(fmt.Sprintf("publish: server requested disconnect: %s", d.Properties.ReasonString))
				} else {
					app.GetLogger().Notice(fmt.Sprintf("publish:server requested disconnect; reason code: %d", d.ReasonCode))
				}
			},
		},
	}

	if vars.Has("client_id") {
		cnf.ClientID = vars.Get("client_id")
	}

	if nil != uri.User {
		if "" != uri.User.Username() {
			cnf.ConnectUsername = uri.User.Username()
		}
		if pwd, ok := uri.User.Password(); ok {
			cnf.ConnectPassword = []byte(pwd)
		}
	}

	return autopaho.NewConnection(app.ctx, cnf)
}

func getTopicContext(name string, protocol string) url.Values {
	return url.Values{
		"interface": []string{name},
		"protocol":  []string{protocol},
	}
}

func publish(app *App, ip net.IP, name string, protocol string, topic *Topic) error {

	if err := app.client.AwaitConnection(app.ctx); err != nil {
		return err
	}

	_, err := app.client.Publish(app.ctx, &paho.Publish{
		QoS:     1,
		Topic:   topic.GetPath(getTopicContext(name, protocol)),
		Payload: []byte(ip.String()),
		Retain:  true,
	})

	if err != nil {
		return err
	}

	return nil
}
