# storage-service
Start with the following command
```
docker-compose up
```

## Database
The command above will start a postgres db and a go service serving port 8080. The go service will use the postgres db to save username password (hashed with bcrypt), and save the list of user's files together with the Content-Type and Content-Length of the file. The driver used is `lib/pq`

## Routing
Routing is handled with `gorilla/mux`. The `router.go` file loops all routes listed in `routes.go` and register them.

## Logging
`access-logger.go` contains a wrapper that will log access to any route. `router.go` wrap all routes with it so all access are logged.

## Session
Session is managed by `gorilla/sessions`, which uses cookies so browsers will automatically have session attached as cookies when make request after user logged in. Also `X-Token` header is also supported to be used to specify the session.

## Storage
Files are simply stored under the location `files/{username}/{filename}`

## Future improvements
1. The storage layer could be abstracted to support other storage types like S3, HDFS, etc.
2. Some code in the handlers could be refactored better. Maybe have middlewares and chain them.
3. Logging could add more details.
