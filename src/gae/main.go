package gae

import (
	"github.com/ukai/blogplus"
	"net/http"
)

func init() {
	s := NewDatastoreStorage()
	s.SetFilter(blogplus.IsMeaningfulPost)
	fetcher := blogplus.NewFetcher(userId, key)
	c := NewController("/fetcher", fetcher, s)
	http.Handle("/fetcher", c)

	b := blogplus.NewBlogplus(s, c)
	b.Title = title
	b.AuthorName = authorName
	b.AuthorUri = authorUri
	b.Scheme = scheme
	b.Host = host
	b.SetStaticDir(staticDir)
	if templateDir != "" {
		b.LoadTemplates(templateDir)
	}
	http.Handle("/", b)
}
