<a href="https://nbd.wtf"><img align="right" height="196" src="https://user-images.githubusercontent.com/1653275/194609043-0add674b-dd40-41ed-986c-ab4a2e053092.png" /></a>

# Satdress

Federated Lightning Address Server

## How to run

1. Download the binary from the releases page (or compile with `go build` or `go get`)
2. Set the following environment variables somehow (using example values from bitmia.com):

```
PORT=17422
DOMAIN=bitmia.com
SECRET=askdbasjdhvakjvsdjasd
SITE_OWNER_URL=https://t.me/qecez
SITE_OWNER_NAME=@qecez
SITE_NAME=Bitmia
```

3. Start the app with `./satdress`
4. Serve the app to the world on your domain using whatever technique you're used to

## Multiple domains

Note that `DOMAIN` can be a single domain or a comma-separated list. When using multiple domains
you need to make sure "Host" HTTP header is forwarded to satdress process if you have some reverse-proxy).

If you come from an old installation everything should get migrated in a seamless way, but there is also a
`FORCE_MIGRATE` environment variable to force a migration (else this is done just the first time).

There is also a `GLOBAL_USERS` to make sure the user@ part is unique across all domains. But be warned that when enabling
this option, existing users won't work anymore (which is by design).

## Nostr zaps (NIP-57)

Set `NOSTR_PRIVATE_KEY` (a 64-character hex secret key, used only to sign zap receipts) to enable
[NIP-57](https://github.com/nostr-protocol/nips/blob/master/57.md) zaps:

```
NOSTR_PRIVATE_KEY=<64 hex characters, e.g. from `openssl rand -hex 32`>
ZAP_RELAYS=wss://relay.damus.io,wss://nos.lol
```

- The lnurl-pay response then carries `allowsNostr` and `nostrPubkey` for addresses whose backend
  satdress can ask about payments: **LND** (the invoice macaroon can look invoices up) and **LNbits**
  (the invoice key). Other backends keep receiving regular payments.
- A zap request (`nostr` parameter, kind 9734) is validated as NIP-57 says and its hash becomes the
  invoice's `description_hash`.
- satdress then polls the owner's node for that invoice (every 3 s for the first minute, then every
  20 s, for up to an hour) and, once paid, publishes the signed receipt (kind 9735) to the relays in
  the zap request plus the optional `ZAP_RELAYS`. `.onion` hosts go through `TOR_PROXY_URL`, as invoices do.
- The server never touches the sats. Pending zaps are kept in memory, so restarting in the middle of
  a zap loses that receipt (the payment itself still reaches the owner).

Without `NOSTR_PRIVATE_KEY` nothing changes.

## Get help

Maybe ask for help on https://t.me/lnurl if you're in trouble.
