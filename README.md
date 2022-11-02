# Orderer

`orderer` is a CLI for importing orders into shopify.

## Usage

Install `orderer` with

	brew tap OfficiallyEQL/tap
	brew install orderer

Alternatively you can download the executable directly from [releases](https://github.com/OfficiallyEQL/orderer/releases) or install with `go install github.com/OfficiallyEQL/orderer@latest`

After installation try

	orderer --version

	export SHOPIFY_STORE=julias-delights
	export SHOPIFY_TOKEN=shpat_6a....5470  # ask Julia for token or setup your own store and admin token.
	
	orderer create testdata/order.json
	orderer list testdata/order.json

[releases]: https://github.com/OfficiallyEQL/orderer/releases

## Development

Tooling (go, golangci-lint, goreleaser, make) is automatically
bootstrapped with [hermit]. Clone this repo and run `./bin/make` for
available targets. `./bin/make ci` is run on GitHub actions.

[hermit]: https://cashapp.github.io/hermit/
