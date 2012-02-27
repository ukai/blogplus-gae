package gae

import (
	"appengine"
	"appengine/memcache"
	"appengine/taskqueue"
	"appengine/urlfetch"
	"github.com/ukai/blogplus"
	"net/http"
	"net/url"
)

const (
	FetcherCount = 1000
)

type Controller struct {
	path    string
	fetcher *blogplus.Fetcher
	s       blogplus.Storage
}

func NewController(path string, fetcher *blogplus.Fetcher, s blogplus.Storage) *Controller {
	return &Controller{path: path, fetcher: fetcher, s: s}
}

// handles task request.
func (c *Controller) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)
	client := urlfetch.Client(ctx)
	activityId := req.FormValue("activityId")
	start := req.FormValue("start")
	pageToken := req.FormValue("pageToken")

	if activityId != "" {
		ctx.Infof("fetch post:%s", activityId)
		posts := c.fetcher.FetchPost(client, activityId)
		c.s.StorePosts(req, posts)
		return
	} else if start == "" && pageToken == "" {
		ctx.Infof("fetch")
		posts := c.fetcher.Fetch(client)
		c.s.StorePosts(req, posts)
		return
	}
	ctx.Infof("fetch start pageToken:%s", pageToken)
	activityFeed, err := c.fetcher.GetActivities(client, pageToken)
	if err != nil {
		ctx.Errorf("fetcher error:%#v", err)
		return
	}
	c.s.StorePosts(req, activityFeed.Items)
	if activityFeed.NextPageToken != "" {
		ctx.Infof("nextPageToken:%s", activityFeed.NextPageToken)
		t := taskqueue.NewPOSTTask(c.path, url.Values{
			"pageToken": {activityFeed.NextPageToken}})
		taskqueue.Add(ctx, t, "")
	}
}

func (c *Controller) ForceFetch(req *http.Request) {
	ctx := appengine.NewContext(req)
	ctx.Infof("request force fetch")
	t := taskqueue.NewPOSTTask(c.path, url.Values{})
	taskqueue.Add(ctx, t, "force")
}

func (c *Controller) MaybeFetch(req *http.Request) {
	ctx := appengine.NewContext(req)
	if count, err := memcache.Increment(ctx, "fetchCounter", 1, 0); err != nil {
		ctx.Errorf("memcache increment fetchCounter:%#v", err)
	} else {
		if (count % FetcherCount) != 0 {
			ctx.Debugf("ignore fetch count:%d", count)
			return
		}
	}
	ctx.Infof("request fetch task")
	t := taskqueue.NewPOSTTask(c.path, url.Values{})
	taskqueue.Add(ctx, t, "")
}

func (c *Controller) MaybeFetchPost(req *http.Request, activityId string) {
	ctx := appengine.NewContext(req)
	if count, err := memcache.Increment(ctx, "fetchCounter."+activityId, 1, 0); err != nil {
		ctx.Errorf("memcache increment fetchCounter.%s: %#v", activityId, err)
	} else {
		if (count % FetcherCount) != 0 {
			ctx.Debugf("ignore fetch %s count:%d", activityId, count)
			return
		}
	}
	ctx.Infof("request fetch task:%s", activityId)
	t := taskqueue.NewPOSTTask(c.path, url.Values{
		"activityId": {activityId}})
	taskqueue.Add(ctx, t, "")
}
