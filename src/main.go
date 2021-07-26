package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/xml"
	"io"
	"log"
	"os"
	"strings"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

type MessageBody struct {
	stanza.Message
	Body string `xml:"body"`
}

func main() {

	log.Println("Hello, please input your credentials")
	reader := bufio.NewReader(os.Stdin)
	log.Println("Enter your user: ")
	var username, _ = reader.ReadString('\n')
	log.Println("---------------------")
	log.Println("Enter your password: ")
	var password, _ = reader.ReadString('\n')
	log.Println("---------------------")
	log.Println("Enter the servers host: ")
	var serverDomain, _ = reader.ReadString('\n')
	log.Println("---------------------")
	log.Println("Attempting to contact host...")

	username = strings.TrimRight(username, "\r\n")
	password = strings.TrimRight(password, "\r\n")
	serverDomain = strings.TrimRight(serverDomain, "\r\n")

	xmppAddrFormat := jid.MustParse(username)
	session, err := xmpp.DialClientSession(
		context.TODO(), xmppAddrFormat,
		xmpp.BindResource(),
		xmpp.StartTLS(&tls.Config{
			ServerName: serverDomain,
		}),
		xmpp.SASL("", password, sasl.ScramSha1Plus, sasl.ScramSha1, sasl.Plain),
	)
	if err != nil {
		log.Printf("Error establishing a session: %q", err)
		return
	}

	// Exit and logout messages
	defer func() {
		log.Println("Closing session…")
		if err := session.Close(); err != nil {
			log.Printf("Error closing session: %q", err)
		}
		log.Println("Closing connection…")
		if err := session.Conn().Close(); err != nil {
			log.Printf("Error closing connection: %q", err)
		}
	}()

	// Send initial presence to let the server know we want to receive messages.
	err = session.Send(context.TODO(), stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		log.Printf("Error sending initial presence: %q", err)
		return
	}

	session.Serve(xmpp.HandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
		decoder := xml.NewTokenDecoder(t)

		// Ignore anything that's not a message
		if start.Name.Local != "message" {
			return nil
		}

		msg := MessageBody{}
		err = decoder.DecodeElement(&msg, start)
		if err != nil && err != io.EOF {
			log.Printf("Error decoding message: %q", err)
			return nil
		}

		// Don't reflect messages unless they are chat messages and actually have a body
		if msg.Body == "" || msg.Type != stanza.ChatMessage {
			return nil
		}

		log.Printf("%q: %q", msg.Body, msg.From.Bare())
		reply := MessageBody{
			Message: stanza.Message{
				To: msg.From.Bare(),
			},
			Body: msg.Body,
		}
		log.Printf("Replying to message %q from %s with body %q", msg.ID, reply.To, reply.Body)
		err = t.Encode(reply)
		if err != nil {
			log.Printf("Error responding to message %q: %q", msg.ID, err)
		}
		return nil
	}))
}
