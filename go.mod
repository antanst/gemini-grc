module gemini-grc

go 1.24.3

require (
	git.antanst.com/antanst/logging v0.0.1
	git.antanst.com/antanst/uid v0.0.1
	git.antanst.com/antanst/xerrors v0.0.2
	github.com/guregu/null/v5 v5.0.0
	github.com/jackc/pgx/v5 v5.7.2
	github.com/jmoiron/sqlx v1.4.0
	github.com/lib/pq v1.10.9
	github.com/stretchr/testify v1.9.0
	golang.org/x/text v0.21.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace git.antanst.com/antanst/xerrors => ../xerrors

replace git.antanst.com/antanst/uid => ../uid

replace git.antanst.com/antanst/logging => ../logging
