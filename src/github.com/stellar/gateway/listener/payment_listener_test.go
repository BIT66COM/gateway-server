package listener

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stellar/gateway/config"
	"github.com/stellar/gateway/db"
	"github.com/stellar/gateway/horizon"
	"github.com/stellar/gateway/mocks"
	"github.com/stretchr/testify/assert"
)

func TestPaymentListener(t *testing.T) {
	mockEntityManager := new(mocks.MockEntityManager)
	mockHorizon := new(mocks.MockHorizon)
	mockRepository := new(mocks.MockRepository)

	var receiveHookStatusCode int

	receiveHookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "1", r.FormValue("id"))
		assert.Equal(t, "GBIHSMPXC2KJ3NJVHEYTG3KCHYEUQRT45X6AWYWXMAXZOAX4F5LFZYYQ", r.FormValue("from"))
		assert.Equal(t, "200", r.FormValue("amount"))
		assert.Equal(t, "USD", r.FormValue("asset_code"))
		assert.Equal(t, "text", r.FormValue("memo_type"))
		assert.Equal(t, "testing", r.FormValue("memo"))
		w.WriteHeader(receiveHookStatusCode)
		fmt.Fprintln(w, "Response")
	}))
	defer receiveHookServer.Close()

	IssuingSeed := "SC34WILLHVADXMP6ACPMIRA6TRAWJMVCLPFNW7S6MUMXJVLAZUC4EWHP"
	ReceivingAccountId := "GATKP6ZQM5CSLECPMTAC5226PE367QALCPM6AFHTSULPPZMT62OOPMQB"

	config := &config.Config{
		Assets: []string{"USD", "EUR"},
		Accounts: &config.Accounts{
			// GD4I7AFSLZGTDL34TQLWJOM2NHLIIOEKD5RHHZUW54HERBLSIRKUOXRR
			IssuingSeed:        &IssuingSeed,
			ReceivingAccountId: &ReceivingAccountId,
		},
		Hooks: &config.Hooks{
			Receive: &receiveHookServer.URL,
		},
	}

	paymentListener, _ := NewPaymentListener(
		config,
		mockEntityManager,
		mockHorizon,
		mockRepository,
		mocks.Now,
	)

	Convey("PaymentListener", t, func() {
		operation := horizon.PaymentResponse{
			Id:          "1",
			From:        "GBIHSMPXC2KJ3NJVHEYTG3KCHYEUQRT45X6AWYWXMAXZOAX4F5LFZYYQ",
			PagingToken: "2",
			Amount:      "200",
		}

		mocks.PredefinedTime = time.Now()

		dbPayment := db.ReceivedPayment{
			OperationId: operation.Id,
			ProcessedAt: mocks.PredefinedTime,
			PagingToken: operation.PagingToken,
		}

		Convey("When operation is not a payment", func() {
			operation.Type = "create_account"
			dbPayment.Status = "Not a payment operation"
			mockEntityManager.On("Persist", &dbPayment).Return(nil).Once()

			Convey("it should save the status", func() {
				err := paymentListener.onPayment(operation)
				assert.Nil(t, err)
				mockEntityManager.AssertExpectations(t)
			})
		})

		Convey("When payment is sent not received", func() {
			operation.Type = "payment"
			operation.To = "GDNXBMIJLLLXZYKZBHXJ45WQ4AJQBRVT776YKGQTDBHTSPMNAFO3OZOS"
			dbPayment.Status = "Operation sent not received"
			mockEntityManager.On("Persist", &dbPayment).Return(nil).Once()

			Convey("it should save the status", func() {
				err := paymentListener.onPayment(operation)
				assert.Nil(t, err)
				mockEntityManager.AssertExpectations(t)
			})
		})

		Convey("When asset is not allowed (issuer)", func() {
			operation.Type = "payment"
			operation.To = "GATKP6ZQM5CSLECPMTAC5226PE367QALCPM6AFHTSULPPZMT62OOPMQB"
			operation.AssetCode = "USD"
			operation.AssetIssuer = "GC4WWLMUGZJMRVJM7JUVVZBY3LJ5HL4RKIPADEGKEMLAAJEDRONUGYG7"
			dbPayment.Status = "Asset not allowed"
			mockEntityManager.On("Persist", &dbPayment).Return(nil).Once()

			Convey("it should save the status", func() {
				err := paymentListener.onPayment(operation)
				assert.Nil(t, err)
				mockEntityManager.AssertExpectations(t)
			})
		})

		Convey("When asset is not allowed (code)", func() {
			operation.Type = "payment"
			operation.To = "GATKP6ZQM5CSLECPMTAC5226PE367QALCPM6AFHTSULPPZMT62OOPMQB"
			operation.AssetCode = "GBP"
			operation.AssetIssuer = "GD4I7AFSLZGTDL34TQLWJOM2NHLIIOEKD5RHHZUW54HERBLSIRKUOXRR"
			dbPayment.Status = "Asset not allowed"
			mockEntityManager.On("Persist", &dbPayment).Return(nil).Once()

			Convey("it should save the status", func() {
				err := paymentListener.onPayment(operation)
				assert.Nil(t, err)
				mockEntityManager.AssertExpectations(t)
			})
		})

		Convey("When transaction does not have memo", func() {
			operation.Type = "payment"
			operation.To = "GATKP6ZQM5CSLECPMTAC5226PE367QALCPM6AFHTSULPPZMT62OOPMQB"
			operation.AssetCode = "USD"
			operation.AssetIssuer = "GD4I7AFSLZGTDL34TQLWJOM2NHLIIOEKD5RHHZUW54HERBLSIRKUOXRR"
			dbPayment.Status = "Transaction does not have memo"

			mockHorizon.On("LoadMemo", &operation).Return(nil).Once()
			mockEntityManager.On("Persist", &dbPayment).Return(nil).Once()

			Convey("it should save the status", func() {
				err := paymentListener.onPayment(operation)
				assert.Nil(t, err)
				mockHorizon.AssertExpectations(t)
				mockEntityManager.AssertExpectations(t)
			})
		})

		Convey("When unable to load transaction memo", func() {
			operation.Type = "payment"
			operation.To = "GATKP6ZQM5CSLECPMTAC5226PE367QALCPM6AFHTSULPPZMT62OOPMQB"
			operation.AssetCode = "USD"
			operation.AssetIssuer = "GD4I7AFSLZGTDL34TQLWJOM2NHLIIOEKD5RHHZUW54HERBLSIRKUOXRR"

			mockHorizon.On("LoadMemo", &operation).Return(errors.New("Connection error")).Once()

			Convey("it should return error", func() {
				err := paymentListener.onPayment(operation)
				assert.Error(t, err)
				mockHorizon.AssertExpectations(t)
				mockEntityManager.AssertNotCalled(t, "Persist")
			})
		})

		Convey("When receive hook returns error", func() {
			operation.Type = "payment"
			operation.To = "GATKP6ZQM5CSLECPMTAC5226PE367QALCPM6AFHTSULPPZMT62OOPMQB"
			operation.AssetCode = "USD"
			operation.AssetIssuer = "GD4I7AFSLZGTDL34TQLWJOM2NHLIIOEKD5RHHZUW54HERBLSIRKUOXRR"
			operation.Memo.Type = "text"
			operation.Memo.Value = "testing"

			mockHorizon.On("LoadMemo", &operation).Return(nil).Once()
			receiveHookStatusCode = 503

			Convey("it should save the status", func() {
				err := paymentListener.onPayment(operation)
				assert.Error(t, err)
				mockHorizon.AssertExpectations(t)
				mockEntityManager.AssertNotCalled(t, "Persist")
			})
		})

		Convey("When receive hook returns success", func() {
			operation.Type = "payment"
			operation.To = "GATKP6ZQM5CSLECPMTAC5226PE367QALCPM6AFHTSULPPZMT62OOPMQB"
			operation.AssetCode = "USD"
			operation.AssetIssuer = "GD4I7AFSLZGTDL34TQLWJOM2NHLIIOEKD5RHHZUW54HERBLSIRKUOXRR"
			operation.Memo.Type = "text"
			operation.Memo.Value = "testing"

			dbPayment.Status = "Success"

			mockHorizon.On("LoadMemo", &operation).Return(nil).Once()
			mockEntityManager.On("Persist", &dbPayment).Return(nil).Once()
			receiveHookStatusCode = 200

			Convey("it should save the status", func() {
				err := paymentListener.onPayment(operation)
				assert.Nil(t, err)
				mockHorizon.AssertExpectations(t)
				mockEntityManager.AssertExpectations(t)
			})
		})
	})
}
