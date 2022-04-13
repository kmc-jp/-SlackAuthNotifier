package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/TKMAX777/AuthNotify/slack_webhook"
	"github.com/TKMAX777/AuthNotify/ssh_log"
)

func main() {
	var sshHandler = ssh_log.New()
	defer sshHandler.Close()

	messageChan, err := sshHandler.Start()
	if err != nil {
		panic(err)
	}

	var slackHook = slack_webhook.New(os.Getenv("SLACK_TOKEN"))

	var accepted = regexp.MustCompile(`Accepted\s(password|publickey)\sfor\s(\S+)from\s(\S+)port\s(\S+)`)
	var failed = regexp.MustCompile(`Failed\s(password|publickey)\sfor\s(\S+)from\s(\S+)port\s(\S+)`)
	var failedInvalidUser = regexp.MustCompile(`Failed\s(password|publickey)\sfor\sinvalid\suser\s(\S+)from\s(\S+)port\s(\S+)`)

	fmt.Println("Start Auth Notify")

	for {
		var loginMessage = <-messageChan

		var message = slack_webhook.Message{
			Username: "SSH Auth Notifier",
		}

		var sendChannels []string

		switch {
		case accepted.MatchString(loginMessage.LastLine):
			message.Text = fmt.Sprintf("*%s*", loginMessage.LastLine)
			sendChannels = strings.Split(os.Getenv("SLACK_ACCEPTED_CHANNELS"), ",")
		case failed.MatchString(loginMessage.LastLine):
			message.Text = fmt.Sprintf("*%s*", loginMessage.LastLine)

			var blocks = make([]slack_webhook.BlockBase, 0)

			var section = slack_webhook.SectionBlock()
			section.Text = slack_webhook.MrkdwnElement(fmt.Sprintf("*%s*", loginMessage.LastLine), false)

			blocks = append(blocks, section)
			blocks = append(blocks, slack_webhook.HeaderBlock("!Caution!", true))

			message.Blocks = blocks

			sendChannels = strings.Split(os.Getenv("SLACK_CAUTION_CHANNELS"), ",")
		case failedInvalidUser.MatchString(loginMessage.LastLine):
			message.Text = loginMessage.LastLine
			sendChannels = strings.Split(os.Getenv("SLACK_FAILED_CHANNELS"), ",")
		default:
			message.Text = loginMessage.LastLine
			sendChannels = strings.Split(os.Getenv("SLACK_OTHER_CHANNELS"), ",")
		}

		for _, channel := range sendChannels {
			message.Channel = strings.TrimSpace(channel)
			slackHook.Send(message)
		}
	}
}
