package arouter

import (
	"crypto/ecdsa"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	x402sdk "github.com/coinbase/x402/go"
	x402http "github.com/coinbase/x402/go/http"
	evmclient "github.com/coinbase/x402/go/mechanisms/evm/exact/client"
	evmsigners "github.com/coinbase/x402/go/signers/evm"
)

// WithX402CoinbasePayment configures the client with Coinbase's official x402 SDK
// for automatic on-chain USDC payment when credits are insufficient.
//
//	key, _ := crypto.HexToECDSA("your-private-key-hex")
//	client := arouter.NewClient(baseURL, "",
//	    arouter.WithX402CoinbasePayment(key),
//	)
//
// Sets up both wallet authentication and automatic x402 payment.
func WithX402CoinbasePayment(privateKey *ecdsa.PrivateKey) Option {
	return func(c *Client) {
		walletSigner := NewEvmWalletSigner(privateKey)
		WithWalletAuth(walletSigner)(c)

		keyHex := hex.EncodeToString(crypto.FromECDSA(privateKey))
		evmSigner, err := evmsigners.NewClientSignerFromPrivateKey(keyHex)
		if err != nil {
			return
		}

		x402Client := x402sdk.Newx402Client().
			Register("eip155:*", evmclient.NewExactEvmScheme(evmSigner, nil))

		c.httpClient = x402http.WrapHTTPClientWithPayment(
			c.httpClient,
			x402http.Newx402HTTPClient(x402Client),
		)
	}
}

// WithX402CoinbasePaymentFromHex is a convenience wrapper accepting a hex private key string.
//
//	client := arouter.NewClient(baseURL, "",
//	    arouter.WithX402CoinbasePaymentFromHex(os.Getenv("EVM_PRIVATE_KEY")),
//	)
func WithX402CoinbasePaymentFromHex(hexKey string) Option {
	return func(c *Client) {
		hexKey = strings.TrimPrefix(hexKey, "0x")
		key, err := crypto.HexToECDSA(hexKey)
		if err != nil {
			return
		}
		WithX402CoinbasePayment(key)(c)
	}
}

// compile-time check
var _ = func() *http.Client {
	return x402http.WrapHTTPClientWithPayment(nil, nil)
}
