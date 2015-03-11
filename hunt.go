package hunt

import (
	"appengine"
	"appengine/datastore"
	"encoding/json"
	//"github.com/goji/param"
	"github.com/unrolled/render"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"net/http"
	"strconv"
	"time"
)

type Greeting struct {
	Id int64

	Author  string
	Content string
	Date    time.Time
}

var (
	R = render.New()
)

func jsonBind(req *http.Request, data interface{}) error {
	defer req.Body.Close()

	err := json.NewDecoder(req.Body).Decode(&data)
	if err != nil {
		return err
	}

	return nil
}

func guestbookKey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "Guestbook", "default_guestbook", 0, nil)
}

func init() {
	goji.Post("/", postHandler)
	goji.Get("/", getAllHandler)
	goji.Get("/:id", getHandler)
	goji.Delete("/", deleteAllHandler)
	goji.Delete("/:id", deleteHandler)

	goji.Serve()
}

func postHandler(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	var greet Greeting
	err := jsonBind(req, &greet)

	greet.Date = time.Now()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	key := datastore.NewIncompleteKey(ctx, "Greeting", guestbookKey(ctx))

	_, err = datastore.Put(ctx, key, &greet)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getAllHandler(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	query := datastore.NewQuery("Greeting").Ancestor(guestbookKey(ctx))

	var greets []Greeting

	keys, err := query.GetAll(ctx, &greets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for i, _ := range greets {
		greets[i].Id = keys[i].IntID()
	}

	R.JSON(w, http.StatusOK, greets)
}

func getHandler(c web.C, w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	id, err := strconv.ParseInt(c.URLParams["id"], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var greet Greeting

	key := datastore.NewKey(ctx, "Greeting", "", id, guestbookKey(ctx))
	err = datastore.Get(ctx, key, &greet)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	greet.Id = key.IntID()

	R.JSON(w, http.StatusOK, greet)
}

func deleteAllHandler(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	query := datastore.NewQuery("Greeting").Ancestor(guestbookKey(ctx))

	var greets []Greeting

	keys, err := query.GetAll(ctx, &greets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = datastore.DeleteMulti(ctx, keys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deleteHandler(c web.C, w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	id, err := strconv.ParseInt(c.URLParams["id"], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	key := datastore.NewKey(ctx, "Greeting", "", id, guestbookKey(ctx))
	err = datastore.Delete(ctx, key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
