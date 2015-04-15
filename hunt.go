package hunt

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"google.golang.org/appengine"
	//"google.golang.org/appengine/blobstore"
	"google.golang.org/appengine/datastore"

	"github.com/unrolled/render"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	"golang.org/x/net/context"
)

const formImageHTML = `
<html>
        <body>
                <form action="{{.}}" method="POST" enctype="multipart/form-data">
                        <input type="file" name="file"><br>
                        <input type="submit" name="submit" value="Submit">
                </form>
        </body>
</html>
`

var (
	R    = render.New()
	form = template.Must(template.New("root").Parse(formImageHTML))
)

func jsonBind(req *http.Request, data interface{}) error {
	defer req.Body.Close()

	err := json.NewDecoder(req.Body).Decode(&data)
	if err != nil {
		return err
	}

	return nil
}

func getAncestorKey(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, "GDGTH", "default", 0, nil)
}

func init() {
	goji.Put("/api/hunt", postHandler)
	goji.Get("/api/hunt", getAllHandler)
	goji.Get("/api/hunt/:id", getHandler)
	goji.Delete("/api/hunt", deleteAllHandler)
	goji.Delete("/api/hunt/:id", deleteHandler)

	goji.Put("/api/clue/:id", putClueHandler)

	// goji.Get("/api/image", getFormImgHandler)
	// goji.Post("/api/image", uploadImgHandler)

	goji.Serve()
}

func postHandler(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	var hunt Hunt

	err := jsonBind(req, &hunt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	key := datastore.NewKey(ctx, "Hunt", hunt.Id, 0, getAncestorKey(ctx))

	_, err = datastore.Put(ctx, key, &hunt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	R.JSON(w, http.StatusCreated, hunt)
}

func getAllHandler(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	var huntsList HuntsList
	var err error

	order := "id"
	page := 0
	per_page := 10

	opt := req.URL.Query().Get("order")
	if opt != "" {
		order = opt
	}

	opt = req.URL.Query().Get("page")
	if opt != "" {
		page, err = strconv.Atoi(opt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	opt = req.URL.Query().Get("per_page")
	if opt != "" {
		per_page, err = strconv.Atoi(opt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	offset := page * per_page

	query := datastore.NewQuery("Hunt").Ancestor(getAncestorKey(ctx)).Limit(per_page).Offset(offset).Order(order)

	_, err = query.GetAll(ctx, &(huntsList.Hunts))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	R.JSON(w, http.StatusOK, huntsList)
}

func getHandler(c web.C, w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	var hunt Hunt

	key := datastore.NewKey(ctx, "Hunt", c.URLParams["id"], 0, getAncestorKey(ctx))

	err := datastore.Get(ctx, key, &hunt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	R.JSON(w, http.StatusOK, hunt)
}

func deleteAllHandler(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	keys, err := datastore.NewQuery("Hunt").Ancestor(getAncestorKey(ctx)).KeysOnly().GetAll(ctx, nil)
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

	key := datastore.NewKey(ctx, "Hunt", c.URLParams["id"], 0, getAncestorKey(ctx))

	err := datastore.Delete(ctx, key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func putClueHandler(c web.C, w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	var hunt Hunt

	key := datastore.NewKey(ctx, "Hunt", c.URLParams["id"], 0, getAncestorKey(ctx))

	err := datastore.Get(ctx, key, &hunt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var clue Clue

	err = jsonBind(req, &clue)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hunt.Clues = append(hunt.Clues, &clue)

	_, err = datastore.Put(ctx, key, &hunt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	R.JSON(w, http.StatusCreated, hunt)
}

// func getFormImgHandler(c web.C, w http.ResponseWriter, req *http.Request) {
// 	ctx := appengine.NewContext(req)

// 	URL, err := blobstore.UploadURL(ctx, "/api/image", nil)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "text/html")

// 	err = form.Execute(w, URL)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// }

// func uploadImgHandler(c web.C, w http.ResponseWriter, req *http.Request) {
// 	_, _, err := blobstore.ParseUpload(req)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// }
