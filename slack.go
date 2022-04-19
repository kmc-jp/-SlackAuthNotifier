package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	"github.com/TKMAX777/SlackAuthNotifier/slack_webhook"
	"github.com/ipinfo/go/v2/ipinfo"
)

type MessageType int

const (
	MessageTypeAccept = iota
	MessageTypeFailed
	MessageTypeCaution
)

type SlackMessageHandler struct {
	ipinfoClient *ipinfo.Client
}

type SlackUserSetting struct {
	CustomName string
	SlackID    string
}

func NewSlackMessageHandler(ipinfoClient *ipinfo.Client) *SlackMessageHandler {
	return &SlackMessageHandler{
		ipinfoClient: ipinfoClient,
	}
}

func (s SlackMessageHandler) NewMessage(addr, username, LastLine string, messageType MessageType) slack_webhook.Message {
	var message slack_webhook.Message

	var section = slack_webhook.SectionBlock()
	if messageType == MessageTypeAccept {
		section.Text = slack_webhook.MrkdwnElement(fmt.Sprintf("*%s*", LastLine), false)
	} else {
		section.Text = slack_webhook.MrkdwnElement(LastLine, false)
	}

	var blocks = make([]slack_webhook.BlockBase, 0)
	blocks = append(blocks, section)

	section = slack_webhook.SectionBlock()

	b, err := ioutil.ReadFile(fmt.Sprintf("/home/%s/.slack_notifier", username))
	if err == nil {
		var settings = s.readSettingsFile(string(b))
		if settings.CustomName != "" {
			username = settings.CustomName
		}

		var text = fmt.Sprintf("User: %s", username)

		if settings.SlackID != "" {
			text += fmt.Sprintf("(<@%s>)", settings.SlackID)
		}

		section.Text = slack_webhook.MrkdwnElement(text, false)
	} else {
		section.Text = slack_webhook.MrkdwnElement(fmt.Sprintf("User: %s", username), false)
	}

	blocks = append(blocks, section)

	var content string
	// get DN for the address
	addrs, err := net.LookupAddr(addr)
	if err == nil && len(addrs) > 0 {
		section = slack_webhook.SectionBlock()
		section.Text = slack_webhook.MrkdwnElement(fmt.Sprintf("addr: %s", strings.Join(addrs, " ")), true)

		blocks = append(blocks, section)

		content = fmt.Sprintf("*%s*\naddr: %s", LastLine, strings.Join(addrs, " "))
	} else {
		content = fmt.Sprintf("*%s*", LastLine)
	}

	var IP = net.ParseIP(addr)
	if IP != nil {
		core, err := s.ipinfoClient.GetIPInfo(IP)
		if err == nil && core != nil {
			section = slack_webhook.SectionBlock()
			section.Text = slack_webhook.MrkdwnElement(fmt.Sprintf("Country: %s\nCity: %s\nOrg: %s", core.Country, core.City, core.Org), true)

			blocks = append(blocks, section)
		}
	}

	message.Blocks = blocks
	message.Text = content

	return message
}

func (s SlackMessageHandler) readSettingsFile(File string) SlackUserSetting {
	var lines = strings.Split(File, "\n")
	var settings = SlackUserSetting{}

	for _, line := range lines {
		sep := strings.Split(line, "=")
		var value = strings.TrimSpace(strings.Join(sep[1:], "="))
		value = strings.TrimSuffix(value, "\"")
		value = strings.TrimSuffix(value, "'")

		value = strings.TrimPrefix(value, "\"")
		value = strings.TrimPrefix(value, "'")

		switch strings.TrimSpace(sep[0]) {
		case "CustomName":
			settings.CustomName = value
		case "SlackID":
			settings.SlackID = value
		default:
			continue
		}
	}

	return settings
}
