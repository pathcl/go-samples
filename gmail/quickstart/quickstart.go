/**
 * @license
 * Copyright Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
// [START gmail_quickstart]
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

type Message struct {
	From      string
	To        string
	Subject   string
	BodyPlain string
	BodyHtml  string
}

func findHeader(messagePart *gmail.MessagePart, name string) string {
	for _, header := range messagePart.Headers {
		if header.Name == name {
			return header.Value
		}
	}
	return ""
}

func findMessagePartByMimeType(messagePart *gmail.MessagePart, mimeType string) *gmail.MessagePart {
	if messagePart.MimeType == mimeType {
		return messagePart
	}
	if strings.HasPrefix(messagePart.MimeType, "multipart") {
		for _, part := range messagePart.Parts {
			if mp := findMessagePartByMimeType(part, mimeType); mp != nil {
				return mp
			}
		}
	}
	return nil
}

func getMessagePartData(srv *gmail.Service, user, messageId string, messagePart *gmail.MessagePart) (string, error) {
	var dataBase64 string

	if messagePart.Body.AttachmentId != "" {
		body, err := srv.Users.Messages.Attachments.Get(user, messageId, messagePart.Body.AttachmentId).Do()
		if err != nil {
			return "", errors.Wrap(err, "getMessagePartData get attachment")
		}

		dataBase64 = body.Data
	} else {
		dataBase64 = messagePart.Body.Data
	}

	data, err := base64.URLEncoding.DecodeString(dataBase64)
	if err != nil {
		return "", errors.Wrap(err, "getMessagePartData base64 decode")
	}

	return string(data), nil
}

func parseMessage(srv *gmail.Service, gmailMessage *gmail.Message, user string) (*Message, error) {
	if gmailMessage.Payload == nil {
		return nil, fmt.Errorf("No payload in gmail message.")
	}

	message := &Message{
		From:    findHeader(gmailMessage.Payload, "From"),
		To:      findHeader(gmailMessage.Payload, "To"),
		Subject: findHeader(gmailMessage.Payload, "Subject"),
	}

	//	plainMessagePart := findMessagePartByMimeType(gmailMessage.Payload, "text/plain")
	//	if plainMessagePart != nil {
	//		plainMessage, err := getMessagePartData(srv, user, gmailMessage.Id, plainMessagePart)
	//		if err != nil {
	//			return nil, errors.Wrap(err, "parseMessage plain")
	//		}
	//		message.BodyPlain = plainMessage
	//	}

	htmlMessagePart := findMessagePartByMimeType(gmailMessage.Payload, "text/html")
	if htmlMessagePart != nil {
		htmlMessage, err := getMessagePartData(srv, user, gmailMessage.Id, htmlMessagePart)
		if err != nil {
			return nil, errors.Wrap(err, "parseMessage html")
		}
		message.BodyHtml = htmlMessage
	}

	return message, nil
}

func main() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	m, _ := srv.Users.Messages.List("me").Q("label:newsletter after:2021/05/01 from: hi@vimtricks.com").Do()

	for _, email := range m.Messages {

		msg, err := srv.Users.Messages.Get("me", email.Id).Format("full").Do()
		if err != nil {
			log.Fatalf("Unable to retrieve message %v: %v", email.Id, err)
		}

		body, _ := parseMessage(srv, msg, "me")
		fmt.Println(body)
	}

}

// [END gmail_quickstart]
