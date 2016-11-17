package handler

import (
	"encoding/json"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	c "github.com/lastbackend/lastbackend/cmd/daemon/context"
	e "github.com/lastbackend/lastbackend/libs/errors"
	"github.com/lastbackend/lastbackend/libs/model"
	"io"
	"io/ioutil"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"net/http"
)

func ProjectListH(w http.ResponseWriter, r *http.Request) {

	var (
		err     *e.Err
		session *model.Session
		ctx     = c.Get()
	)

	ctx.Log.Debug("List project handler")

	s, ok := context.GetOk(r, `session`)
	if !ok {
		ctx.Log.Error("Error: get session context")
		e.User.AccessDenied().Http(w)
		return
	}

	session = s.(*model.Session)

	projects, err := ctx.Storage.Project().GetByUser(session.Uid)
	if err != nil {
		ctx.Log.Error("Error: find projects by user", err)
		e.HTTP.InternalServerError(w)
		return
	}

	response, err := projects.ToJson()
	if err != nil {
		ctx.Log.Error("Error: convert struct to json", err.Err())
		err.Http(w)
		return
	}

	w.WriteHeader(200)
	w.Write(response)
}

func ProjectInfoH(w http.ResponseWriter, r *http.Request) {

	var (
		err     *e.Err
		session *model.Session
		ctx     = c.Get()
		params  = mux.Vars(r)
		id      = params["id"]
	)

	ctx.Log.Debug("Get project handler")

	s, ok := context.GetOk(r, `session`)
	if !ok {
		ctx.Log.Error("Error: get session context")
		e.User.AccessDenied().Http(w)
		return
	}

	session = s.(*model.Session)

	project, err := ctx.Storage.Project().GetByID(session.Uid, id)
	if err == nil && project == nil {
		e.Project.NotFound().Http(w)
		return
	}
	if err != nil {
		ctx.Log.Error("Error: find project by id", err.Err())
		err.Http(w)
		return
	}

	response, err := project.ToJson()
	if err != nil {
		ctx.Log.Error("Error: convert struct to json", err.Err())
		err.Http(w)
		return
	}

	w.WriteHeader(200)
	w.Write(response)
}

type projectCreate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (s *projectCreate) decodeAndValidate(reader io.Reader) *e.Err {

	var (
		err error
		ctx = c.Get()
	)

	body, err := ioutil.ReadAll(reader)
	if err != nil {
		ctx.Log.Error(err)
		return e.User.Unknown(err)
	}

	err = json.Unmarshal(body, s)
	if err != nil {
		return e.Project.IncorrectJSON(err)
	}

	if s.Name == nil {
		return e.Project.BadParameter("name")
	}

	if s.Description == nil {
		s.Description = new(string)
	}

	return nil
}

func ProjectCreateH(w http.ResponseWriter, r *http.Request) {

	var (
		er      error
		err     *e.Err
		session *model.Session
		ctx     = c.Get()
	)

	ctx.Log.Debug("Create project handler")

	s, ok := context.GetOk(r, `session`)
	if !ok {
		ctx.Log.Error("Error: get session context")
		e.User.AccessDenied().Http(w)
		return
	}

	session = s.(*model.Session)

	// request body struct
	rq := new(projectCreate)
	if err := rq.decodeAndValidate(r.Body); err != nil {
		ctx.Log.Error("Error: validation incomming data", err)
		err.Http(w)
		return
	}

	p := new(model.Project)
	p.User = session.Uid
	p.Name = *rq.Name
	p.Description = *rq.Description

	project, err := ctx.Storage.Project().Insert(p)
	if err != nil {
		ctx.Log.Error("Error: insert project to db", err)
		e.HTTP.InternalServerError(w)
		return
	}

	namespace := &v1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name:      project.ID,
			Namespace: project.ID,
			Labels: map[string]string{
				"user": session.Username,
			},
		},
	}

	_, er = ctx.K8S.Core().Namespaces().Create(namespace)
	if er != nil {
		ctx.Log.Error("Error: create namespace", er.Error())
		e.HTTP.InternalServerError(w)
		return
	}

	response, err := project.ToJson()
	if err != nil {
		ctx.Log.Error("Error: convert struct to json", err.Err())
		err.Http(w)
		return
	}

	w.WriteHeader(200)
	w.Write(response)
}

type projectReplace struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (s *projectReplace) decodeAndValidate(reader io.Reader) *e.Err {

	var (
		err error
		ctx = c.Get()
	)

	body, err := ioutil.ReadAll(reader)
	if err != nil {
		ctx.Log.Error(err)
		return e.User.Unknown(err)
	}

	err = json.Unmarshal(body, s)
	if err != nil {
		return e.Project.IncorrectJSON(err)
	}

	if s.Name == nil {
		return e.Project.BadParameter("name")
	}

	if s.Description == nil {
		s.Description = new(string)
	}

	return nil
}

func ProjectUpdateH(w http.ResponseWriter, r *http.Request) {

	var (
		err     *e.Err
		session *model.Session
		ctx     = c.Get()
	)

	ctx.Log.Debug("Update project handler")

	s, ok := context.GetOk(r, `session`)
	if !ok {
		ctx.Log.Error("Error: get session context")
		e.User.AccessDenied().Http(w)
		return
	}

	session = s.(*model.Session)

	// request body struct
	rq := new(projectReplace)
	if err := rq.decodeAndValidate(r.Body); err != nil {
		ctx.Log.Error("Error: validation incomming data", err)
		err.Http(w)
		return
	}

	p := new(model.Project)
	p.User = session.Uid
	p.Name = *rq.Name
	p.Description = *rq.Description

	project, err := ctx.Storage.Project().Update(p)
	if err != nil {
		ctx.Log.Error("Error: insert project to db", err)
		e.HTTP.InternalServerError(w)
		return
	}

	response, err := project.ToJson()
	if err != nil {
		ctx.Log.Error("Error: convert struct to json", err.Err())
		err.Http(w)
		return
	}

	w.WriteHeader(200)
	w.Write(response)
}

func ProjectRemoveH(w http.ResponseWriter, r *http.Request) {

	var (
		ctx    = c.Get()
		params = mux.Vars(r)
		id     = params["id"]
	)

	ctx.Log.Info("Remove project")

	err := ctx.Storage.Project().Remove(id)
	if err != nil {
		ctx.Log.Error("Error: remove project from db", err)
		e.HTTP.InternalServerError(w)
		return
	}

	w.WriteHeader(200)
	w.Write([]byte{})
}