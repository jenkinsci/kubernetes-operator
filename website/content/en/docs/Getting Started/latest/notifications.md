---
title: "Notifications"
linkTitle: "Notifications"
weight: 7
date: 2021-08-19
description: >
    How to setup operator notifications.
---

## Slack

Please follow [this](https://api.slack.com/incoming-webhooks) instructions to get web hook URL.

Create web hook secret with name `jenkins-operator-notification-data`. Contains key `url` with provided web hook URL.

```bash
$ kubectl create secret generic jenkins-operator-notification-data --from-literal=url=<webhook_url>
```

Example configuration for Slack:

```yaml
kind: Jenkins
spec:
  master:
    notifications:
    - level: info
      verbose: true
      name: <name>
      slack:
        webHookURLSecretKeySelector:
          secret:
            name: <secret_name>
          key: <key>
```

## Microsoft Teams

Please follow [this](https://docs.microsoft.com/en-gb/outlook/actionable-messages/send-via-connectors) instructions to get web hook URL.

Example configuration for Microsoft Teams:

```yaml
kind: Jenkins
spec:
  master:
    notifications:
    - level: info
      verbose: true
      name: <name>
      teams:
        webHookURLSecretKeySelector:
          secret:
            name: <secret_name>
          key: <key>
```

## Mailgun

Example configuration for Mailgun:

```yaml
kind: Jenkins
spec:
  master:
    notifications:
    - level: info
      verbose: true
      name: <name>
      mailgun:
        domain: <domain>
        apiKeySecretKeySelector:
          secret:
            name: <secret_name>
          key: <key>
        recipient: <your_email>
        from: <mailgun_email>
```

## Debug options

As you see there is two debugging options: 

* `level` (warning/info) - Set level of messages to send.

* `verbose` - Print stacktrace and additional error messages

## Multiple providers

You can use multiple providers to send notification to another communication channels at the same time.
For example you will send notifications to Slack and Teams.

```yaml
kind: Jenkins
spec:
  master:
    notifications:
    - level: info
      verbose: true
      name: nslack
      slack:
        webHookURLSecretKeySelector:
          secret:
            name: <secret_name>
          key: <key>
    - level: info
      verbose: true
      name: nteams
      teams:
        webHookURLSecretKeySelector:
          secret:
            name: <secret_name>
          key: <key>
```
