package main

import (
	"context"
	"database/sql"
	"errors"
	sq "github.com/Masterminds/squirrel"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"log"
	"net/http"
	"strconv"
)

type Post struct {
	Id int64 `json:"id"`
	Created sql.NullTime `json:"created,omitempty"`
	Updated sql.NullTime `json:"updated,omitempty"`
	ThreadId int64 `json:"thread_id,omitempty"`
	UserId int64 `json:"user_id, omitempty"`
	Content string `json:"content,omitempty"`
}
func (p *Post) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
type PostRequest struct {
	*Post
}
func (pr *PostRequest) Bind(r *http.Request) error {
	if pr.Content == "" || pr.ThreadId < 1 {
		return errors.New("empty content or bad thread id")
	}
	return nil
}
func ThreadExists(id int64) bool {
	var selectedId int64
	result, err := sq.Select("id").From("threads").Where(sq.Eq{"id": id}).
		RunWith(DB.db).Query()
	if err != nil {
		return false
	}
	for result.Next() {
		result.Scan(&selectedId)
	}
	return selectedId == id
}
type Thread struct {
	Id int64 `json:"id"`
	Title string `json:"title,omitempty"`
	NodeId int64 `json:"node_id,omitempty"`
}
func (t *Thread) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
func (t *Thread) NewPost(userId int64, content string) (int64, error) {
	result, err := sq.Insert("posts").Columns("thread_id", "user_id", "content").
		Values(t.Id, userId, content).RunWith(DB.db).Exec()
	if err != nil {
		return -1, err
	}
	postId, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	return postId, nil
}
func (t *Thread) Posts () []Post {
	result, err := sq.Select("created", "user_id", "content").From("posts").
		Where(sq.Eq{"thread_id": t.Id}).OrderBy("created").RunWith(DB.db).Query()
	if err != nil {
		return []Post{}
	}
	var (
		Created sql.NullTime
		UserId int64
		Content string
	)
	var posts []Post
	for result.Next() {
		err := result.Scan(&Created, &UserId, &Content)
		if err != nil {
			log.Println(err)
			return []Post{}
		}
		posts = append(posts, Post{
			Created: Created,
			UserId: UserId,
			Content: Content,
		})
	}
	return posts
}

type ThreadNode struct {
	Id int64 `json:"id"`
	Slug string `json:"slug,omitempty"`
}

func ThreadNodeFromSlug(slug string) (*ThreadNode, error){
	result, err := sq.Select("id").From("nodes").Where(sq.Eq{"slug": slug}).
		RunWith(DB.db).Query()
	if err != nil {
		return nil, err
	}
	var nodeId int64
	for result.Next() {
		err := result.Scan(&nodeId)
		if err != nil {
			return nil, err
		}
	}
	return &ThreadNode{
		Id: nodeId,
		Slug: slug,
	}, nil
}
func (tn *ThreadNode) NewThread(userId int64, title string, postContent string) (int64, error) {
	result, err := sq.Insert("threads").Columns("title", "node_id").
		Values(title, tn.Id).RunWith(DB.db).Exec()
	if err != nil {
		return -1, err
	}
	threadId, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	thread := Thread{Id: threadId}
	_, err = thread.NewPost(userId, postContent)
	if err != nil {
		return -1, err
	}
	return threadId, nil
}
func (tn *ThreadNode) Threads() ([]*Thread, error) {
	result, err := sq.Select("id", "title").From("threads").Where(sq.Eq{"node_id": tn.Id}).
		RunWith(DB.db).Query()
	if err != nil {
		return nil, err
	}
	var threads []*Thread
	var (
		threadId int64
		threadTitle string
	)
	for result.Next() {
		err := result.Scan(&threadId, &threadTitle)
		if err != nil {
			return nil, err
		}

		threads = append(threads, &Thread{
			Id: threadId,
			Title: threadTitle,
		})
	}
	return threads, nil
}
type ThreadNodeRequest struct {
	NodeId int64 `json:"node_id"`
	Title string `json:"title"`
	Content string `json:"content"`
}
func (tnr *ThreadNodeRequest) Bind(r *http.Request) error {
	if tnr.NodeId < 1 || tnr.Title == "" || tnr.Content == ""{
		return errors.New("invalid id/title/post")
	}
	return nil
}
func ThreadsRouter() chi.Router {
	r := chi.NewRouter()
	r.With(NodeCtx).With(paginate).Get("/{nodeSlug}", ListThreads)
	r.With(ThreadCtx).With(paginate).Get("/{nodeSlug}/{threadId}", ListPosts)
	r.With(ConvoUserTokenCtx).Post("/", HandleNewThread)
	r.With(ConvoUserTokenCtx).Post("/posts", HandleNewPost)
	return r

}
func ThreadCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var thread *Thread
		var err error
		var id int
		if id, err = strconv.Atoi(chi.URLParam(r, "threadId")); id < 1 || err != nil {
			if Debug {
				log.Println("thread id", id, err)
			}
			render.Render(w, r, ErrNotFound)
			return
		}
		if !ThreadExists(int64(id)) {
			render.Render(w, r, ErrNotFound)
			return
		}
		thread = &Thread{
			Id: int64(id),
		}

		ctx := context.WithValue(r.Context(), "thread", thread)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func NodeCtx (next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var node *ThreadNode
		var err error

		if slug := chi.URLParam(r, "nodeSlug"); slug != "" {
			node, err = ThreadNodeFromSlug(slug)
		} else {
			render.Render(w, r, ErrNotFound)
			return
		}
		if err != nil {
			render.Render(w, r, ErrNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), "node", node)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ListThreads(w http.ResponseWriter, r *http.Request) {
	node := r.Context().Value("node").(*ThreadNode)
	threads, err := node.Threads()
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	var renderers []render.Renderer
	for index, _ := range threads {
		renderers = append(renderers, threads[index])
	}
	render.RenderList(w, r, renderers)

}
func ListPosts(w http.ResponseWriter, r *http.Request) {
	thread := r.Context().Value("thread").(*Thread)
	posts := thread.Posts()
	var renderers []render.Renderer
	for i, _ := range posts {
		renderers = append(renderers, &posts[i])
	}
	render.RenderList(w, r, renderers)
}
func HandleNewPost(w http.ResponseWriter, r *http.Request) {
	pr := &PostRequest{}
	if err := render.Bind(r, pr); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	user := r.Context().Value("convouser").(*ConvoUser)
	thread := &Thread{
		Id: pr.ThreadId,
	}
	postId, err := thread.NewPost(user.ID, pr.Content)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	render.Render(w, r, &Post{Id: postId})
}
func HandleNewThread(w http.ResponseWriter, r *http.Request) {
	tnr := &ThreadNodeRequest{}
	if err := render.Bind(r, tnr); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	user := r.Context().Value("convouser").(*ConvoUser)
	threadNode := ThreadNode{
		Id: tnr.NodeId,
	}
	threadId, err := threadNode.NewThread(user.ID, tnr.Title, tnr.Content)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	render.Render(w, r, &Thread{Id: threadId})
}