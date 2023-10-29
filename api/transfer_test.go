package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	mockdb "github.com/techschool/simplebank/db/mock"
	db "github.com/techschool/simplebank/db/sqlc"
	"github.com/techschool/simplebank/util"
)

func TestCreateTransfer(t *testing.T) {
	t.Parallel() // Run in parallel

	amount := int64(10)
	currency := util.USD
	fromAccount := randomAccountSettingCurrency(currency)
	toAccount := randomAccountSettingCurrency(currency)
	toAccountCAD := randomAccountSettingCurrency(util.CAD)
	transferTxResult := db.TransferTxResult{
		Transfer:    db.Transfer{},
		FromAccount: fromAccount,
		ToAccount:   toAccount,
		FromEntry:   db.Entry{},
		ToEntry:     db.Entry{},
	}

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, records *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          amount,
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetAccount(gomock.Any(), fromAccount.ID).
					Times(1).
					Return(fromAccount, nil)
				store.EXPECT().
					GetAccount(gomock.Any(), toAccount.ID).
					Times(1).
					Return(toAccount, nil)

				arg := db.TransferTxParams{
					FromAccountID: fromAccount.ID,
					ToAccountID:   toAccount.ID,
					Amount:        amount,
				}

				store.EXPECT().
					TransferTx(gomock.Any(), arg).
					Times(1).
					Return(transferTxResult, nil)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "Invalid from account, to account, amount and currency and owner in request body",
			body: gin.H{
				"from_account_id": 0,
				"to_account_id":   0,
				"amount":          0,
				"currency":        nil,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), fromAccount.ID).
					Times(0)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Invalid from account (account does not exist)",
			body: gin.H{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          amount,
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), fromAccount.ID).
					Times(1).
					Return(db.Account{}, sql.ErrNoRows)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "Invalid to account (account does not exist)",
			body: gin.H{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          amount,
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), fromAccount.ID).
					Times(1).
					Return(fromAccount, nil)
				store.EXPECT().
					GetAccount(gomock.Any(), toAccount.ID).
					Times(1).
					Return(db.Account{}, sql.ErrNoRows)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "Error when reading from account",
			body: gin.H{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          amount,
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), fromAccount.ID).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Error when reading to account",
			body: gin.H{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          amount,
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), fromAccount.ID).
					Times(1).
					Return(fromAccount, nil)
				store.EXPECT().
					GetAccount(gomock.Any(), toAccount.ID).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Error when currency is different of the from account",
			body: gin.H{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          amount,
				"currency":        util.CAD,
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetAccount(gomock.Any(), fromAccount.ID).
					Times(1).
					Return(fromAccount, nil)

				store.EXPECT().
					GetAccount(gomock.Any(), toAccount.ID).
					Times(0)
				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Any()).
					Times(0)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Error when currency is different of the to account",
			body: gin.H{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccountCAD.ID,
				"amount":          amount,
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetAccount(gomock.Any(), fromAccount.ID).
					Times(1).
					Return(fromAccount, nil)
				store.EXPECT().
					GetAccount(gomock.Any(), toAccountCAD.ID).
					Times(1).
					Return(toAccountCAD, nil)

				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Any()).
					Times(0)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Error when creating the transfer at the db",
			body: gin.H{
				"from_account_id": fromAccount.ID,
				"to_account_id":   toAccount.ID,
				"amount":          amount,
				"currency":        currency,
			},
			buildStubs: func(store *mockdb.MockStore) {

				store.EXPECT().
					GetAccount(gomock.Any(), fromAccount.ID).
					Times(1).
					Return(fromAccount, nil)
				store.EXPECT().
					GetAccount(gomock.Any(), toAccount.ID).
					Times(1).
					Return(toAccount, nil)

				arg := db.TransferTxParams{
					FromAccountID: fromAccount.ID,
					ToAccountID:   toAccount.ID,
					Amount:        amount,
				}

				store.EXPECT().
					TransferTx(gomock.Any(), arg).
					Times(1).
					Return(db.TransferTxResult{}, sql.ErrConnDone)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable https://golang.org/pkg/testing/#hdr-Subtests_and_Sub_benchmarks
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)

			tc.buildStubs(store)

			//start test server and send request
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/transfers"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})

	}
}
