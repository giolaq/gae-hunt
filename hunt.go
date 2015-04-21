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

const (
	HUNT_ENTITY     = "HUNT"
	CLUE_ENTITY     = "CLUE"
	TAG_ENTITY      = "TAG"
	QUESTION_ENTITY = "QUESTION"
	ANSWER_ENTITY   = "ANSWER"
)

var (
	R    = render.New()
	form = template.Must(template.New("root").Parse(formImageHTML))
)

func getAncestorKey(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, "GDGTH", "default", 0, nil)
}

func init() {
	goji.Post("/api/hunt", newHuntHandler)
	goji.Put("/api/hunt", newHuntHandler)
	goji.Get("/api/hunt", getAllHuntsHandler)
	goji.Get("/api/hunt/:hid", getHuntHandler)
	goji.Delete("/api/hunt", delAllHuntsHandler)
	goji.Delete("/api/hunt/:hid", delHuntHandler)

	//goji.Post("/api/hunt/:"+HUNT_ID_PARAM+"/clue", newClueHandler)
	//goji.Delete("/api/hunt/:"+HUNT_ID_PARAM+"/clue", delAllCluesHandler)

	// goji.Get("/api/image", getFormImgHandler)
	// goji.Post("/api/image", uploadImgHandler)

	goji.Serve()
}

func newHuntHandler(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	var hunt Hunt

	err := json.NewDecoder(req.Body).Decode(&hunt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hunt_key, err := datastore.Put(ctx, datastore.NewKey(ctx, HUNT_ENTITY, hunt.Id, 0, getAncestorKey(ctx)), &hunt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, clue := range hunt.Clues {
		clue_key, err := datastore.Put(ctx, datastore.NewKey(ctx, CLUE_ENTITY, clue.Id, 0, hunt_key), &clue)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, tag := range clue.Tags {
			_, err := datastore.Put(ctx, datastore.NewKey(ctx, TAG_ENTITY, tag.Id, 0, clue_key), &tag)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		for _, question := range clue.Questions {
			question_key, err := datastore.Put(ctx, datastore.NewIncompleteKey(ctx, QUESTION_ENTITY, clue_key), &question)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			for _, answer := range question.Answers {
				_, err := datastore.Put(ctx, datastore.NewIncompleteKey(ctx, ANSWER_ENTITY, question_key), &answer)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

		}
	}

	R.JSON(w, http.StatusCreated, hunt)
}

func getAllHuntsHandler(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	var hunts []Hunt
	var err error

	order := "Id"
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

	hunts_query := datastore.NewQuery(HUNT_ENTITY).Ancestor(getAncestorKey(ctx)).Limit(per_page).Offset(offset).Order(order)

	hunts_key, err := hunts_query.GetAll(ctx, &hunts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for hunt_id, hunt_key := range hunts_key {
		clues_query := datastore.NewQuery(CLUE_ENTITY).Ancestor(hunt_key)

		clues_key, err := clues_query.GetAll(ctx, &(hunts[hunt_id].Clues))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for clue_id, clue_key := range clues_key {
			tags_query := datastore.NewQuery(TAG_ENTITY).Ancestor(clue_key)
			_, err := tags_query.GetAll(ctx, &(hunts[hunt_id].Clues[clue_id].Tags))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			questions_query := datastore.NewQuery(QUESTION_ENTITY).Ancestor(clue_key)
			questions_key, err := questions_query.GetAll(ctx, &(hunts[hunt_id].Clues[clue_id].Questions))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			for question_id, question_key := range questions_key {
				answers_query := datastore.NewQuery(ANSWER_ENTITY).Ancestor(question_key)
				_, err := answers_query.GetAll(ctx, &(hunts[hunt_id].Clues[clue_id].Questions[question_id].Answers))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			}
		}
	}

	R.JSON(w, http.StatusOK, hunts)
}

func getHuntHandler(c web.C, w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	var hunt Hunt

	hunt_id := c.URLParams["hid"]
	hunt_key := datastore.NewKey(ctx, HUNT_ENTITY, hunt_id, 0, getAncestorKey(ctx))

	err := datastore.Get(ctx, hunt_key, &hunt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	clues_query := datastore.NewQuery(CLUE_ENTITY).Ancestor(hunt_key)

	clues_key, err := clues_query.GetAll(ctx, &(hunt.Clues))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for clue_id, clue_key := range clues_key {
		tags_query := datastore.NewQuery(TAG_ENTITY).Ancestor(clue_key)
		_, err := tags_query.GetAll(ctx, &(hunt.Clues[clue_id].Tags))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		questions_query := datastore.NewQuery(QUESTION_ENTITY).Ancestor(clue_key)
		questions_key, err := questions_query.GetAll(ctx, &(hunt.Clues[clue_id].Questions))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for question_id, question_key := range questions_key {
			answers_query := datastore.NewQuery(ANSWER_ENTITY).Ancestor(question_key)
			_, err := answers_query.GetAll(ctx, &(hunt.Clues[clue_id].Questions[question_id].Answers))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		}
	}

	R.JSON(w, http.StatusOK, hunt)
}

func delAllHuntsHandler(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	hunts_key, err := datastore.NewQuery(HUNT_ENTITY).Ancestor(getAncestorKey(ctx)).KeysOnly().GetAll(ctx, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = datastore.DeleteMulti(ctx, hunts_key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//TODO cancellare campi relativi
}

func delHuntHandler(c web.C, w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	hunt_id := c.URLParams["hid"]
	hunt_key := datastore.NewKey(ctx, HUNT_ENTITY, hunt_id, 0, getAncestorKey(ctx))

	err := datastore.Delete(ctx, hunt_key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//TODO cancellare campi relativi
}

// func newClueHandler(c web.C, w http.ResponseWriter, req *http.Request) {
// 	ctx := appengine.NewContext(req)

// 	var clue Clue

// 	err := json.NewDecoder(req.Body).Decode(&clue)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	hkey := datastore.NewKey(ctx, HUNT_ENTITY, c.URLParams[HUNT_ID_PARAM], 0, getAncestorKey(ctx))
// 	ckey := datastore.NewKey(ctx, CLUE_ENTITY, clue.Id, 0, hkey)

// 	_, err = datastore.Put(ctx, ckey, &clue)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	R.JSON(w, http.StatusCreated, clue)
// }

// func delAllCluesHandler(c web.C, w http.ResponseWriter, req *http.Request) {
// 	ctx := appengine.NewContext(req)

// 	var hunt Hunt

// 	key := datastore.NewKey(ctx, HUNT_ENTITY, c.URLParams[HUNT_ID_PARAM], 0, getAncestorKey(ctx))

// 	err := datastore.Get(ctx, key, &hunt)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	hunt.Clues = nil

// 	_, err = datastore.Put(ctx, key, &hunt)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	R.JSON(w, http.StatusCreated, hunt)
// }

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
