# Slack Auth Notifier

## About

This program monitors /var/log/auth.log to post a history of login attempts to Slack.

## Install

```sh
go install github.com/TKMAX777/SlackAuthNotifier@latest
```

## Usage

Specify the following environment variables

```
# Notification channel for successful login
SLACK_ACCEPTED_CHANNELS=C*******, C*******

# Notification channel for failed login attempts
SLACK_CAUTION_CHANNELS=C*******, C*******

# Other information channels
SLACK_OTHER_CHANNELS=C*******

# Specify whether to post root login attempt history to caution channels
ROOT_NOTIFY=no

SLACK_TOKEN=xoxb-*****
SLACK_ICON_EMOJI=:key:
```
