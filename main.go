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

	var accepted = regexp.MustCompile(`Accepted\s+(password|publickey)\s+for\s+(\S+)\s+from\s+(\S+)\s+port\s+(\S+)`)
	//Accepted publickey for tkmax777 from 192.168.100.64 port 52396 s
	var failed = regexp.MustCompile(`Failed\s+(password|publickey)\s+for\s+(\S+)\s+from\s(\S+)\s+port\s+(\S+)`)
	var failedInvalidUser = regexp.MustCompile(`Failed\s+(password|publickey)\s+for\s+invalid\s+user\s+(\S+)\s+from\s+(\S+)\s+port\s+(\S+)`)

	fmt.Println("Start Auth Notify")

	host, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	for {
		var loginMessage = <-messageChan

		var message = slack_webhook.Message{
			Username: fmt.Sprintf("[%s] SSH Auth Notifier", host),
		}

		var sendChannels []string

		switch {
		case accepted.MatchString(loginMessage.LastLine):
			message.Text = fmt.Sprintf("*%s*", loginMessage.LastLine)
			sendChannels = strings.Split(os.Getenv("SLACK_ACCEPTED_CHANNELS"), ",")
		case failed.MatchString(loginMessage.LastLine):
			submatch := failed.FindAllStringSubmatch(loginMessage.LastLine, 1)[0]
			if submatch[2] == "root" && os.Getenv("ROOT_NOTIFY") == "no" {
				message.Text = loginMessage.LastLine
				sendChannels = strings.Split(os.Getenv("SLACK_OTHER_CHANNELS"), ",")
				break
			}

			message.Text = fmt.Sprintf("*%s*", loginMessage.LastLine)

			var blocks = make([]slack_webhook.BlockBase, 0)

			var section = slack_webhook.SectionBlock()
			section.Text = slack_webhook.MrkdwnElement(fmt.Sprintf("*%s*", loginMessage.LastLine), false)

			blocks = append(blocks, slack_webhook.HeaderBlock("!Caution!", true))
			blocks = append(blocks, section)

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
