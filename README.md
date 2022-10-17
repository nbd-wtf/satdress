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

## Get help

Maybe ask for help on https://t.me/lnurl if you're in trouble.
