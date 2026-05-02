package main

import (
	"net/http"
	"io"
	"encoding/xml"
	"html"
	"context"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}


func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gator")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	cookie := &RSSFeed{}
	err = xml.Unmarshal(data, cookie)
	if err != nil {
	return nil, err
	}
	
	cookie.Channel.Title = html.UnescapeString(cookie.Channel.Title)
	cookie.Channel.Description = html.UnescapeString(cookie.Channel.Description)
	for i := range cookie.Channel.Item {
		cookie.Channel.Item[i].Title = html.UnescapeString(cookie.Channel.Item[i].Title)
		cookie.Channel.Item[i].Description = html.UnescapeString(cookie.Channel.Item[i].Description)
	}


	return cookie, nil
}