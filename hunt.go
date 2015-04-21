package hunt

import (
	"encoding/json"
	//"html/template"
	"net/http"
	"strconv"

	"google.golang.org/appengine"
	//"google.golang.org/appengine/blobstore"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"

	"github.com/unrolled/render"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	"golang.org/x/net/context"
)

// const formImageHTML = `
// <html>
//         <body>
//                 <form action="{{.}}" method="POST" enctype="multipart/form-data">
//                         <input type="file" name="file"><br>
//                         <input type="submit" name="submit" value="Submit">
//                 </form>
//         </body>
// </html>
// `

//var (
//	form = template.Must(template.New("root").Parse(formImageHTML))
//)

const (
	HUNT_ENTITY     = "HUNT"
	CLUE_ENTITY     = "CLUE"
	TAG_ENTITY      = "TAG"
	QUESTION_ENTITY = "QUESTION"
	ANSWER_ENTITY   = "ANSWER"
)

func init() {
	goji.Post("/api/hunt", newHuntHandler)
	goji.Put("/api/hunt", newHuntHandler)
	goji.Get("/api/hunt", getAllHuntsHandler)
	goji.Get("/api/hunt/:hid", getHuntHandler)
	goji.Delete("/api/hunt", delAllHuntsHandler)
	goji.Delete("/api/hunt/:hid", delHuntHandler)

	// goji.Get("/api/image", getFormImgHandler)
	// goji.Post("/api/image", uploadImgHandler)

	goji.Serve()
}

func getMainDatastoreKey(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, "GDGTH_debug", "default", 0, nil)
}

