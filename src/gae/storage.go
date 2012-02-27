package gae

import (
	"appengine"
	"appengine/datastore"
	"github.com/ukai/blogplus"
	"net/http"
	"sort"
)

const (
	activityKind = "Activity"
	datespecKind = "Datespec"
	activityRef  = "ActivityRef"
)

// string key is activityId (post.Id)
type ActivityEntity struct {
	Id        string
	Published string
	Post      []byte
}

// key is datespec / activityId
type DatespecEntity struct {
	Id *datastore.Key
}

type DatastoreStorage struct {
	filter func(blogplus.Activity) bool
}

func NewDatastoreStorage() *DatastoreStorage {
	return &DatastoreStorage{}
}

func (d *DatastoreStorage) SetFilter(filter func(blogplus.Activity) bool) {
	d.filter = filter
}

func (d *DatastoreStorage) StorePosts(req *http.Request, posts []blogplus.Activity) {
	c := appengine.NewContext(req)
	var keys []*datastore.Key
	var src []interface{}
	for _, post := range posts {
		if d.filter != nil && !d.filter(post) {
			c.Debugf("ignore post:%s", post.Id)
			continue
		}
		datespec := blogplus.GetDatespec(post.Published)
		data, err := blogplus.EncodeActivity(post)
		if err != nil {
			c.Errorf("encode error:%#v", err)
			continue
		}
		c.Infof("store %s datespec %s", post.Id, datespec)
		datekey := datastore.NewKey(c, activityRef, post.Id, 0, datastore.NewKey(c, datespecKind, datespec, 0, nil))
		key := datastore.NewKey(c, activityKind, post.Id, 0, nil)
		keys = append(keys, datekey)
		src = append(src, &DatespecEntity{Id: key})
		keys = append(keys, key)
		src = append(src, &ActivityEntity{
			Id:        post.Id,
			Published: post.Published,
			Post:      data})
	}
	_, err := datastore.PutMulti(c, keys, src)
	if err != nil {
		c.Errorf("put error:%#v", err)
	}
}

func (d *DatastoreStorage) GetLatestPosts(req *http.Request) []blogplus.Activity {
	c := appengine.NewContext(req)
	q := datastore.NewQuery(activityKind).Order("-Published").Limit(10)
	t := q.Run(c)
	var posts []blogplus.Activity
	for {
		var ae ActivityEntity
		_, err := t.Next(&ae)
		if err == datastore.Done {
			break
		}
		if err != nil {
			c.Errorf("query error:%#v", err)
			break
		}
		post, err := blogplus.DecodeActivity(ae.Post)
		if err != nil {
			c.Errorf("decode error:%#v", err)
			continue
		}
		posts = append(posts, post)
	}
	return posts
}

func (d *DatastoreStorage) GetPost(req *http.Request, activityId string) (post blogplus.Activity, ok bool) {
	c := appengine.NewContext(req)
	k := datastore.NewKey(c, activityKind, activityId, 0, nil)
	var ae ActivityEntity
	if err := datastore.Get(c, k, &ae); err != nil {
		c.Errorf("get error:%#v", err)
		return post, false
	}
	post, err := blogplus.DecodeActivity(ae.Post)
	if err != nil {
		c.Errorf("decode error:%#v", err)
		return post, false
	}
	return post, true
}

func (d *DatastoreStorage) GetDates(req *http.Request) []blogplus.ArchiveItem {
	c := appengine.NewContext(req)
	q := datastore.NewQuery(activityRef)
	t := q.Run(c)
	m := make(map[string]int)
	for {
		var de DatespecEntity
		key, err := t.Next(&de)
		if err == datastore.Done {
			break
		}
		if err != nil {
			c.Errorf("query error:%#v", err)
			break
		}
		datespec := key.Parent().StringID()
		m[datespec] += 1
	}
	var items blogplus.ArchiveItemList
	for datespec, count := range m {
		items = append(items, blogplus.ArchiveItem{
			Datespec: datespec,
			Count:    count})
	}
	sort.Sort(items)
	return items
}

func (d *DatastoreStorage) GetArchivedPosts(req *http.Request, datespec string) []blogplus.Activity {
	c := appengine.NewContext(req)
	datespecKey := datastore.NewKey(c, datespecKind, datespec, 0, nil)
	q := datastore.NewQuery(activityRef).Ancestor(datespecKey)
	t := q.Run(c)
	var keys []*datastore.Key
	for {
		var de DatespecEntity
		_, err := t.Next(&de)
		if err == datastore.Done {
			break
		}
		if err != nil {
			c.Errorf("query error:%#v", err)
			break
		}
		keys = append(keys, de.Id)
	}
	aelist := make([]ActivityEntity, len(keys))
	err := datastore.GetMulti(c, keys, aelist)
	if err != nil {
		c.Errorf("get multi error:%#v", err)
		return nil
	}
	var posts []blogplus.Activity
	for _, ae := range aelist {
		post, err := blogplus.DecodeActivity(ae.Post)
		if err != nil {
			c.Errorf("decode error:%#v", err)
			continue
		}
		posts = append(posts, post)
	}
	return posts
}
