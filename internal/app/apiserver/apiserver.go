package apiserver

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"gorestapi/internal/app/model"
	"gorestapi/internal/app/store"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

var jwtSecretKey = []byte(os.Getenv("JWTSKEY"))

var caCert *x509.CertPool
var ownCert tls.Certificate

type Claims struct {
	Sub       int    `json:"sub"`
	UserEmail string `json:"userEmail"`

	jwt.RegisteredClaims
}

type APIServer struct {
	config *Config
	logger *logrus.Logger
	router *mux.Router
	store  *store.Store
}

func New(conf *Config) *APIServer {
	return &APIServer{
		config: conf,
		logger: logrus.New(),
		router: mux.NewRouter(),
	}
}

func (s *APIServer) configureLogger() error {
	level, err := logrus.ParseLevel(s.config.LogLevel)
	if err != nil {
		return err
	}

	s.logger.SetLevel(level)

	return nil
}

func (s *APIServer) Start() error {
	caPem, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		return err
	}

	caCert = x509.NewCertPool()
	if ok := caCert.AppendCertsFromPEM(caPem); !ok {
		return err
	}

	ownCert, err = tls.LoadX509KeyPair("certs/profiles.crt", "certs/profiles.key")
	if err != nil {
		return err
	}

	if err := s.configureLogger(); err != nil {
		return err
	}

	if err := s.configureStore(); err != nil {
		return err
	}

	s.configureRouter()

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"X-Requested-With", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           86400,
	})

	fmt.Println(caCert)
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		//ClientAuth:   tls.RequireAndVerifyClientCert,
		//ClientCAs:    caCert,
		//Certificates: []tls.Certificate{ownCert},
	}

	server := &http.Server{
		Addr:      s.config.BindAddr,
		Handler:   corsMiddleware.Handler(s.router),
		TLSConfig: tlsConfig,
	}

	s.logger.Info("starting api server")

	return server.ListenAndServeTLS("certs/profiles.crt", "certs/profiles.key")
}

func (s *APIServer) configureRouter() {
	s.router.HandleFunc("/api/profiles/me", s.apiGetMyProfile()).Methods("GET")
	s.router.HandleFunc("/api/profiles", s.apiGetProfilesByPattern()).Methods("GET")
	s.router.HandleFunc("/api/profiles/{username}", s.apiGetProfile()).Methods("GET")
	s.router.HandleFunc("/api/profiles/subscribe/{username}", s.apiSubscribe()).Methods("POST")
	s.router.HandleFunc("/api/profiles/unsubscribe/{username}", s.apiUnsubscribe()).Methods("POST")
	s.router.HandleFunc("/api/profiles/crprofile", s.apiCreateProfile()).Methods("POST")
	s.router.HandleFunc("/api/profiles/followers/{username}", s.apiGetFollowers()).Methods("GET")
	s.router.HandleFunc("/api/profiles/followed/{username}", s.apiGetFollowees()).Methods("GET")
}

func (s *APIServer) configureStore() error {
	st := store.New(s.config.Store)
	if err := st.Open(); err != nil {
		return err
	}
	s.store = st

	return nil
}

func (s *APIServer) apiResult(resp http.ResponseWriter, req *http.Request, loglevel string, res int, info string) {
	resLog := fmt.Sprintf("%s request [%s]: %s - %d", info, req.RemoteAddr, req.Method, res)
	switch loglevel {
	case "INFO":
		s.logger.Info(resLog)
	case "DEBUG":
		s.logger.Debug(resLog)
	case "WARNING":
		s.logger.Warn(resLog)
	}
	resp.WriteHeader(res)
}

func parseJWT(r *http.Request) (*Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("invalid Auth header")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecretKey, nil
	})

	fmt.Println(err)
	if err != nil {
		return nil, fmt.Errorf("faild parse JWT")
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid JWT")
}

func (s *APIServer) apiCreateProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Body == nil {
			s.apiResult(w, r, "DEBUG", http.StatusBadRequest, "ReadBody")
			return
		}
		defer r.Body.Close()

		data, err := io.ReadAll(r.Body)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusUnprocessableEntity, "ReadBody")
			return
		}

		reqProfile := &model.Profile{}

		if err = json.Unmarshal(data, reqProfile); err != nil {
			fmt.Println(err)
			s.apiResult(w, r, "DEBUG", http.StatusBadRequest, "Unmarshal")
			return
		}

		_, err = s.store.Profile().CreateProfile(reqProfile)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusNotFound, "Create")
			return
		}

		s.apiResult(w, r, "DEBUG", http.StatusOK, "CreateProfile")
	}
}

