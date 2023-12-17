package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleRegister(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	uid, token, code, msg := s.handleRegisterInner(c)
	if code == CodeSuccess {
		respWrapper.Resp = RegisterResponse{
			ID:    idN2S(uid),
			Token: token,
		}
	}

	respWrapper.Apply(code, msg)

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) handleRegisterInner(c *gin.Context) (uid uint64, token string, code Code, msg string) {
	var req RegisterRequest

	err := c.BindJSON(&req)
	if err != nil {
		code = CodeProtocol
		msg = err.Error()

		return
	}

	if !req.Valid() {
		code = CodeMissArgs

		return
	}

	_, err = s.accounts.Register(req.UserName, req.Password)
	if err != nil {
		code = CodeInternalError

		return
	}

	uid, token, err = s.accounts.Login(req.UserName, req.Password)
	if err != nil {
		code = CodeInternalError
		msg = err.Error()

		return
	}

	_, _, _ = s.storage.NewPersonEx(req.UserName, uid)

	code = CodeSuccess

	return
}

func (s *Server) handleLogin(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	uid, token, code, msg := s.handleLoginInner(c)
	if code == CodeSuccess {
		respWrapper.Resp = &LoginResponse{
			ID:    idN2S(uid),
			Token: token,
		}
	}

	respWrapper.Apply(code, msg)

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) handleLoginInner(c *gin.Context) (uid uint64, token string, code Code, msg string) {
	var req LoginRequest

	err := c.BindJSON(&req)
	if err != nil {
		code = CodeProtocol
		msg = err.Error()

		return
	}

	if !req.Valid() {
		code = CodeMissArgs

		return
	}

	uid, token, err = s.accounts.Login(req.UserName, req.Password)
	if err != nil {
		code = CodeVerifyFailed

		return
	}

	code = CodeSuccess

	return
}

func (s *Server) handleLogout(c *gin.Context) {
	respWrapper := &ResponseWrapper{}

	respWrapper.Apply(s.handleLogoutInner(c))

	c.JSON(http.StatusOK, respWrapper)
}

func (s *Server) getAndCheckToken(c *gin.Context) (token string, uid uint64, userName string, code Code, msg string) {
	token = c.GetHeader("token")
	if token == "" {
		code = CodeNeedAuth
		msg = "need auth"

		return
	}

	uid, userName, err := s.accounts.Who(token)
	if err != nil {
		code = CodeInvalidToken
		msg = "invalid token"
	} else {
		code = CodeSuccess
	}

	return
}

func (s *Server) handleLogoutInner(c *gin.Context) (code_ Code, msg string) {
	code_ = CodeSuccess

	token, _, _, code, msg := s.getAndCheckToken(c)
	if code != CodeSuccess {
		return
	}

	err := s.accounts.Logout(token)
	if err != nil {
		msg = err.Error()

		return
	}

	return
}
