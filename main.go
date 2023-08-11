package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/ipinfo/go/v2/ipinfo"
	"github.com/kmc-jp/SlackAuthNotifier/slack_webhook"
	"github.com/kmc-jp/SlackAuthNotifier/ssh_log"
)

func main() {
	var sshHandler = ssh_log.New()
	defer sshHandler.Close()

	if os.Getenv("TIME_FORMAT") != "" {
		sshHandler.TimeFormat = os.Getenv("TIME_FORMAT")
	}

	messageChan, err := sshHandler.Start()
	if err != nil {
		panic(err)
	}

	var slackHook = slack_webhook.New(os.Getenv("SLACK_TOKEN"))

	var accepted = regexp.MustCompile(`Accepted\s+(password|publickey)\s+for\s+(\S+)\s+from\s+(\S+)\s+port\s+(\S+)`)
	var failed = regexp.MustCompile(`Failed\s+(password|publickey)\s+for\s+(\S+)\s+from\s(\S+)\s+port\s+(\S+)`)
	var failedInvalidUser = regexp.MustCompile(`Failed\s+(password|publickey)\s+for\s+invalid\s+user\s+(\S+)\s+from\s+(\S+)\s+port\s+(\S+)`)

	var ipinfoClient = ipinfo.NewClient(nil, nil, os.Getenv("IPINFO_TOKEN"))
	var slackMessage = NewSlackMessageHandler(ipinfoClient)

	fmt.Println("Start Auth Notify")

	host, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	for {
		var loginMessage, ok = <-messageChan
		if !ok {
			log.Println("Message Channel Closed")
			return
		}

		var message slack_webhook.Message
		var sendChannels []string

		switch {
		case accepted.MatchString(loginMessage.LastLine):
			submatch := accepted.FindAllStringSubmatch(loginMessage.LastLine, 1)[0]

			message = slackMessage.NewMessage(submatch[3], submatch[2], loginMessage.LastLine, MessageTypeAccept)
			sendChannels = strings.Split(os.Getenv("SLACK_ACCEPTED_CHANNELS"), ",")
		case failed.MatchString(loginMessage.LastLine):
			submatch := failed.FindAllStringSubmatch(loginMessage.LastLine, 1)[0]
			if submatch[2] == "root" && os.Getenv("ROOT_NOTIFY") == "no" {
				message.Text = loginMessage.LastLine
				sendChannels = strings.Split(os.Getenv("SLACK_OTHER_CHANNELS"), ",")
				break
			}

			message = slackMessage.NewMessage(submatch[3], submatch[2], loginMessage.LastLine, MessageTypeCaution)

			sendChannels = strings.Split(os.Getenv("SLACK_CAUTION_CHANNELS"), ",")
		case failedInvalidUser.MatchString(loginMessage.LastLine):
			submatch := failedInvalidUser.FindAllStringSubmatch(loginMessage.LastLine, 1)[0]

			message = slackMessage.NewMessage(submatch[3], submatch[2], loginMessage.LastLine, MessageTypeFailed)

			sendChannels = strings.Split(os.Getenv("SLACK_FAILED_CHANNELS"), ",")
		default:
			message.Text = loginMessage.LastLine
			sendChannels = strings.Split(os.Getenv("SLACK_OTHER_CHANNELS"), ",")
		}

		message.Username = fmt.Sprintf("[%s] SSH Auth Notifier", strings.ToUpper(host))
		message.IconEmoji = os.Getenv("SLACK_ICON_EMOJI")
		message.IconURL = os.Getenv("SLACK_ICON_URI")

		for _, channel := range sendChannels {
			message.Channel = strings.TrimSpace(channel)
			slackHook.Send(message)
		}
	}
}
