module github.com/omnichannel/auth_service

go 1.26.4

require (
	github.com/aarondl/authboss/v3 v3.5.3
	github.com/go-chi/chi/v5 v5.3.2
	github.com/go-chi/cors v1.2.2
	github.com/go-chi/httprate v0.16.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/golang-migrate/migrate/v4 v4.17.1
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/omnichannel/common v0.0.0-00010101000000-000000000000
	github.com/redis/go-redis/v9 v9.22.0
	github.com/sqlc-dev/pqtype v0.3.0
	go.uber.org/zap v1.28.0
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/friendsofgo/errors v0.9.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)

replace github.com/omnichannel/common => ../common
