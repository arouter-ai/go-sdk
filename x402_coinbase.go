package arouter

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"

	x402sdk "github.com/coinbase/x402/go"
	x402http "github.com/coinbase/x402/go/http"
	evmclient "github.com/coinbase/x402/go/mechanisms/evm/exact/client"
	svmclient "github.com/coinbase/x402/go/mechanisms/svm/exact/client"
	evmsigners "github.com/coinbase/x402/go/signers/evm"
	svmsigners "github.com/coinbase/x402/go/signers/svm"
)

// WithX402CoinbasePayment configures the client with Coinbase's official x402 SDK
// for automatic on-chain USDC payment on EVM networks (Base, Ethereum, etc.).
//
// On the first request the gateway returns 402, the x402 SDK signs payment
// and retries. The response includes an X-API-Key header which is cached
// and used as Bearer token for all subsequent requests.
//
//	key, _ := crypto.HexToECDSA("your-private-key-hex")
//	client := arouter.NewClient(baseURL, "",
//	    arouter.WithX402CoinbasePayment(key),
//	)
func WithX402CoinbasePayment(privateKey *ecdsa.PrivateKey) Option {
	return func(c *Client) {
		keyHex := hex.EncodeToString(crypto.FromECDSA(privateKey))
		evmSigner, err := evmsigners.NewClientSignerFromPrivateKey(keyHex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "arouter: WARNING: x402 EVM payment signer init failed: %v\n", err)
			return
		}

		x402Client := x402sdk.Newx402Client().
			Register("eip155:*", evmclient.NewExactEvmScheme(evmSigner, nil))

		c.httpClient = x402http.WrapHTTPClientWithPayment(
			c.httpClient,
			x402http.Newx402HTTPClient(x402Client),
		)
		wrapWithAPIKeyCache(c)
	}
}

// WithX402CoinbasePaymentFromHex is a convenience wrapper accepting a hex private key string.
func WithX402CoinbasePaymentFromHex(hexKey string) Option {
	return func(c *Client) {
		hexKey = strings.TrimPrefix(hexKey, "0x")
		key, err := crypto.HexToECDSA(hexKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "arouter: WARNING: invalid EVM private key: %v\n", err)
			return
		}
		WithX402CoinbasePayment(key)(c)
	}
}

// WithX402SolanaPayment configures the client with Coinbase's x402 SDK
// for automatic on-chain SPL token payment on Solana networks.
//
//	solKey := ed25519.NewKeyFromSeed(seed)
//	client := arouter.NewClient(baseURL, "",
//	    arouter.WithX402SolanaPayment(solKey),
//	)
func WithX402SolanaPayment(privateKey ed25519.PrivateKey) Option {
	return func(c *Client) {
		b58Key := base58.Encode(privateKey)
		svmSigner, err := svmsigners.NewClientSignerFromPrivateKey(b58Key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "arouter: WARNING: x402 Solana payment signer init failed: %v\n", err)
			return
		}

		x402Client := x402sdk.Newx402Client().
			Register("solana:*", svmclient.NewExactSvmScheme(svmSigner, nil))

		c.httpClient = x402http.WrapHTTPClientWithPayment(
			c.httpClient,
			x402http.Newx402HTTPClient(x402Client),
		)
		wrapWithAPIKeyCache(c)
	}
}

// WithX402DualChainPayment configures both EVM and Solana x402 payment in one call.
func WithX402DualChainPayment(evmKey *ecdsa.PrivateKey, solKey ed25519.PrivateKey) Option {
	return func(c *Client) {
		keyHex := hex.EncodeToString(crypto.FromECDSA(evmKey))
		evmSigner, err := evmsigners.NewClientSignerFromPrivateKey(keyHex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "arouter: WARNING: x402 EVM signer init failed: %v\n", err)
			return
		}

		b58Key := base58.Encode(solKey)
		svmSigner, err := svmsigners.NewClientSignerFromPrivateKey(b58Key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "arouter: WARNING: x402 Solana signer init failed: %v\n", err)
			return
		}

		x402Client := x402sdk.Newx402Client().
			Register("eip155:*", evmclient.NewExactEvmScheme(evmSigner, nil)).
			Register("solana:*", svmclient.NewExactSvmScheme(svmSigner, nil))

		c.httpClient = x402http.WrapHTTPClientWithPayment(
			c.httpClient,
			x402http.Newx402HTTPClient(x402Client),
		)
		wrapWithAPIKeyCache(c)
	}
}

// wrapWithAPIKeyCache wraps the client's HTTP transport to cache the X-API-Key
// header from x402 payment responses and inject it as Bearer token on subsequent requests.
func wrapWithAPIKeyCache(c *Client) {
	base := c.httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	c.httpClient.Transport = &apiKeyCachingTransport{base: base}
}

type apiKeyCachingTransport struct {
	base   http.RoundTripper
	apiKey string
	mu     sync.Mutex
}

func (t *apiKeyCachingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	key := t.apiKey
	t.mu.Unlock()

	if key != "" {
		auth := req.Header.Get("Authorization")
		if auth == "" || auth == "Bearer" || auth == "Bearer " {
			req = req.Clone(req.Context())
			req.Header.Set("Authorization", "Bearer "+key)
		}
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if newKey := resp.Header.Get("X-API-Key"); newKey != "" {
		t.mu.Lock()
		t.apiKey = newKey
		t.mu.Unlock()
	}

	return resp, err
}
