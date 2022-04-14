package main

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/TKMAX777/SlackAuthNotifier/slack_webhook"
	"github.com/TKMAX777/SlackAuthNotifier/ssh_log"
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
			Username:  fmt.Sprintf("[%s] SSH Auth Notifier", strings.ToUpper(host)),
			IconEmoji: os.Getenv("SLACK_ICON_EMOJI"),
			IconURL:   os.Getenv("SLACK_ICON_URI"),
		}

		var sendChannels []string

		switch {
		case accepted.MatchString(loginMessage.LastLine):
			submatch := accepted.FindAllStringSubmatch(loginMessage.LastLine, 1)[0]

			var content string

			// get DN for the address
			addrs, err := net.LookupAddr(submatch[3])
			if err == nil && len(addrs) > 0 {
				content = fmt.Sprintf("*%s*\naddr: %s", loginMessage.LastLine, strings.Join(addrs, " "))
			} else {
				content = fmt.Sprintf("*%s*", loginMessage.LastLine)
			}

			message.Text = content
			sendChannels = strings.Split(os.Getenv("SLACK_ACCEPTED_CHANNELS"), ",")
		case failed.MatchString(loginMessage.LastLine):
			submatch := failed.FindAllStringSubmatch(loginMessage.LastLine, 1)[0]
			if submatch[2] == "root" && os.Getenv("ROOT_NOTIFY") == "no" {
				message.Text = loginMessage.LastLine
				sendChannels = strings.Split(os.Getenv("SLACK_OTHER_CHANNELS"), ",")
				break
			}

			var content string

			// get DN for the address
			addrs, err := net.LookupAddr(submatch[3])
			if err == nil && len(addrs) > 0 {
				content = fmt.Sprintf("*%s*\naddr: %s", loginMessage.LastLine, strings.Join(addrs, " "))
			} else {
				content = fmt.Sprintf("*%s*", loginMessage.LastLine)
			}

			message.Text = content

			var blocks = make([]slack_webhook.BlockBase, 0)

			var section = slack_webhook.SectionBlock()
			section.Text = slack_webhook.MrkdwnElement(content, false)

			blocks = append(blocks, slack_webhook.HeaderBlock("!Caution!", true))
			blocks = append(blocks, section)

			message.Blocks = blocks

			sendChannels = strings.Split(os.Getenv("SLACK_CAUTION_CHANNELS"), ",")
		case failedInvalidUser.MatchString(loginMessage.LastLine):

			submatch := failedInvalidUser.FindAllStringSubmatch(loginMessage.LastLine, 1)[0]

			var content string

			// get DN for the address
			addrs, err := net.LookupAddr(submatch[3])
			if err == nil && len(addrs) > 0 {
				content = fmt.Sprintf("*%s*\naddr: %s", loginMessage.LastLine, strings.Join(addrs, " "))
			} else {
				content = fmt.Sprintf("*%s*", loginMessage.LastLine)
			}

			message.Text = content

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
