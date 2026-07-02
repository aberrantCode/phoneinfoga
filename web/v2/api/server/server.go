package server

import (
	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/handlers"
	"net/http"
)

type Server struct {
	router *gin.Engine
}

func NewServer() *Server {
	gin.DefaultWriter = color.Output
	gin.DefaultErrorWriter = color.Error
	s := &Server{
		router: gin.Default(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.router.Group("/v2").
		POST("/numbers", api.WrapHandler(handlers.AddNumber)).
		POST("/scanners/:scanner/dryrun", api.WrapHandler(handlers.DryRunScanner)).
		POST("/scanners/:scanner/run", api.WrapHandler(handlers.RunScanner)).
		GET("/scanners", api.WrapHandler(handlers.GetAllScanners)).
		POST("/lookups", api.WrapHandler(handlers.CreateLookup)).
		POST("/lookups/:id/close", api.WrapHandler(handlers.CloseLookup)).
		GET("/lookups", api.WrapHandler(handlers.ListLookups)).
		GET("/lookups/latest", api.WrapHandler(handlers.GetLatestLookup)).
		GET("/lookups/:id", api.WrapHandler(handlers.GetLookup))
}

func (s *Server) Routes() gin.RoutesInfo {
	return s.router.Routes()
}

func (s *Server) ListenAndServe(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
