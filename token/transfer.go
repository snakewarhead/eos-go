package token

import eos "github.com/snakewarhead/eos-go"

func NewTransfer(from, to eos.AccountName, quantity eos.Asset, memo string) *eos.Action {
	return NewTransferCommon(AN("eosio.token"), from, to, quantity, memo)
}

func NewTransferCommon(contract, from, to eos.AccountName, quantity eos.Asset, memo string) *eos.Action {
	return &eos.Action{
		Account: contract,
		Name:    ActN("transfer"),
		Authorization: []eos.PermissionLevel{
			{Actor: from, Permission: PN("active")},
		},
		ActionData: eos.NewActionData(Transfer{
			From:     from,
			To:       to,
			Quantity: quantity,
			Memo:     memo,
		}),
	}
}

// Transfer represents the `transfer` struct on `eosio.token` contract.
type Transfer struct {
	From     eos.AccountName `json:"from"`
	To       eos.AccountName `json:"to"`
	Quantity eos.Asset       `json:"quantity"`
	Memo     string          `json:"memo"`
}
