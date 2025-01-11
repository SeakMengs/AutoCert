package main

import (
	appcontext "github.com/SeakMengs/AutoCert/internal/app_context"
	"github.com/SeakMengs/AutoCert/internal/auth"
	"github.com/SeakMengs/AutoCert/internal/config"
	"github.com/SeakMengs/AutoCert/internal/controller"
	"github.com/SeakMengs/AutoCert/internal/database"
	"github.com/SeakMengs/AutoCert/internal/env"
	"github.com/SeakMengs/AutoCert/internal/mailer"
	"github.com/SeakMengs/AutoCert/internal/middleware"
	ratelimiter "github.com/SeakMengs/AutoCert/internal/rate_limiter"
	"github.com/SeakMengs/AutoCert/internal/repository"
	"github.com/SeakMengs/AutoCert/internal/route"
	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// this function run before main
func init() {
	env.LoadEnv(".env")
}

func main() {
	cfg := config.GetConfig()

	logger := util.NewLogger()
	logger.Infof("Configuration: %+v \n", cfg)

	db, err := database.ConnectReturnGormDB(cfg.DB)
	if err != nil {
		logger.Panic(err)
	}

	sqlDb, err := db.DB()
	if err != nil {
		logger.Panic(err)
	}
	defer sqlDb.Close()
	logger.Info("Database connected \n")

	rateLimiter := ratelimiter.NewRateLimiter(cfg.RateLimiter, logger)
	mail := mailer.NewSendgrid(cfg.Mail.SEND_GRID.API_KEY, cfg.Mail.FROM_EMAIL, cfg.IsProduction(), logger)
	jwtService := auth.NewJwt(cfg.Auth,
		logger)
	repo := repository.NewRepository(db, logger, jwtService)
	app := appcontext.Application{
		Config:     &cfg,
		Repository: repo,
		Logger:     logger,
		Mailer:     mail,
		JWTService: jwtService,
	}

	_middleware := middleware.NewMiddleware(app.Logger, rateLimiter)
	r := gin.Default()

	// docs: https://github.com/gin-contrib/cors?tab=readme-ov-file#using-defaultconfig-as-start-point
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	r.Use(cors.New(corsConfig))
	r.Use(_middleware.RateLimiterMiddleware)

	_controller := controller.NewController(&app)

	r.GET("/", _controller.Index.Index)

	rApi := r.Group("/api")

	route.V1_Users(rApi, _controller.User)
	route.V1_Auth(rApi, _controller.Auth)
	route.V1_OAuth(rApi, _controller.OAuth)

	if err := r.Run("0.0.0.0:" + app.Config.Port); err != nil {
		logger.Panic("Error running server: %v \n", err)
	}
}
