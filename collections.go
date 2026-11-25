package main

import (
	"database/sql"
	"strings"
)

type Collection struct {
	CollectionID ID               `json:"collection_id"`
	Name         string           `json:"name"`
	Kind         string           `json:"kind"`
	Count        int              `json:"count"`
	Items        []CollectionItem `json:"items"`
}
type CollectionItem struct {
	ItemID     ID      `json:"item_id"`
	CorpusID   *int    `json:"corpus_id"`
	QuestionID ID      `json:"question_id"`
	Corpus     *Record `json:"corpus"`
	QuizType   *string `json:"quiz_type"`
	Variant    *string `json:"variant"`
}
type CollectionRequest struct {
	Name   string  `json:"name"`
	Kind   string  `json:"kind"`
	Action *Action `json:"action"`
}
type ItemRequest struct {
	CorpusID       *int    `json:"corpus_id"`
	QuestionID     ID      `json:"question_id"`
	AllowDuplicate bool    `json:"allow_duplicate"`
	Action         *Action `json:"action"`
}

func (s *Store) ListCollections() ([]Collection, error) {
	rows, err := s.db.Query("SELECT c.collection_id,c.name,c.kind,(SELECT COUNT(*) FROM collection_items i WHERE i.collection_id=c.collection_id) FROM collections c ORDER BY c.collection_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Collection{}
	for rows.Next() {
		v := Collection{Items: []CollectionItem{}}
		if err = rows.Scan(&v.CollectionID, &v.Name, &v.Kind, &v.Count); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) GetCollection(id ID) (Collection, error) {
	v := Collection{Items: []CollectionItem{}}
	if err := s.db.QueryRow("SELECT collection_id,name,kind FROM collections WHERE collection_id=?", id).Scan(&v.CollectionID, &v.Name, &v.Kind); err != nil {
		return v, asNotFound(err, "Collection")
	}
	rows, err := s.db.Query(`SELECT i.item_id,i.corpus_id,i.question_id,q.quiz_type,q.variant,q.parameters
FROM collection_items i LEFT JOIN question_specs q ON q.question_id=i.question_id WHERE i.collection_id=? ORDER BY i.item_id`, id)
	if err != nil {
		return v, err
	}
	defer rows.Close()
	for rows.Next() {
		item := CollectionItem{}
		var qid sql.NullInt64
		var params *string
		if err = rows.Scan(&item.ItemID, &item.CorpusID, &qid, &item.QuizType, &item.Variant, &params); err != nil {
			return v, err
		}
		if qid.Valid {
			item.QuestionID = ID(qid.Int64)
			variant := ""
			if item.Variant != nil {
				variant = *item.Variant
			}
			p, e := s.parseSpec(QuestionSpec{item.QuestionID, *item.QuizType, variant, *params})
			if e != nil {
				return v, e
			}
			item.Corpus = s.Corpus.ByID[p[0]]
		} else {
			item.Corpus = s.Corpus.ByID[*item.CorpusID]
		}
		v.Items = append(v.Items, item)
	}
	v.Count = len(v.Items)
	return v, rows.Err()
}

func operationScreen(a *Action, defaultScreen string) string {
	if a != nil && a.Screen != "" {
		return a.Screen
	}
	return defaultScreen
}

func (s *Store) CreateCollection(r CollectionRequest) (Collection, error) {
	r.Name = strings.TrimSpace(r.Name)
	if len([]rune(r.Name)) < 1 || len([]rune(r.Name)) > 120 {
		return Collection{}, bad("Collection name must contain 1..120 characters")
	}
	if r.Kind != "vocabulary" && r.Kind != "question" {
		return Collection{}, bad("Collection kind must be vocabulary or question")
	}
	var id ID
	err := s.transaction(func(tx *sql.Tx) error {
		var err error
		id, err = allocate(tx, "collections", "collection_id")
		if err != nil {
			return err
		}
		if _, err = tx.Exec("INSERT INTO collections(collection_id,name,kind) VALUES(?,?,?)", id, r.Name, r.Kind); err != nil {
			return err
		}
		a, err := actionFor(tx, r.Action, operationScreen(r.Action, "dictionary"), "create", "collection:"+id.String(), 0, 0)
		if err != nil {
			return err
		}
		a.Value = ptrString(r.Name)
		_, err = insertAction(tx, a)
		return err
	})
	if err != nil {
		return Collection{}, err
	}
	return s.GetCollection(id)
}

func (s *Store) AddCollectionItem(id ID, r ItemRequest) (CollectionItem, error) {
	var itemID ID
	err := s.transaction(func(tx *sql.Tx) error {
		var kind string
		if err := tx.QueryRow("SELECT kind FROM collections WHERE collection_id=?", id).Scan(&kind); err != nil {
			return asNotFound(err, "Collection")
		}
		var duplicate int
		if kind == "vocabulary" {
			if r.CorpusID == nil || r.QuestionID != 0 || s.Corpus.ByID[*r.CorpusID] == nil {
				return bad("Vocabulary Collection requires a valid corpus_id only")
			}
			if err := tx.QueryRow("SELECT COUNT(*) FROM collection_items WHERE collection_id=? AND corpus_id=?", id, *r.CorpusID).Scan(&duplicate); err != nil {
				return err
			}
		} else {
			if r.CorpusID != nil || r.QuestionID == 0 {
				return bad("Question Collection requires question_id only")
			}
			if _, err := s.getSpec(tx, r.QuestionID); err != nil {
				return err
			}
			if err := tx.QueryRow("SELECT COUNT(*) FROM collection_items WHERE collection_id=? AND question_id=?", id, r.QuestionID).Scan(&duplicate); err != nil {
				return err
			}
		}
		if duplicate > 0 && !r.AllowDuplicate {
			return &AppError{409, "duplicate", "This ID is already registered. Confirm to add another independent registration."}
		}
		var err error
		itemID, err = allocate(tx, "collection_items", "item_id")
		if err != nil {
			return err
		}
		if _, err = tx.Exec("INSERT INTO collection_items(item_id,collection_id,corpus_id,question_id) VALUES(?,?,?,?)", itemID, id, r.CorpusID, nullID(r.QuestionID)); err != nil {
			return err
		}
		a, err := actionFor(tx, r.Action, operationScreen(r.Action, "dictionary"), "add", "collection_item:"+itemID.String(), 0, 0)
		if err != nil {
			return err
		}
		a.Value = ptrString("collection:" + id.String())
		_, err = insertAction(tx, a)
		return err
	})
	if err != nil {
		return CollectionItem{}, err
	}
	collection, err := s.GetCollection(id)
	if err != nil {
		return CollectionItem{}, err
	}
	for _, item := range collection.Items {
		if item.ItemID == itemID {
			return item, nil
		}
	}
	return CollectionItem{}, notFound("Registration disappeared")
}

func (s *Store) DeleteCollectionItem(id, itemID ID, a *Action) error {
	return s.transaction(func(tx *sql.Tx) error {
		res, err := tx.Exec("DELETE FROM collection_items WHERE collection_id=? AND item_id=?", id, itemID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return notFound("Collection item not found")
		}
		action, err := actionFor(tx, a, operationScreen(a, "dictionary"), "remove", "collection_item:"+itemID.String(), 0, 0)
		if err != nil {
			return err
		}
		action.Value = ptrString("collection:" + id.String())
		_, err = insertAction(tx, action)
		return err
	})
}

func (s *Store) DeleteCollection(id ID, a *Action) error {
	return s.transaction(func(tx *sql.Tx) error {
		res, err := tx.Exec("DELETE FROM collections WHERE collection_id=?", id)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return notFound("Collection not found")
		}
		action, err := actionFor(tx, a, operationScreen(a, "dictionary"), "delete", "collection:"+id.String(), 0, 0)
		if err != nil {
			return err
		}
		_, err = insertAction(tx, action)
		return err
	})
}
