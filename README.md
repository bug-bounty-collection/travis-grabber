# Travis Grabber
![Docker Image CI](https://github.com/RiRa12621/travis-grabber/workflows/Docker%20Image%20CI/badge.svg)
![Go](https://github.com/RiRa12621/travis-grabber/workflows/Go/badge.svg)
![Travis-Grabber](images/travis-grabber-gopher.png)
## Purpose
Grabs all logs for all builds for any given Organisation from Travis CI

## Installation
It's as easy as 
`go get -u github.com/RiRa12621/travis-grabber`

## Usage

  -github-token string

        GitHub oAuth token used for authentication with GitHub to not instantly get rate limited

  -org string

        the org to scan (this is case sensitive)

  -travis-token string

	Travis auth token you can get from https://travis-ci.org/account/preferences
	
## Contributing

* Create an issue describing your bug or feature request
* Fork it
* Commit your change
* Submit a PR


## ToDo

- [ ] Scan logs for Regex
