package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/s-min-sys/lifecostbe/internal/config"
	"github.com/s-min-sys/lifecostbe/internal/storage"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libcomponents/account"
	"github.com/sgostarter/libcomponents/account/impls/fmaccountstorage"
	"github.com/sgostarter/libeasygo/routineman"
)

const (
	dataRoot = "data"
)

type Server struct {
	routineMan routineman.RoutineMan
	cfg        *config.Config
	logger     l.Wrapper

	accounts account.Account
	storage  storage.Storage
}

func NewServer(ctx context.Context, routineMan routineman.RoutineMan, cfg *config.Config, logger l.Wrapper) *Server {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	if routineMan == nil {
		routineMan = routineman.NewRoutineMan(ctx, logger)
	}

	if cfg == nil || !cfg.Valid() {
		logger.Error("no valid config")

		return nil
	}

	s := &Server{
		routineMan: routineMan,
		cfg:        cfg,
		logger:     logger.WithFields(l.StringField(l.ClsKey, "Server")),
		accounts: account.NewAccount(fmaccountstorage.NewFMAccountStorageEx(dataRoot, nil, cfg.Debug),
			&cfg.AccountConfig, logger),
		storage: storage.NewStorage(dataRoot, cfg.Debug, logger),
	}

	s.init()

	return s
}

func (s *Server) Wait() {
	s.routineMan.Wait()
}

func (s *Server) init() {
	s.routineMan.StartRoutine(s.httpRoutine, "httpRoutine")
}

func JSONMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Next()
	}
}

func (s *Server) httpRoutine(ctx context.Context, exiting func() bool) {
	logger := s.logger.WithFields(l.StringField(l.RoutineKey, "httpRoutine"))

	logger.Debug("enter")

	defer logger.Debug("leave")

	if s.cfg.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(gin.Recovery())
	r.Use(requestid.New())
	r.Use(JSONMiddleware())

	r.Any("/healthy", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	r.POST("/register", s.handleRegister)
	r.POST("/login", s.handleLogin)
	r.POST("/logout", s.handleLogout)
	r.GET("/base-infos", s.handleGetBaseInfos)
	r.POST("/record", s.handleRecord)
	r.POST("/records", s.handleGetRecords)
	r.GET("/statistics", s.handleStatistics)

	r.POST("/manager/wallet/new", s.handleWalletNew)
	r.POST("/manager/group/new", s.handleGroupNew)
	r.POST("/manager/group/enter-codes", s.handleGroupEnterCodes)
	r.POST("/manager/group/join/:code", s.handleGroupJoin)
	r.POST("/manager/wallet/new-by-dir", s.handleWalletNewByDir)

	fnListen := func(listen string) {
		srv := &http.Server{
			Addr:        listen,
			ReadTimeout: time.Second,
			Handler:     r,
		}

		logger.WithFields(l.StringField("listen", listen)).Debug("start listen")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.WithFields(l.ErrorField(err), l.StringField("listen", listen)).Error("listen failed")
		}
	}

	listens := strings.Split(s.cfg.Listen, " ")

	for idx := 0; idx < len(listens)-1; idx++ {
		go fnListen(listens[idx])
	}

	fnListen(listens[len(listens)-1])
}
