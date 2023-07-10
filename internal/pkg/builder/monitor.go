package builder

import (
	"net/http"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/service/account"
  "github.com/rikikudohust-thesis/l2node/internal/pkg/service/transaction"
  "github.com/rikikudohust-thesis/l2node/internal/pkg/service/exit"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type setupRoute func(router *gin.RouterGroup, db *gorm.DB, redis model.IService)

type Server struct {
	engine *gin.Engine
}

func NewServer() (*Server, error) {
	dbCfg, rCfg, err := LoadEnvConfig()
	if err != nil {
		return nil, err
	}

	db, err := NewPostgres(dbCfg)
	if err != nil {
		return nil, err
	}

	r, err := New(rCfg)
	if err != nil {
		return nil, err
	}

	setup := []setupRoute{
		account.SetupRouter,
    transaction.SetupRouter,
    exit.SetupRouter,
	}

	engine, router := newHTTP("/v1/zkPayment")
	for _, route := range setup {
		route(router.Group("/"), db, r)
	}

	return &Server{engine: engine}, nil
}

func (s *Server) Run() error {
  return s.engine.Run()
}

func newHTTP(rootPath string) (*gin.Engine, *gin.RouterGroup) {
	server := gin.Default()
	setCORS(server)
	server.GET("/ping", func(c *gin.Context) { c.AbortWithStatus(http.StatusOK) })
	router := server.Group(rootPath)
	return server, router
}

func setCORS(engine *gin.Engine) {
	corsConfig := cors.DefaultConfig()
	corsConfig.AddAllowMethods(http.MethodOptions)
	corsConfig.AllowAllOrigins = true
	corsConfig.AddAllowHeaders("Authorization")
	corsConfig.AddAllowHeaders("x-request-id")
	corsConfig.AddAllowHeaders("X-Request-Id")
	engine.Use(cors.New(corsConfig))
}
