package gateway

import (
	"errors"
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"time"

	"github.com/stellar/gateway/config"
	"github.com/stellar/gateway/db"
	"github.com/stellar/gateway/handlers"
	"github.com/stellar/gateway/horizon"
	"github.com/stellar/gateway/listener"
	"github.com/stellar/gateway/submitter"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web/middleware"
)

type App struct {
	config               config.Config
	entityManager        db.EntityManagerInterface
	horizon              horizon.HorizonInterface
	transactionSubmitter *submitter.TransactionSubmitter
	repository           db.RepositoryInterface
}

// NewApp constructs an new App instance from the provided config.
func NewApp(config config.Config) (app *App, err error) {
	entityManager, err := db.NewEntityManager(config.Database.Type, config.Database.Url)
	if err != nil {
		return
	}
	repository, err := db.NewRepository(config.Database.Type, config.Database.Url)
	if err != nil {
		return
	}

	h := horizon.New(*config.Horizon)

	if config.NetworkPassphrase == "" {
		config.NetworkPassphrase = "Test SDF Network ; September 2015"
	}

	log.Print("Creating and initializing TransactionSubmitter")
	ts := submitter.NewTransactionSubmitter(&h, &entityManager, config.NetworkPassphrase)
	if err != nil {
		return
	}

	log.Print("Initializing Authorizing account")

	if config.Accounts.AuthorizingSeed == nil {
		log.Warning("No accounts.authorizing_seed param. Skipping...")
	} else {
		err = ts.InitAccount(*config.Accounts.AuthorizingSeed)
		if err != nil {
			return
		}
	}

	if config.Accounts.IssuingSeed == nil {
		log.Warning("No accounts.issuing_seed param. Skipping...")
	} else {
		log.Print("Initializing Issuing account")
		err = ts.InitAccount(*config.Accounts.IssuingSeed)
		if err != nil {
			return
		}
	}

	log.Print("TransactionSubmitter created")

	log.Print("Creating and starting PaymentListener")

	if config.Accounts.ReceivingAccountId == nil {
		log.Warning("No accounts.receiving_account_id param. Skipping...")
	} else if config.Hooks.Receive == nil {
		log.Warning("No hooks.receive param. Skipping...")
	} else {
		var paymentListener listener.PaymentListener
		paymentListener, err = listener.NewPaymentListener(&config, &entityManager, &h, &repository, time.Now)
		if err != nil {
			return
		}
		err = paymentListener.Listen()
		if err != nil {
			return
		}

		log.Print("PaymentListener created")
	}

	if len(config.ApiKey) > 0 && len(config.ApiKey) < 15 {
		err = errors.New("api-key have to be at least 15 chars long.")
		return
	}

	app = &App{
		config:               config,
		entityManager:        &entityManager,
		horizon:              &h,
		repository:           &repository,
		transactionSubmitter: &ts,
	}
	return
}

func (a *App) Serve() {
	requestHandlers := &handlers.RequestHandler{
		Config:               &a.config,
		Horizon:              a.horizon,
		TransactionSubmitter: a.transactionSubmitter,
	}

	portString := fmt.Sprintf(":%d", *a.config.Port)
	flag.Set("bind", portString)

	goji.Abandon(middleware.Logger)
	goji.Use(handlers.StripTrailingSlashMiddleware())
	goji.Use(handlers.HeadersMiddleware())
	if a.config.ApiKey != "" {
		goji.Use(handlers.ApiKeyMiddleware(a.config.ApiKey))
	}

	if a.config.Accounts.AuthorizingSeed != nil {
		goji.Post("/authorize", requestHandlers.Authorize)
	} else {
		log.Warning("accounts.authorizing_seed not provided. /authorize endpoint will not be available.")
	}

	if a.config.Accounts.IssuingSeed != nil {
		goji.Post("/send", requestHandlers.Send)
	} else {
		log.Warning("accounts.issuing_seed not provided. /send endpoint will not be available.")
	}

	goji.Post("/payment", requestHandlers.Payment)
	goji.Serve()
}