func newHuntHandler(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	var hunt Hunt

	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		err := json.NewDecoder(req.Body).Decode(&hunt)
		if err != nil {
			return err
		}

		hunt_key, err := datastore.Put(ctx, datastore.NewKey(ctx, HUNT_ENTITY, hunt.Id, 0, getMainDatastoreKey(ctx)), &hunt)
		if err != nil {
			return err
		}

		for _, clue := range hunt.Clues {
			clue_key, err := datastore.Put(ctx, datastore.NewKey(ctx, CLUE_ENTITY, clue.Id, 0, hunt_key), &clue)
			if err != nil {
				return err
			}

			for _, tag := range clue.Tags {
				_, err := datastore.Put(ctx, datastore.NewKey(ctx, TAG_ENTITY, tag.Id, 0, clue_key), &tag)
				if err != nil {
					return err
				}
			}

			for _, question := range clue.Questions {
				question_key, err := datastore.Put(ctx, datastore.NewIncompleteKey(ctx, QUESTION_ENTITY, clue_key), &question)
				if err != nil {
					return err
				}

				for _, answer := range question.Answers {
					_, err := datastore.Put(ctx, datastore.NewIncompleteKey(ctx, ANSWER_ENTITY, question_key), &answer)
					if err != nil {
						return err
					}
				}
			}
		}

		return memcache.Flush(ctx)
	}, nil)

	if err != nil {
		log.Errorf(ctx, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getAllHuntsHandler(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	memcache_key := req.URL.String()

	var hunts []Hunt

	_, err := memcache.JSON.Get(ctx, memcache_key, &hunts)
	if err != nil {
		err = datastore.RunInTransaction(ctx, func(ctx context.Context) error {
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
					return err
				}
			}

			opt = req.URL.Query().Get("per_page")
			if opt != "" {
				per_page, err = strconv.Atoi(opt)
				if err != nil {
					return err
				}
			}

			offset := page * per_page

			hunts_query := datastore.NewQuery(HUNT_ENTITY).Ancestor(getMainDatastoreKey(ctx)).Limit(per_page).Offset(offset).Order(order)

			hunt_keys, err := hunts_query.GetAll(ctx, &hunts)
			if err != nil {
				return err
			}

			for hunt_id, hunt_key := range hunt_keys {
				clues_query := datastore.NewQuery(CLUE_ENTITY).Ancestor(hunt_key)

				clue_keys, err := clues_query.GetAll(ctx, &(hunts[hunt_id].Clues))
				if err != nil {
					return err
				}

				for clue_id, clue_key := range clue_keys {
					tags_query := datastore.NewQuery(TAG_ENTITY).Ancestor(clue_key)
					_, err := tags_query.GetAll(ctx, &(hunts[hunt_id].Clues[clue_id].Tags))
					if err != nil {
						return err
					}

					questions_query := datastore.NewQuery(QUESTION_ENTITY).Ancestor(clue_key)
					question_keys, err := questions_query.GetAll(ctx, &(hunts[hunt_id].Clues[clue_id].Questions))
					if err != nil {
						return err
					}

					for question_id, question_key := range question_keys {
						answers_query := datastore.NewQuery(ANSWER_ENTITY).Ancestor(question_key)
						_, err := answers_query.GetAll(ctx, &(hunts[hunt_id].Clues[clue_id].Questions[question_id].Answers))
						if err != nil {
							return err
						}
					}
				}
			}

			return memcache.JSON.Add(ctx, &memcache.Item{Key: memcache_key, Object: hunts})
		}, nil)
	}

	if err != nil {
		log.Errorf(ctx, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.New().JSON(w, http.StatusOK, hunts)
}

func getHuntHandler(c web.C, w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	memcache_key := req.URL.String()

	var hunt Hunt

	_, err := memcache.JSON.Get(ctx, memcache_key, &hunt)
	if err != nil {
		err = datastore.RunInTransaction(ctx, func(ctx context.Context) error {
			hunt_id := c.URLParams["hid"]
			hunt_key := datastore.NewKey(ctx, HUNT_ENTITY, hunt_id, 0, getMainDatastoreKey(ctx))

			err = datastore.Get(ctx, hunt_key, &hunt)
			if err != nil {
				return err
			}

			clues_query := datastore.NewQuery(CLUE_ENTITY).Ancestor(hunt_key)

			clue_keys, err := clues_query.GetAll(ctx, &(hunt.Clues))
			if err != nil {
				return err
			}

			for clue_id, clue_key := range clue_keys {
				tags_query := datastore.NewQuery(TAG_ENTITY).Ancestor(clue_key)
				_, err := tags_query.GetAll(ctx, &(hunt.Clues[clue_id].Tags))
				if err != nil {
					return err
				}

				questions_query := datastore.NewQuery(QUESTION_ENTITY).Ancestor(clue_key)
				question_keys, err := questions_query.GetAll(ctx, &(hunt.Clues[clue_id].Questions))
				if err != nil {
					return err
				}

				for question_id, question_key := range question_keys {
					answers_query := datastore.NewQuery(ANSWER_ENTITY).Ancestor(question_key)
					_, err := answers_query.GetAll(ctx, &(hunt.Clues[clue_id].Questions[question_id].Answers))
					if err != nil {
						return err
					}
				}
			}

			return memcache.JSON.Add(ctx, &memcache.Item{Key: memcache_key, Object: hunt})
		}, nil)
	}

	if err != nil {
		log.Errorf(ctx, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.New().JSON(w, http.StatusOK, hunt)
}

func delAllHuntsHandler(w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		hunt_keys, err := datastore.NewQuery(HUNT_ENTITY).Ancestor(getMainDatastoreKey(ctx)).KeysOnly().GetAll(ctx, nil)
		if err != nil {
			return err
		}

		err = datastore.DeleteMulti(ctx, hunt_keys)
		if err != nil {
			return err
		}

		for _, hunt_key := range hunt_keys {
			clue_keys, err := datastore.NewQuery(CLUE_ENTITY).Ancestor(hunt_key).KeysOnly().GetAll(ctx, nil)
			if err != nil {
				return err
			}

			err = datastore.DeleteMulti(ctx, clue_keys)
			if err != nil {
				return err
			}

			for _, clue_key := range clue_keys {
				tag_keys, err := datastore.NewQuery(TAG_ENTITY).Ancestor(clue_key).KeysOnly().GetAll(ctx, nil)
				if err != nil {
					return err
				}

				err = datastore.DeleteMulti(ctx, tag_keys)
				if err != nil {
					return err
				}

				question_keys, err := datastore.NewQuery(QUESTION_ENTITY).Ancestor(clue_key).KeysOnly().GetAll(ctx, nil)
				if err != nil {
					return err
				}

				err = datastore.DeleteMulti(ctx, question_keys)
				if err != nil {
					return err
				}

				for _, question_key := range question_keys {
					answer_keys, err := datastore.NewQuery(ANSWER_ENTITY).Ancestor(question_key).KeysOnly().GetAll(ctx, nil)
					if err != nil {
						return err
					}

					err = datastore.DeleteMulti(ctx, answer_keys)
					if err != nil {
						return err
					}
				}
			}
		}

		return memcache.Flush(ctx)
	}, nil)

	if err != nil {
		log.Errorf(ctx, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func delHuntHandler(c web.C, w http.ResponseWriter, req *http.Request) {
	ctx := appengine.NewContext(req)

	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		hunt_id := c.URLParams["hid"]
		hunt_key := datastore.NewKey(ctx, HUNT_ENTITY, hunt_id, 0, getMainDatastoreKey(ctx))

		err := datastore.Delete(ctx, hunt_key)
		if err != nil {
			return err
		}

		clue_keys, err := datastore.NewQuery(CLUE_ENTITY).Ancestor(hunt_key).KeysOnly().GetAll(ctx, nil)
		if err != nil {
			return err
		}

		err = datastore.DeleteMulti(ctx, clue_keys)
		if err != nil {
			return err
		}

		for _, clue_key := range clue_keys {
			tag_keys, err := datastore.NewQuery(TAG_ENTITY).Ancestor(clue_key).KeysOnly().GetAll(ctx, nil)
			if err != nil {
				return err
			}

			err = datastore.DeleteMulti(ctx, tag_keys)
			if err != nil {
				return err
			}

			question_keys, err := datastore.NewQuery(QUESTION_ENTITY).Ancestor(clue_key).KeysOnly().GetAll(ctx, nil)
			if err != nil {
				return err
			}

			err = datastore.DeleteMulti(ctx, question_keys)
			if err != nil {
				return err
			}

			for _, question_key := range question_keys {
				answer_keys, err := datastore.NewQuery(ANSWER_ENTITY).Ancestor(question_key).KeysOnly().GetAll(ctx, nil)
				if err != nil {
					return err
				}

				err = datastore.DeleteMulti(ctx, answer_keys)
				if err != nil {
					return err
				}
			}
		}

		return memcache.Flush(ctx)
	}, nil)

	if err != nil {
		log.Errorf(ctx, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
