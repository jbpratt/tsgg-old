package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type command struct {
	c     func(*chat, []string) error
	usage string
}

// TODO need to refactor this... usage strings incomplete/double
var commands = map[string]command{
	"/w":           {sendWhisper, "user message"},
	"/whisper":     {sendWhisper, "user message"},
	"/me":          {sendAction, "message"},
	"/tag":         {addTag, "user color"},
	"/untag":       {removeTag, "user"},
	"/highlight":   {addHighlight, "user"},
	"/unhighlight": {removeHighlight, "user"},
	"/mute":        {sendMute, "user [time in seconds]"},
	"/unmute":      {sendUnmute, "user"},
	"/ban":         {sendBan, "user reason [time (in seconds)]"},
	"/ipban":       {sendBan, "user reason [time (in seconds)]"},
	"/unban":       {sendUnban, "user"},
	"/subonly":     {sendSubOnly, "{on,off}"},
	"/broadcast":   {sendBroadcast, "message"},
}

func (c *chat) handleCommand(message string) error {
	s := strings.Split(message, " ")

	f, ok := commands[s[0]]
	if ok {
		return f.c(c, s)
	}

	return fmt.Errorf("unknown command: %s", s[0])
}

func addHighlight(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("Usage: /highlight user")
	}

	user := strings.ToLower(tokens[1])
	if contains(c.config.Highlighted, user) {
		return fmt.Errorf("%s is already highlighted", user)
	}

	c.config.Lock()
	c.config.Highlighted = append(c.config.Highlighted, user)
	c.config.Unlock()

	return c.config.save()
}

func removeHighlight(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("Usage: /unhighlight user")
	}

	user := strings.ToLower(tokens[1])
	c.config.Lock()
	defer c.config.Unlock()

	for i := 0; i < len(c.config.Highlighted); i++ {
		if strings.ToLower(c.config.Highlighted[i]) == user {
			c.config.Highlighted = append(c.config.Highlighted[:i], c.config.Highlighted[i+1:]...)
			return c.config.save()
		}
	}

	return fmt.Errorf("User: %s is not in highlight list", user)
}

func addTag(c *chat, tokens []string) error {
	if len(tokens) < 3 {
		return errors.New("Usage: /tag user [Black, Red, Green, Yellow, Blue, Magenta, Cyan, White]")
	}

	color := strings.ToLower(tokens[2])
	user := strings.ToLower(tokens[1])

	_, ok := tagMap[color]
	if !ok {
		return fmt.Errorf("invalid color: %s", color)
	}

	c.config.Lock()
	if c.config.Tags == nil {
		c.config.Tags = make(map[string]string)
	}
	c.config.Tags[user] = color
	c.config.Unlock()

	return c.config.save()
}

func removeTag(c *chat, tokens []string) error {
	if len(tokens) != 2 {
		return errors.New("Usage: /untag user")
	}

	user := strings.ToLower(tokens[1])

	c.config.Lock()
	defer c.config.Unlock()

	if _, ok := c.config.Tags[user]; ok {
		delete(c.config.Tags, user)
		return c.config.save()
	}

	return fmt.Errorf("%s is not tagged", user)
}

func sendMute(c *chat, tokens []string) error {
	if len(tokens) < 2 || len(tokens) > 3 {
		return errors.New("Usage: /mute user [time in seconds]")
	}

	var err error
	var duration int64 //server chooses default duration

	if len(tokens) >= 3 {
		duration, err = strconv.ParseInt(strings.TrimSpace(tokens[2]), 10, 64)
		if err != nil {
			return err
		}
	}

	return c.Session.SendMute(tokens[1], time.Duration(duration)*time.Second)
}

func sendUnmute(c *chat, tokens []string) error {
	if len(tokens) != 2 {
		return errors.New("Usage: /unmute user")
	}

	return c.Session.SendUnmute(tokens[1])
}

func sendBan(c *chat, tokens []string) error {
	if len(tokens) < 3 || len(tokens) > 4 {
		return errors.New("Usage: /[ip]ban user reason [time (in seconds)]")
	}

	var err error
	var duration int64 //server chooses default duration
	banip := tokens[0] == "/ipban"

	if len(tokens) == 4 {
		duration, err = strconv.ParseInt(strings.TrimSpace(tokens[3]), 10, 64)
		if err != nil {
			return err
		}
	}

	return c.Session.SendBan(tokens[1], tokens[2], time.Duration(duration)*time.Second, banip)
}

func sendUnban(c *chat, tokens []string) error {
	if len(tokens) != 2 {
		return errors.New("Usage: /unban user")
	}

	return c.Session.SendUnban(tokens[1])
}

func sendSubOnly(c *chat, tokens []string) error {
	so := tokens[1]
	if len(tokens) != 2 || (so != "on" && so != "off") {
		return errors.New("Usage: /subonly {on,off}")
	}

	subonly := so == "on"
	return c.Session.SendSubOnly(subonly)
}

func sendAction(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("Usage: /me message")
	}
	return c.Session.SendAction(tokens[1])
}

func sendBroadcast(c *chat, tokens []string) error {
	if len(tokens) < 2 {
		return errors.New("Usage: /broadcast message")
	}
	//TODO dggchat
	return errors.New("not implemented")
}

func sendWhisper(c *chat, tokens []string) error {
	if len(tokens) < 3 {
		return errors.New("Usage: /w user message")
	}

	nick := tokens[1]
	message := strings.Join(tokens[2:], " ")

	tm := time.Unix(time.Now().Unix()/1000, 0)
	tag := fmt.Sprintf(" %s*%s ", bgWhite, reset)
	msg := fmt.Sprintf("%s%s[PM -> %s] %s %s", tag, fgBrightWhite, nick, message, reset)
	c.Session.SendPrivateMessage(nick, message)
	c.renderFormattedMessage(msg, tm)
	return nil
}
