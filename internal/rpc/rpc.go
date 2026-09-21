package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	URL  string
	HTTP *http.Client
}

func New(url string) *Client {
	return &Client{URL: strings.TrimRight(url, "/"), HTTP: &http.Client{Timeout: 10 * time.Second}}
}

type request struct {
	Version string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

func (c *Client) Call(ctx context.Context, method string, params []any) (json.RawMessage, error) {
	body, err := json.Marshal(request{Version: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.URL, err)
	}
	defer resp.Body.Close()

	var r response
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("%s: decode %s response: %w", c.URL, method, err)
	}
	if r.Error != nil {
		return nil, fmt.Errorf("%s: %s (code %d)", method, r.Error.Message, r.Error.Code)
	}
	return r.Result, nil
}

func (c *Client) ClientVersion(ctx context.Context) (string, error) {
	res, err := c.Call(ctx, "web3_clientVersion", nil)
	if err != nil {
		return "", err
	}
	return strings.Trim(string(res), `"`), nil
}

func (c *Client) ChainID(ctx context.Context) (uint64, error) {
	res, err := c.Call(ctx, "eth_chainId", nil)
	if err != nil {
		return 0, err
	}
	return hexUint(res)
}

func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	res, err := c.Call(ctx, "eth_blockNumber", nil)
	if err != nil {
		return 0, err
	}
	return hexUint(res)
}

func (c *Client) PeerCount(ctx context.Context) (uint64, error) {
	res, err := c.Call(ctx, "net_peerCount", nil)
	if err != nil {
		return 0, err
	}
	return hexUint(res)
}

func (c *Client) Accounts(ctx context.Context) ([]string, error) {
	res, err := c.Call(ctx, "eth_accounts", nil)
	if err != nil {
		return nil, err
	}
	var accounts []string
	return accounts, json.Unmarshal(res, &accounts)
}

func (c *Client) SendValue(ctx context.Context, from, to string, wei *big.Int) (string, error) {
	tx := map[string]string{"from": from, "to": to, "value": "0x" + wei.Text(16)}
	res, err := c.Call(ctx, "eth_sendTransaction", []any{tx})
	if err != nil {
		return "", err
	}
	return strings.Trim(string(res), `"`), nil
}

func (c *Client) SetBalance(ctx context.Context, addr string, wei *big.Int) error {
	_, err := c.Call(ctx, "anvil_setBalance", []any{addr, "0x" + wei.Text(16)})
	return err
}

func (c *Client) Receipt(ctx context.Context, txHash string) (map[string]any, error) {
	res, err := c.Call(ctx, "eth_getTransactionReceipt", []any{txHash})
	if err != nil {
		return nil, err
	}
	if string(res) == "null" {
		return nil, nil
	}
	var receipt map[string]any
	return receipt, json.Unmarshal(res, &receipt)
}

func hexUint(raw json.RawMessage) (uint64, error) {
	s := strings.Trim(string(raw), `"`)
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return 0, nil
	}
	return strconv.ParseUint(s, 16, 64)
}

func EthToWei(amount string) (*big.Int, error) {
	whole, frac, _ := strings.Cut(strings.TrimSpace(amount), ".")
	if whole == "" {
		whole = "0"
	}
	if len(frac) > 18 {
		return nil, fmt.Errorf("amount %q has more precision than 1 wei", amount)
	}
	frac += strings.Repeat("0", 18-len(frac))
	wei, ok := new(big.Int).SetString(whole+frac, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount %q", amount)
	}
	return wei, nil
}