func (s *APIServer) apiGetFollowees() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		/*jwtClaims, err := parseJWT(r)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusUnauthorized, "Parse JWT")
			return
		}

		//TODO: Добавить проверку токена
		fmt.Println(jwtClaims.Sub)
		*/
		pathVars := mux.Vars(r)

		profiles, err := s.store.Profile().GetFollowees(pathVars["username"])
		if err != nil {
			fmt.Println(err)
			s.apiResult(w, r, "DEBUG", http.StatusNotFound, "GetFollowees")
			return
		}

		byteProfiles, err := json.Marshal(profiles)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusInternalServerError, "Serialize profile")
			return
		}

		w.Header().Set("Content-Type", "application/json")

		s.apiResult(w, r, "DEBUG", http.StatusOK, "GetFollowees")
		w.Write(byteProfiles)
	}
}
func (s *APIServer) apiGetFollowers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jwtClaims, err := parseJWT(r)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusUnauthorized, "Parse JWT")
			return
		}

		//TODO: Добавить проверку токена
		fmt.Println(jwtClaims.Sub)

		pathVars := mux.Vars(r)

		profiles, err := s.store.Profile().GetFollowers(pathVars["username"])
		if err != nil {
			fmt.Println(err)
			s.apiResult(w, r, "DEBUG", http.StatusNotFound, "GetFollowers")
			return
		}

		byteProfiles, err := json.Marshal(profiles)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusInternalServerError, "Serialize profile")
			return
		}

		w.Header().Set("Content-Type", "application/json")

		s.apiResult(w, r, "DEBUG", http.StatusOK, "GetFollowers")
		w.Write(byteProfiles)
	}
}
func (s *APIServer) apiSubscribe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jwtClaims, err := parseJWT(r)
		fmt.Println(err)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusUnauthorized, "Parse JWT")
			return
		}
		//TODO: Добавить проверку токена
		fmt.Println(jwtClaims.Sub)

		pathVars := mux.Vars(r)

		//sub, err := strconv.Atoi(jwtClaims.Subject)
		fmt.Println(jwtClaims.Sub)
		err = s.store.Profile().Subscribe(jwtClaims.Sub, pathVars["username"])
		fmt.Println(err)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusNotFound, "Subscribe")
			return
		}

		s.apiResult(w, r, "DEBUG", http.StatusOK, "Subscribe")
	}
}

func (s *APIServer) apiUnsubscribe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jwtClaims, err := parseJWT(r)
		fmt.Println(err)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusUnauthorized, "Parse JWT")
			return
		}
		//TODO: Добавить проверку токена
		fmt.Println(jwtClaims.Sub)

		pathVars := mux.Vars(r)

		err = s.store.Profile().Unsubscribe(jwtClaims.Sub, pathVars["username"])
		fmt.Println(err)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusNotFound, "Unsubscribe")
			return
		}

		s.apiResult(w, r, "DEBUG", http.StatusOK, "Unsubscribe")
	}
}

func (s *APIServer) apiGetMyProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jwtClaims, err := parseJWT(r)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusUnauthorized, "Parse JWT")
			return
		}
		//TODO: Добавить проверку токена

		reqProfile, err := s.store.Profile().FindProfileByID(jwtClaims.Sub)

		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusNotFound, "FindByID")
			return
		}

		reqProfile.IsOwnProfile = true
		reqProfile.IsFollowed = true

		byteProfile, err := json.Marshal(reqProfile)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusInternalServerError, "Serialize profile")
			return
		}

		w.Header().Set("Content-Type", "application/json")

		s.apiResult(w, r, "DEBUG", http.StatusOK, "Get my profile")
		w.Write(byteProfile)
	}
}
func (s *APIServer) apiGetProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jwtClaims, err := parseJWT(r)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusUnauthorized, "Parse JWT")
			return
		}
		//TODO: Добавить проверку токена
		fmt.Println(jwtClaims.Sub)

		pathVars := mux.Vars(r)

		reqProfile, err := s.store.Profile().FindProfileByUsername(pathVars["username"])
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusNotFound, "FindByUsername")
			return
		}

		if jwtClaims.Sub == reqProfile.User_ID {
			reqProfile.IsOwnProfile = true
			reqProfile.IsFollowed = true
		} else {
			reqProfile.IsOwnProfile = false
			if bl, _ := s.store.Profile().IsFollow(jwtClaims.Sub, reqProfile.User_ID); bl {
				reqProfile.IsFollowed = true
			} else {
				reqProfile.IsFollowed = false
			}
		}

		byteProfile, err := json.Marshal(reqProfile)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusInternalServerError, "Serialize profile")
			return
		}

		w.Header().Set("Content-Type", "application/json")

		s.apiResult(w, r, "DEBUG", http.StatusOK, "Get profile")
		w.Write(byteProfile)
	}
}

func (s *APIServer) apiGetProfilesByPattern() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jwtClaims, err := parseJWT(r)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusUnauthorized, "Parse JWT")
			return
		}
		//TODO: Добавить проверку токена
		fmt.Println(jwtClaims.Subject)

		pattern := r.URL.Query().Get("search")

		if pattern == "" {
			s.apiResult(w, r, "DEBUG", http.StatusBadRequest, "No pattern")
			return
		}

		reqProfiles, err := s.store.Profile().FindProfileByPattern(pattern)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusNotFound, "FindByPattern")
			return
		}

		byteProfiles, err := json.Marshal(reqProfiles)
		if err != nil {
			s.apiResult(w, r, "DEBUG", http.StatusInternalServerError, "Serialize profiles")
			return
		}

		w.Header().Set("Content-Type", "application/json")

		s.apiResult(w, r, "DEBUG", http.StatusOK, "Get profiles by pattern")
		w.Write(byteProfiles)
	}
}
