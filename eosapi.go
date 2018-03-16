package eosapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
)

type EOSAPI struct {
	HttpClient *http.Client
	BaseURL    string
}

func New(baseURL string) *EOSAPI {
	return &EOSAPI{
		HttpClient: http.DefaultClient,
		BaseURL:    baseURL,
	}
}

// Chain APIs
// Wallet APIs

// List here: https://github.com/Netherdrake/py-eos-api/blob/master/eosapi/client.py

// const string push_txn_func = chain_func_base + "/push_transaction";
// const string push_txns_func = chain_func_base + "/push_transactions";
// const string json_to_bin_func = chain_func_base + "/abi_json_to_bin";
// const string get_block_func = chain_func_base + "/get_block";
// const string get_account_func = chain_func_base + "/get_account";
// const string get_table_func = chain_func_base + "/get_table_rows";
// const string get_code_func = chain_func_base + "/get_code";
// const string get_currency_balance_func = chain_func_base + "/get_currency_balance";
// const string get_currency_stats_func = chain_func_base + "/get_currency_stats";
// const string get_required_keys = chain_func_base + "/get_required_keys";

// const string account_history_func_base = "/v1/account_history";
// const string get_transaction_func = account_history_func_base + "/get_transaction";
// const string get_transactions_func = account_history_func_base + "/get_transactions";
// const string get_key_accounts_func = account_history_func_base + "/get_key_accounts";
// const string get_controlled_accounts_func = account_history_func_base + "/get_controlled_accounts";

// const string net_func_base = "/v1/net";
// const string net_connect = net_func_base + "/connect";
// const string net_disconnect = net_func_base + "/disconnect";
// const string net_status = net_func_base + "/status";
// const string net_connections = net_func_base + "/connections";

// const string wallet_func_base = "/v1/wallet";
// const string wallet_create = wallet_func_base + "/create";
// const string wallet_open = wallet_func_base + "/open";
// const string wallet_list = wallet_func_base + "/list_wallets";
// const string wallet_list_keys = wallet_func_base + "/list_keys";
// const string wallet_public_keys = wallet_func_base + "/get_public_keys";
// const string wallet_lock = wallet_func_base + "/lock";
// const string wallet_lock_all = wallet_func_base + "/lock_all";
// const string wallet_unlock = wallet_func_base + "/unlock";
// const string wallet_import_key = wallet_func_base + "/import_key";
// const string wallet_sign_trx = wallet_func_base + "/sign_transaction";

func (api *EOSAPI) GetAccount(name AccountName) (out *AccountResp, err error) {
	err = api.call("chain", "get_account", M{"account_name": name}, &out)
	return
}

func (api *EOSAPI) GetCode(account AccountName) (out *Code, err error) {
	err = api.call("chain", "get_code", M{"account_name": account}, &out)
	return
}

func (api *EOSAPI) PushTransaction(tx *Transaction) (out *PushTransactionResp, err error) {
	if err := tx.Fill(api); err != nil {
		return nil, err
	}

	err = api.call("chain", "push_transaction", M{"transaction": tx}, &out)
	return
}

func (api *EOSAPI) PushSignedTransaction(tx *SignedTransaction) (out *PushTransactionResp, err error) {
	err = api.call("chain", "push_transaction", M{"transaction": tx.Transaction, "signatures": tx.Signatures, "compression": true}, &out)
	return
}

// func (api *EOSAPI) NewAccount(creator, newAccount AccountName, owner, active, recovery Authority) (out *NewAccountResp, err error) {
// 	tx := &Transaction{
// 		Actions: []*Action{
// 			{
// 				Account,
// 			},
// 		},
// 	}
// }

func (api *EOSAPI) SetCode(account AccountName, wastPath, abiPath string, keybag *KeyBag) (out *PushTransactionResp, err error) {
	// SetCode will create a transaction, call GetRequiredKeys, and sign the transaction with keybag.AvailableKeys().

	tx := &Transaction{
		Actions: []*Action{
			{
				Account: AccountName("eosio"),
				Name:    ActionName("transfer"),
				Authorization: []PermissionLevel{
					{AccountName("eosio"), PermissionName("active")},
				},
			},
		},
	}
	if err := tx.Fill(api); err != nil {
		return nil, err
	}

	resp, err := api.GetRequiredKeys(tx, keybag)
	log.Println("GetRequiredKeys", resp, err)
	if err != nil {
		return nil, err
	}

	signed, err := keybag.Sign(tx, resp.RequiredKeys...)
	if err != nil {
		return nil, err
	}

	return api.PushSignedTransaction(signed)
}

func (api *EOSAPI) GetInfo() (out *InfoResp, err error) {
	err = api.call("chain", "get_info", nil, &out)
	return
}

func (api *EOSAPI) GetBlockByID(id string) (out *BlockResp, err error) {
	err = api.call("chain", "get_block", M{"block_num_or_id": id}, &out)
	return
}

func (api *EOSAPI) GetBlockByNum(num uint64) (out *BlockResp, err error) {
	err = api.call("chain", "get_block", M{"block_num_or_id": fmt.Sprintf("%d", num)}, &out)
	return
}

func (api *EOSAPI) GetTableRows(params GetTableRowsRequest) (out *GetTableRowsResp, err error) {
	err = api.call("chain", "get_table_rows", params, &out)
	return
}

func (api *EOSAPI) GetRequiredKeys(tx *Transaction, keybag *KeyBag) (out *GetRequiredKeysResp, err error) {
	err = api.call("chain", "get_required_keys", M{"transaction": tx, "available_keys": keybag.AvailableKeys()}, &out)
	return
}

func (api *EOSAPI) GetCurrencyBalance(account AccountName, symbol string, code AccountName) (out []Asset, err error) {
	err = api.call("chain", "get_currency_balance", M{"account": account, "symbol": symbol, "code": code}, &out)
	return
}

// See more here: libraries/chain/contracts/abi_serializer.cpp:58...

func (api *EOSAPI) call(baseAPI string, endpoint string, body interface{}, out interface{}) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/%s/%s", api.BaseURL, baseAPI, endpoint), enc(body))
	if err != nil {
		return fmt.Errorf("NewRequest: %s", err)
	}

	// Useful when debugging API calls
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))

	resp, err := api.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Do: %s", err)
	}
	defer resp.Body.Close()

	var cnt bytes.Buffer
	_, err = io.Copy(&cnt, resp.Body)
	if err != nil {
		return fmt.Errorf("Copy: %s", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("status code = %d, body=%q", resp.StatusCode, cnt.String())
	}

	//fmt.Println("STRING", cnt.String())

	if err := json.Unmarshal(cnt.Bytes(), &out); err != nil {
		return fmt.Errorf("Unmarshal: %s", err)
	}

	return nil
}

type M map[string]interface{}

func enc(v interface{}) io.Reader {
	if v == nil {
		return nil
	}

	cnt, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	//fmt.Println("BODY", string(cnt))

	return bytes.NewReader(cnt)
}
