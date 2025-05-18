FROM            golang:1.23-alpine AS build
WORKDIR         /app
RUN             apk add file protoc protobuf-dev build-base git grpc
COPY            . .
RUN             make tidy
RUN             make

FROM            alpine:3.20 AS app
ARG             COMMAND=bot:serve
ENV             COMMAND=$COMMAND
COPY            --from=build /app/bin/* /bin/
VOLUME          /app
CMD             app $COMMAND
