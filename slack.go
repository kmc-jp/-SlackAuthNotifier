package main

import (
	"fmt"
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

func NewMessage(ipinfoClient *ipinfo.Client, addr, LastLine string, messageType MessageType) slack_webhook.Message {
	var message slack_webhook.Message

	var section = slack_webhook.SectionBlock()
	if messageType == MessageTypeAccept {
		section.Text = slack_webhook.MrkdwnElement(fmt.Sprintf("*%s*", LastLine), false)
	} else {
		section.Text = slack_webhook.MrkdwnElement(LastLine, false)
	}

	var blocks = make([]slack_webhook.BlockBase, 0)
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
		core, err := ipinfoClient.GetIPInfo(IP)
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
