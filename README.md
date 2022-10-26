# Orderer

`orderer` is a CLI tool for importing orders into shopify.

## Usage

Download executable from [releases](https://github.com/juliaogris/orderer/releases)
or install with go

	go install github.com/juliaogris/orderer@latest

Try it with:

	orderer --version
	orderer --store julias-delights --token ${SHOPIFY_TOKEN} --input testdata/order1.json

Request `${SHOPIFY_TOKEN}` from Julia or use your own store, admin token and product ID.

[releases]: https://github.com/juliaogris/orderer/releases

## Development

Tooling (go, golangci-lint, goreleaser, make) is automatically
bootstrapped with [hermit]. Clone this repo and run `./bin/make` for
available targets. `./bin/make ci` is run on GitHub actions.

[hermit]: https://cashapp.github.io/hermit/