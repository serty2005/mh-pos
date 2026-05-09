module cloud-backend

go 1.26.2

require (
	github.com/JohannesJHN/iso4217 v0.0.0-20250910211824-d9ba0fe363a8
	github.com/go-chi/chi/v5 v5.2.5
	github.com/jackc/pgx/v5 v5.7.6
	mh-pos-platform v0.0.0
)

replace mh-pos-platform => ../shared/platform

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/text v0.24.0 // indirect
)
