package ssmapi

import (
	"net/http"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing/common"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func (s *Server) setupRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Get("/info", s.getServerInfo)

		r.Get("/nodes", s.getNodes)
		r.Post("/nodes", s.addNode)
		r.Get("/nodes/{id}", s.getNode)
		r.Put("/nodes/{id}", s.updateNode)
		r.Delete("/nodes/{id}", s.deleteNode)

		r.Get("/users", s.getUsers)
		r.Post("/users", s.addUser)
		r.Get("/users/{username}", s.getUser)
		r.Put("/users/{username}", s.updateUser)
		r.Delete("/users/{username}", s.deleteUser)

		r.Get("/stats/data-usage", s.getDataUsage)
	})
}

func (s *Server) getServerInfo(writer http.ResponseWriter, request *http.Request) {
	render.PlainText(writer, request, "sing-box/"+C.Version)
}

func (s *Server) getNodes(writer http.ResponseWriter, request *http.Request) {
	var response struct {
		Protocols   []string                `json:"protocols"`
		Shadowsocks []ShadowsocksNodeObject `json:"shadowsocks,omitempty"`
	}
	for _, node := range s.nodes {
		if !common.Contains(response.Protocols, node.Protocol()) {
			response.Protocols = append(response.Protocols, node.Protocol())
		}
		switch node.Protocol() {
		case C.TypeShadowsocks:
			response.Shadowsocks = append(response.Shadowsocks, node.Shadowsocks())
		}
	}
	render.JSON(writer, request, &response)
}

func (s *Server) addNode(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) getNode(writer http.ResponseWriter, request *http.Request) {
	nodeID := chi.URLParam(request, "id")
	if nodeID == "" {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	for _, node := range s.nodes {
		if nodeID == node.ID() {
			render.JSON(writer, request, render.M{
				node.Protocol(): node.Object(),
			})
			return
		}
	}
}

func (s *Server) updateNode(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) deleteNode(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusNotImplemented)
}

type SSMUserObject struct {
	UserName      string `json:"username"`
	Password      string `json:"uPSK,omitempty"`
	DownlinkBytes int64  `json:"downlinkBytes"`
	UplinkBytes   int64  `json:"uplinkBytes"`
}

func (s *Server) getUsers(writer http.ResponseWriter, request *http.Request) {
	users := s.userManager.List()
	uplinkList, downlinkList := s.trafficManager.ReadUsers(common.Map(users, func(it SSMUserObject) string {
		return it.UserName
	}))
	for i := range users {
		users[i].DownlinkBytes = downlinkList[i]
		users[i].UplinkBytes = uplinkList[i]
	}
	render.JSON(writer, request, render.M{
		"users": users,
	})
}

func (s *Server) addUser(writer http.ResponseWriter, request *http.Request) {
	var addRequest struct {
		UserName string `json:"username"`
		Password string `json:"uPSK"`
	}
	err := render.DecodeJSON(request.Body, &addRequest)
	if err != nil {
		render.Status(request, http.StatusBadRequest)
		render.PlainText(writer, request, err.Error())
		return
	}
	err = s.userManager.Add(addRequest.UserName, addRequest.Password)
	if err != nil {
		render.Status(request, http.StatusBadRequest)
		render.PlainText(writer, request, err.Error())
		return
	}
	writer.WriteHeader(http.StatusCreated)
}

func (s *Server) getUser(writer http.ResponseWriter, request *http.Request) {
	userName := chi.URLParam(request, "username")
	if userName == "" {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	uPSK, loaded := s.userManager.Get(userName)
	if !loaded {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	uplink, downlink := s.trafficManager.ReadUser(userName)
	render.JSON(writer, request, SSMUserObject{
		UserName:      userName,
		Password:      uPSK,
		DownlinkBytes: downlink,
		UplinkBytes:   uplink,
	})
}

func (s *Server) updateUser(writer http.ResponseWriter, request *http.Request) {
	userName := chi.URLParam(request, "username")
	if userName == "" {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	var updateRequest struct {
		Password string `json:"uPSK"`
	}
	err := render.DecodeJSON(request.Body, &updateRequest)
	if err != nil {
		render.Status(request, http.StatusBadRequest)
		render.PlainText(writer, request, err.Error())
		return
	}
	_, loaded := s.userManager.Get(userName)
	if !loaded {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	err = s.userManager.Update(userName, updateRequest.Password)
	if err != nil {
		render.Status(request, http.StatusBadRequest)
		render.PlainText(writer, request, err.Error())
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteUser(writer http.ResponseWriter, request *http.Request) {
	userName := chi.URLParam(request, "username")
	if userName == "" {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	_, loaded := s.userManager.Get(userName)
	if !loaded {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	err := s.userManager.Delete(userName)
	if err != nil {
		render.Status(request, http.StatusBadRequest)
		render.PlainText(writer, request, err.Error())
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) getDataUsage(writer http.ResponseWriter, request *http.Request) {
	users := s.userManager.List()
	uplinkList, downlinkList := s.trafficManager.ReadUsers(common.Map(users, func(it SSMUserObject) string {
		return it.UserName
	}))
	for i := range users {
		users[i].Password = ""
		users[i].DownlinkBytes = downlinkList[i]
		users[i].UplinkBytes = uplinkList[i]
	}
	uplink, downlink := s.trafficManager.ReadGlobal()
	render.JSON(writer, request, render.M{
		"downlinkBytes": downlink,
		"uplinkBytes":   uplink,
		"users":         users,
	})
}
